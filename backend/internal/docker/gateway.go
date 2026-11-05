// Package docker contains Orbit's only access path to the Docker Engine.
//
// Security: the Gateway type intentionally exposes an explicit allowlist of
// operations. There is no generic pass-through/proxy endpoint, so the web API can
// never be used to create containers, exec into containers, mount host paths or
// otherwise escalate to root on the host. Mutating operations are additionally
// gated by feature flags in the API layer.
package docker

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/volume"
	"github.com/moby/moby/client"
)

var ErrNotFound = errors.New("not found")

type Gateway struct {
	cli *client.Client

	mu        sync.RWMutex
	connected atomic.Bool
	lastErr   string
	version   client.ServerVersionResult
	info      SystemInfo
	infoAt    time.Time

	onState []func(bool)
}

// SystemInfo is the subset of docker info Orbit displays.
type SystemInfo struct {
	ServerVersion   string `json:"serverVersion"`
	APIVersion      string `json:"apiVersion"`
	OperatingSystem string `json:"operatingSystem"`
	OSType          string `json:"osType"`
	KernelVersion   string `json:"kernelVersion"`
	Architecture    string `json:"architecture"`
	NCPU            int    `json:"ncpu"`
	MemTotal        int64  `json:"memTotal"`
	Name            string `json:"name"`
	Driver          string `json:"storageDriver"`
	CgroupDriver    string `json:"cgroupDriver"`
	CgroupVersion   string `json:"cgroupVersion"`
	DockerRootDir   string `json:"dockerRootDir"`
	Containers      int    `json:"containers"`
	Running         int    `json:"containersRunning"`
	Paused          int    `json:"containersPaused"`
	Stopped         int    `json:"containersStopped"`
	Images          int    `json:"images"`
	Runtimes        []string `json:"runtimes"`
	DefaultRuntime  string `json:"defaultRuntime"`
	SecurityOptions []string `json:"securityOptions"`
	Warnings        []string `json:"warnings"`
}

func NewGateway(host string) (*Gateway, error) {
	cli, err := client.New(
		client.WithHost(host),
		client.WithAPIVersionNegotiation(),
		client.WithUserAgent("orbit"),
	)
	if err != nil {
		return nil, err
	}
	return &Gateway{cli: cli}, nil
}

func (g *Gateway) Close() error { return g.cli.Close() }

func (g *Gateway) Connected() bool { return g.connected.Load() }

func (g *Gateway) LastError() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.lastErr
}

// OnStateChange registers a callback invoked when connectivity flips.
func (g *Gateway) OnStateChange(f func(connected bool)) {
	g.mu.Lock()
	g.onState = append(g.onState, f)
	g.mu.Unlock()
}

func (g *Gateway) setState(ok bool, err error) {
	g.mu.Lock()
	if err != nil {
		g.lastErr = err.Error()
	} else {
		g.lastErr = ""
	}
	cbs := append([]func(bool){}, g.onState...)
	g.mu.Unlock()
	if g.connected.Swap(ok) != ok {
		for _, f := range cbs {
			f(ok)
		}
	}
}

// Ping checks connectivity and refreshes version information.
func (g *Gateway) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if _, err := g.cli.Ping(ctx, client.PingOptions{NegotiateAPIVersion: true}); err != nil {
		g.setState(false, err)
		return err
	}
	v, err := g.cli.ServerVersion(ctx, client.ServerVersionOptions{})
	if err != nil {
		g.setState(false, err)
		return err
	}
	g.mu.Lock()
	g.version = v
	g.mu.Unlock()
	g.setState(true, nil)
	return nil
}

