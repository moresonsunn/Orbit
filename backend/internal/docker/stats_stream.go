package docker

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/moby/moby/api/types/container"
)

type StatsSubscriber func(ContainerStats)

type StatsStreamer struct {
	gw              *Gateway
	inv             *Inventory
	log             *slog.Logger
	maxStreams      int
	mu              sync.RWMutex
	running         map[string]*containerWorker
	latestStats     map[string]ContainerStats
	recentHistory   map[string][]ContainerStats // short ring buffer for live charts (e.g. last 60 samples)
	subscribers     []StatsSubscriber
}

type containerWorker struct {
	id     string
	cancel context.CancelFunc
	done   chan struct{}
}

func NewStatsStreamer(gw *Gateway, inv *Inventory, maxStreams int, log *slog.Logger) *StatsStreamer {
	if maxStreams <= 0 {
		maxStreams = 250
	}
	return &StatsStreamer{
		gw:            gw,
		inv:           inv,
		log:           log,
		maxStreams:    maxStreams,
		running:       make(map[string]*containerWorker),
		latestStats:   make(map[string]ContainerStats),
		recentHistory: make(map[string][]ContainerStats),
	}
}

func (ss *StatsStreamer) Subscribe(sub StatsSubscriber) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.subscribers = append(ss.subscribers, sub)
}

func (ss *StatsStreamer) emit(stat ContainerStats) {
	ss.mu.Lock()
	ss.latestStats[stat.ID] = stat

	// Keep last 60 samples in ring buffer
	hist := ss.recentHistory[stat.ID]
	if len(hist) >= 60 {
		hist = append(hist[1:], stat)
	} else {
		hist = append(hist, stat)
	}
	ss.recentHistory[stat.ID] = hist

	subs := append([]StatsSubscriber{}, ss.subscribers...)
	ss.mu.Unlock()

	for _, sub := range subs {
		sub(stat)
	}
}

func (ss *StatsStreamer) GetLatest(id string) (ContainerStats, bool) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	st, ok := ss.latestStats[id]
	return st, ok
}

func (ss *StatsStreamer) GetAllLatest() []ContainerStats {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	out := make([]ContainerStats, 0, len(ss.latestStats))
	for _, st := range ss.latestStats {
		out = append(out, st)
	}
	return out
}

func (ss *StatsStreamer) GetRecentHistory(id string) []ContainerStats {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	hist := ss.recentHistory[id]
	if len(hist) == 0 {
		return nil
	}
	out := make([]ContainerStats, len(hist))
	copy(out, hist)
	return out
}

func (ss *StatsStreamer) Run(ctx context.Context) {
	// Watch inventory changes to start/stop workers
	ss.inv.OnChange(func(ch ContainerChange) {
		if ch.Kind == ChangeRemove {
			ss.stopWorker(ch.ID)
		} else if ch.Container != nil {
			if ch.Container.State == "running" {
				ss.startWorker(ctx, ch.Container.ID, ch.Container.Limits.CPUs)
			} else {
				ss.stopWorker(ch.Container.ID)
			}
		}
	})

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			ss.stopAll()
			return
		case <-ticker.C:
			ss.reconcile(ctx)
		}
	}
}

func (ss *StatsStreamer) reconcile(ctx context.Context) {
	if !ss.gw.Connected() {
		return
	}

	containers := ss.inv.Containers()
	activeIDs := make(map[string]bool)

	for _, c := range containers {
		if c.State == "running" {
			activeIDs[c.ID] = true
			ss.startWorker(ctx, c.ID, c.Limits.CPUs)
		}
	}

	ss.mu.Lock()
	for id, worker := range ss.running {
		if !activeIDs[id] {
			worker.cancel()
			delete(ss.running, id)
			delete(ss.latestStats, id)
		}
	}
	ss.mu.Unlock()
}

func (ss *StatsStreamer) startWorker(parentCtx context.Context, id string, cpuLimit float64) {
	ss.mu.Lock()
	if _, exists := ss.running[id]; exists {
		ss.mu.Unlock()
		return
	}

	if len(ss.running) >= ss.maxStreams {
		ss.log.Warn("max stats streams reached, skipping container", "id", id, "limit", ss.maxStreams)
		ss.mu.Unlock()
		return
	}

	ctx, cancel := context.WithCancel(parentCtx)
	worker := &containerWorker{
		id:     id,
		cancel: cancel,
		done:   make(chan struct{}),
	}
	ss.running[id] = worker
	ss.mu.Unlock()

	go ss.streamStats(ctx, worker, cpuLimit)
}

func (ss *StatsStreamer) stopWorker(id string) {
	ss.mu.Lock()
	worker, exists := ss.running[id]
	if exists {
		worker.cancel()
		delete(ss.running, id)
		delete(ss.latestStats, id)
	}
	ss.mu.Unlock()
}

func (ss *StatsStreamer) stopAll() {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	for id, worker := range ss.running {
		worker.cancel()
		delete(ss.running, id)
	}
}

func (ss *StatsStreamer) streamStats(ctx context.Context, worker *containerWorker, cpuLimit float64) {
	defer close(worker.done)
	calc := &Calculator{}

	backoff := time.Second
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		body, err := ss.gw.StatsStream(ctx, worker.id)
		if err != nil {
			if errors.Is(err, ErrNotFound) || errors.Is(err, context.Canceled) {
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
				if backoff < 10*time.Second {
					backoff *= 2
				}
				continue
			}
		}

		backoff = time.Second
		dec := json.NewDecoder(body)

		for {
			var resp container.StatsResponse
			if err := dec.Decode(&resp); err != nil {
				body.Close()
				if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
					return
				}
				// Break to retry loop
				break
			}

			computed := calc.Compute(worker.id, &resp, cpuLimit)
			ss.emit(computed)

			select {
			case <-ctx.Done():
				body.Close()
				return
			default:
			}
		}
	}
}
