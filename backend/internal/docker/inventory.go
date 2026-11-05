package docker

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/volume"
)

// ChangeKind describes an inventory change.
type ChangeKind string

const (
	ChangeUpsert ChangeKind = "upsert"
	ChangeRemove ChangeKind = "remove"
)

type ContainerChange struct {
	Kind      ChangeKind `json:"kind"`
	ID        string     `json:"id"`
	Container *Container `json:"container,omitempty"`
}

// Inventory keeps an in-memory, event-driven picture of all Docker resources.
// It never modifies containers; it only reads.
type Inventory struct {
	gw  *Gateway
	log *slog.Logger

	mu         sync.RWMutex
	containers map[string]*Container
	synced     bool
	syncErr    string
	lastSync   time.Time

	resMu      sync.RWMutex
	networks   []network.Summary
	images     []image.Summary
	volumes    []volume.Volume
	resAt      map[string]time.Time
	resErr     map[string]string

	lisMu     sync.RWMutex
	listeners []func(ContainerChange)
}

func NewInventory(gw *Gateway, log *slog.Logger) *Inventory {
	return &Inventory{
		gw: gw, log: log,
		containers: map[string]*Container{},
		resAt:      map[string]time.Time{},
		resErr:     map[string]string{},
	}
}

func (inv *Inventory) OnChange(f func(ContainerChange)) {
	inv.lisMu.Lock()
	inv.listeners = append(inv.listeners, f)
	inv.lisMu.Unlock()
}

func (inv *Inventory) emit(ch ContainerChange) {
	inv.lisMu.RLock()
	ls := inv.listeners
	inv.lisMu.RUnlock()
	for _, f := range ls {
		f(ch)
	}
}

// Synced reports whether at least one full sync succeeded, plus the last error.
func (inv *Inventory) Synced() (bool, string, time.Time) {
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	return inv.synced, inv.syncErr, inv.lastSync
}

// Sync performs a full reconciliation (list + inspect of every container).
func (inv *Inventory) Sync(ctx context.Context) error {
	list, err := inv.gw.ListContainers(ctx)
	if err != nil {
		inv.mu.Lock()
		inv.syncErr = err.Error()
		inv.mu.Unlock()
		return err
	}
	type res struct {
		c   *Container
		err error
		id  string
	}
	sem := make(chan struct{}, 8)
	out := make(chan res, len(list))
	var wg sync.WaitGroup
	for _, s := range list {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			in, _, err := inv.gw.InspectContainer(ctx, id)
			if err != nil {
				out <- res{err: err, id: id}
				return
			}
			c := FromInspect(in)
			out <- res{c: &c, id: id}
		}(s.ID)
	}
	wg.Wait()
	close(out)

	fresh := map[string]*Container{}
	for r := range out {
		if r.err != nil {
			if errors.Is(r.err, ErrNotFound) {
				continue // removed between list and inspect
			}
			inv.log.Warn("inspect failed during sync", "id", r.id, "err", r.err)
			inv.mu.RLock()
			if old, ok := inv.containers[r.id]; ok {
				fresh[r.id] = old
			}
			inv.mu.RUnlock()
			continue
		}
		fresh[r.c.ID] = r.c
	}

	inv.mu.Lock()
	old := inv.containers
	inv.containers = fresh
	inv.synced = true
	inv.syncErr = ""
	inv.lastSync = time.Now()
	inv.mu.Unlock()

	for id, c := range fresh {
		if prev, ok := old[id]; !ok || !sameContainer(prev, c) {
			inv.emit(ContainerChange{Kind: ChangeUpsert, ID: id, Container: c})
		}
	}
	for id := range old {
		if _, ok := fresh[id]; !ok {
			inv.emit(ContainerChange{Kind: ChangeRemove, ID: id})
		}
	}
	return nil
}

func sameContainer(a, b *Container) bool {
	return a.State == b.State && a.Health == b.Health && a.RestartCount == b.RestartCount &&
		a.Name == b.Name && a.ImageID == b.ImageID && len(a.Networks) == len(b.Networks) &&
		timeEq(a.StartedAt, b.StartedAt) && a.Pid == b.Pid && networksEq(a.Networks, b.Networks)
}

func networksEq(a, b []ContainerNet) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].NetworkID != b[i].NetworkID || a[i].IPAddress != b[i].IPAddress {
			return false
		}
	}
	return true
}

func timeEq(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Equal(*b)
}

// Refresh re-inspects a single container (called from the event stream).
func (inv *Inventory) Refresh(ctx context.Context, id string) {
	in, _, err := inv.gw.InspectContainer(ctx, id)
	if errors.Is(err, ErrNotFound) {
		inv.mu.Lock()
		_, existed := inv.containers[id]
		delete(inv.containers, id)
		inv.mu.Unlock()
		if existed {
			inv.emit(ContainerChange{Kind: ChangeRemove, ID: id})
		}
		return
	}
	if err != nil {
		inv.log.Warn("refresh container failed", "id", id, "err", err)
		return
	}
	c := FromInspect(in)
	inv.mu.Lock()
	inv.containers[c.ID] = &c
	inv.mu.Unlock()
	inv.emit(ContainerChange{Kind: ChangeUpsert, ID: c.ID, Container: &c})
}