// MonitorConnection pings the daemon periodically and updates the state.
func (g *Gateway) MonitorConnection(ctx context.Context) {
	backoff := time.Second
	for {
		err := g.Ping(ctx)
		wait := 10 * time.Second
		if err != nil {
			wait = backoff
			if backoff < 30*time.Second {
				backoff *= 2
			}
		} else {
			backoff = time.Second
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
	}
}

func (g *Gateway) note(err error) error {
	if err == nil {
		return nil
	}
	if client.IsErrConnectionFailed(err) {
		g.setState(false, err)
	}
	if isNotFound(err) {
		return ErrNotFound
	}
	return err
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	if strings.Contains(errStr, "not found") || strings.Contains(errStr, "no such container") || strings.Contains(errStr, "404") {
		return true
	}
	var se interface{ StatusCode() int }
	if errors.As(err, &se) && se.StatusCode() == http.StatusNotFound {
		return true
	}
	type notFound interface{ NotFound() }
	var nf notFound
	return errors.As(err, &nf)
}

func (g *Gateway) Info(ctx context.Context) (SystemInfo, error) {
	g.mu.RLock()
	if time.Since(g.infoAt) < 3*time.Second && g.info.ServerVersion != "" {
		i := g.info
		g.mu.RUnlock()
		return i, nil
	}
	g.mu.RUnlock()
	res, err := g.cli.Info(ctx, client.InfoOptions{})
	if err != nil {
		return SystemInfo{}, g.note(err)
	}
	in := res.Info
	si := SystemInfo{
		ServerVersion: in.ServerVersion, OperatingSystem: in.OperatingSystem, OSType: in.OSType,
		KernelVersion: in.KernelVersion, Architecture: in.Architecture, NCPU: in.NCPU, MemTotal: in.MemTotal,
		Name: in.Name, Driver: in.Driver, CgroupDriver: in.CgroupDriver, CgroupVersion: in.CgroupVersion,
		DockerRootDir: in.DockerRootDir, Containers: in.Containers, Running: in.ContainersRunning,
		Paused: in.ContainersPaused, Stopped: in.ContainersStopped, Images: in.Images,
		DefaultRuntime: in.DefaultRuntime, SecurityOptions: in.SecurityOptions, Warnings: in.Warnings,
	}
	for name := range in.Runtimes {
		si.Runtimes = append(si.Runtimes, name)
	}
	g.mu.Lock()
	si.APIVersion = g.version.APIVersion
	g.info, g.infoAt = si, time.Now()
	g.mu.Unlock()
	return si, nil
}

// ---- read operations ----

func (g *Gateway) ListContainers(ctx context.Context) ([]container.Summary, error) {
	res, err := g.cli.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil {
		return nil, g.note(err)
	}
	return res.Items, nil
}

func (g *Gateway) InspectContainer(ctx context.Context, id string) (container.InspectResponse, json.RawMessage, error) {
	res, err := g.cli.ContainerInspect(ctx, id, client.ContainerInspectOptions{})
	if err != nil {
		return container.InspectResponse{}, nil, g.note(err)
	}
	return res.Container, res.Raw, nil
}

// StatsStream opens a streaming stats connection. The caller must close the reader.
func (g *Gateway) StatsStream(ctx context.Context, id string) (io.ReadCloser, error) {
	res, err := g.cli.ContainerStats(ctx, id, client.ContainerStatsOptions{Stream: true})
	if err != nil {
		return nil, g.note(err)
	}
	return res.Body, nil
}

type LogOptions struct {
	Tail       string
	Since      string
	Until      string
	Follow     bool
	Timestamps bool
}

func (g *Gateway) Logs(ctx context.Context, id string, o LogOptions) (io.ReadCloser, error) {
	res, err := g.cli.ContainerLogs(ctx, id, client.ContainerLogsOptions{
		ShowStdout: true, ShowStderr: true, Tail: o.Tail, Since: o.Since, Until: o.Until,
		Follow: o.Follow, Timestamps: o.Timestamps,
	})
	if err != nil {
		return nil, g.note(err)
	}
	return res, nil
}

func (g *Gateway) Top(ctx context.Context, id string) (client.ContainerTopResult, error) {
	res, err := g.cli.ContainerTop(ctx, id, client.ContainerTopOptions{})
	return res, g.note(err)
}

func (g *Gateway) ListNetworks(ctx context.Context) ([]network.Summary, error) {
	res, err := g.cli.NetworkList(ctx, client.NetworkListOptions{})
	if err != nil {
		return nil, g.note(err)
	}
	return res.Items, nil
}

func (g *Gateway) InspectNetwork(ctx context.Context, id string) (network.Inspect, error) {
	res, err := g.cli.NetworkInspect(ctx, id, client.NetworkInspectOptions{})
	if err != nil {
		return network.Inspect{}, g.note(err)
	}
	return res.Network, nil
}

func (g *Gateway) ListImages(ctx context.Context) ([]image.Summary, error) {
	res, err := g.cli.ImageList(ctx, client.ImageListOptions{All: false})
	if err != nil {
		return nil, g.note(err)
	}
	return res.Items, nil
}

func (g *Gateway) InspectImage(ctx context.Context, id string) (image.InspectResponse, error) {
	res, err := g.cli.ImageInspect(ctx, id)
	if err != nil {
		return image.InspectResponse{}, g.note(err)
	}
	return res.InspectResponse, nil
}

func (g *Gateway) ListVolumes(ctx context.Context) ([]volume.Volume, error) {
	res, err := g.cli.VolumeList(ctx, client.VolumeListOptions{})
	if err != nil {
		return nil, g.note(err)
	}
	return res.Items, nil
}

func (g *Gateway) InspectVolume(ctx context.Context, name string) (volume.Volume, error) {
	res, err := g.cli.VolumeInspect(ctx, name, client.VolumeInspectOptions{})
	if err != nil {
		return volume.Volume{}, g.note(err)
	}
	return res.Volume, nil
}

// VolumeSizes runs the (potentially expensive) disk usage calculation for volumes.
func (g *Gateway) VolumeSizes(ctx context.Context) (map[string]int64, error) {
	res, err := g.cli.DiskUsage(ctx, client.DiskUsageOptions{Volumes: true, Verbose: true})
	if err != nil {
		return nil, g.note(err)
	}
	out := map[string]int64{}
	for _, v := range res.Volumes.Items {
		if v.UsageData != nil && v.UsageData.Size >= 0 {
			out[v.Name] = v.UsageData.Size
		}
	}
	return out, nil
}

func (g *Gateway) Events(ctx context.Context, since string) client.EventsResult {
	return g.cli.Events(ctx, client.EventsListOptions{Since: since})
}

// ---- mutating operations (gated by the API layer) ----

func (g *Gateway) Start(ctx context.Context, id string) error {
	_, err := g.cli.ContainerStart(ctx, id, client.ContainerStartOptions{})
	return g.note(err)
}

func (g *Gateway) Stop(ctx context.Context, id string) error {
	_, err := g.cli.ContainerStop(ctx, id, client.ContainerStopOptions{})
	return g.note(err)
}

func (g *Gateway) Restart(ctx context.Context, id string) error {
	_, err := g.cli.ContainerRestart(ctx, id, client.ContainerRestartOptions{})
	return g.note(err)
}

func (g *Gateway) Pause(ctx context.Context, id string) error {
	_, err := g.cli.ContainerPause(ctx, id, client.ContainerPauseOptions{})
	return g.note(err)
}

func (g *Gateway) Unpause(ctx context.Context, id string) error {
	_, err := g.cli.ContainerUnpause(ctx, id, client.ContainerUnpauseOptions{})
	return g.note(err)
}

// RemoveContainer never removes volumes; force only if explicitly requested.
func (g *Gateway) RemoveContainer(ctx context.Context, id string, force bool) error {
	_, err := g.cli.ContainerRemove(ctx, id, client.ContainerRemoveOptions{Force: force, RemoveVolumes: false})
	return g.note(err)
}

func (g *Gateway) RemoveImage(ctx context.Context, id string) ([]image.DeleteResponse, error) {
	res, err := g.cli.ImageRemove(ctx, id, client.ImageRemoveOptions{Force: false, PruneChildren: true})
	return res.Items, g.note(err)
}

// PruneImages removes dangling images, or all unused images when all is true.
func (g *Gateway) PruneImages(ctx context.Context, all bool) (image.PruneReport, error) {
	f := client.Filters{}
	if all {
		f = f.Add("dangling", "false")
	} else {
		f = f.Add("dangling", "true")
	}
	res, err := g.cli.ImagePrune(ctx, client.ImagePruneOptions{Filters: f})
	return res.Report, g.note(err)
}

func (g *Gateway) RemoveVolume(ctx context.Context, name string) error {
	_, err := g.cli.VolumeRemove(ctx, name, client.VolumeRemoveOptions{Force: false})
	return g.note(err)
}

// PruneVolumes removes unused anonymous volumes, or all unused volumes when all is true.
func (g *Gateway) PruneVolumes(ctx context.Context, all bool) (volume.PruneReport, error) {
	res, err := g.cli.VolumePrune(ctx, client.VolumePruneOptions{All: all})
	return res.Report, g.note(err)
}

func (g *Gateway) RemoveNetwork(ctx context.Context, id string) error {
	_, err := g.cli.NetworkRemove(ctx, id, client.NetworkRemoveOptions{})
	return g.note(err)
}