// Containers returns a sorted snapshot copy.
func (inv *Inventory) Containers() []Container {
	inv.mu.RLock()
	out := make([]Container, 0, len(inv.containers))
	for _, c := range inv.containers {
		out = append(out, *c)
	}
	inv.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Get finds a container by full id, unique id prefix or name.
func (inv *Inventory) Get(ref string) (Container, bool) {
	ref = strings.TrimPrefix(ref, "/")
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	if c, ok := inv.containers[ref]; ok {
		return *c, true
	}
	var found *Container
	for _, c := range inv.containers {
		if c.Name == ref {
			return *c, true
		}
		if len(ref) >= 4 && strings.HasPrefix(c.ID, ref) {
			if found != nil {
				return Container{}, false // ambiguous
			}
			found = c
		}
	}
	if found != nil {
		return *found, true
	}
	return Container{}, false
}

// ByIP maps container IPs (v4/v6) to containers.
func (inv *Inventory) ByIP() map[string]Container {
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	m := map[string]Container{}
	for _, c := range inv.containers {
		for _, n := range c.Networks {
			if n.IPAddress != "" {
				m[n.IPAddress] = *c
			}
			if n.IPv6 != "" {
				m[n.IPv6] = *c
			}
		}
	}
	return m
}

// Counts used by the dashboard.
type Counts struct {
	Total      int `json:"total"`
	Running    int `json:"running"`
	Paused     int `json:"paused"`
	Restarting int `json:"restarting"`
	Stopped    int `json:"stopped"` // exited + created + dead
	Unhealthy  int `json:"unhealthy"`
	Healthy    int `json:"healthy"`
	Images     *int `json:"images"`
	Networks   *int `json:"networks"`
	Volumes    *int `json:"volumes"`
}

func (inv *Inventory) Counts() Counts {
	var c Counts
	inv.mu.RLock()
	for _, x := range inv.containers {
		c.Total++
		switch x.State {
		case "running":
			c.Running++
		case "paused":
			c.Paused++
		case "restarting":
			c.Restarting++
		default:
			c.Stopped++
		}
		switch x.Health {
		case "unhealthy":
			c.Unhealthy++
		case "healthy":
			c.Healthy++
		}
	}
	inv.mu.RUnlock()
	inv.resMu.RLock()
	if _, ok := inv.resAt["images"]; ok {
		n := len(inv.images)
		c.Images = &n
	}
	if _, ok := inv.resAt["networks"]; ok {
		n := len(inv.networks)
		c.Networks = &n
	}
	if _, ok := inv.resAt["volumes"]; ok {
		n := len(inv.volumes)
		c.Volumes = &n
	}
	inv.resMu.RUnlock()
	return c
}

// RefreshResources reloads networks, images and volumes. kinds may be empty for all.
func (inv *Inventory) RefreshResources(ctx context.Context, kinds ...string) {
	want := func(k string) bool {
		if len(kinds) == 0 {
			return true
		}
		for _, x := range kinds {
			if x == k {
				return true
			}
		}
		return false
	}
	if want("networks") {
		n, err := inv.gw.ListNetworks(ctx)
		inv.setRes("networks", err, func() { inv.networks = n })
	}
	if want("images") {
		i, err := inv.gw.ListImages(ctx)
		inv.setRes("images", err, func() { inv.images = i })
	}
	if want("volumes") {
		v, err := inv.gw.ListVolumes(ctx)
		inv.setRes("volumes", err, func() { inv.volumes = v })
	}
}

func (inv *Inventory) setRes(kind string, err error, apply func()) {
	inv.resMu.Lock()
	defer inv.resMu.Unlock()
	if err != nil {
		inv.resErr[kind] = err.Error()
		return
	}
	apply()
	inv.resAt[kind] = time.Now()
	delete(inv.resErr, kind)
}

// Resources returns cached lists, refreshing them when older than maxAge.
func (inv *Inventory) Networks(ctx context.Context, maxAge time.Duration) ([]network.Summary, error) {
	if err := inv.ensure(ctx, "networks", maxAge); err != nil {
		return nil, err
	}
	inv.resMu.RLock()
	defer inv.resMu.RUnlock()
	return append([]network.Summary{}, inv.networks...), nil
}

func (inv *Inventory) Images(ctx context.Context, maxAge time.Duration) ([]image.Summary, error) {
	if err := inv.ensure(ctx, "images", maxAge); err != nil {
		return nil, err
	}
	inv.resMu.RLock()
	defer inv.resMu.RUnlock()
	return append([]image.Summary{}, inv.images...), nil
}

func (inv *Inventory) Volumes(ctx context.Context, maxAge time.Duration) ([]volume.Volume, error) {
	if err := inv.ensure(ctx, "volumes", maxAge); err != nil {
		return nil, err
	}
	inv.resMu.RLock()
	defer inv.resMu.RUnlock()
	return append([]volume.Volume{}, inv.volumes...), nil
}

func (inv *Inventory) ensure(ctx context.Context, kind string, maxAge time.Duration) error {
	inv.resMu.RLock()
	at, ok := inv.resAt[kind]
	inv.resMu.RUnlock()
	if ok && time.Since(at) < maxAge {
		return nil
	}
	inv.RefreshResources(ctx, kind)
	inv.resMu.RLock()
	defer inv.resMu.RUnlock()
	if e, bad := inv.resErr[kind]; bad {
		if _, have := inv.resAt[kind]; !have {
			return errors.New(e)
		}
	}
	return nil
}

// Run keeps the inventory fresh: periodic full resync as safety net in addition
// to the event-driven updates.
func (inv *Inventory) Run(ctx context.Context) {
	t := time.NewTicker(60 * time.Second)
	defer t.Stop()
	for {
		if inv.gw.Connected() {
			if err := inv.Sync(ctx); err != nil && ctx.Err() == nil {
				inv.log.Warn("container sync failed", "err", err)
			}
			inv.RefreshResources(ctx)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

