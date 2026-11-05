package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/phili/orbit/internal/alerts"
	"github.com/phili/orbit/internal/api"
	"github.com/phili/orbit/internal/auth"
	"github.com/phili/orbit/internal/config"
	"github.com/phili/orbit/internal/docker"
	"github.com/phili/orbit/internal/gpu"
	"github.com/phili/orbit/internal/host"
	"github.com/phili/orbit/internal/metrics"
	"github.com/phili/orbit/internal/netflow"
	"github.com/phili/orbit/internal/realtime"
	"github.com/phili/orbit/internal/settings"
	"github.com/phili/orbit/internal/store"
	"github.com/phili/orbit/web"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-healthcheck" {
		client := http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get("http://127.0.0.1:8787/api/v1/health")
		if err != nil || resp.StatusCode != http.StatusOK {
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Setup structured JSON logger
	logLevel := slog.LevelInfo
	if os.Getenv("ORBIT_LOG_LEVEL") == "debug" {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	logger.Info("starting Orbit - Self-hosted Docker Infrastructure, Management & Real-Time Monitoring Platform")

	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration error", "err", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// 1. Initialize SQLite Store
	st, err := store.Open(cfg.DataDir)
	if err != nil {
		logger.Error("failed to open database", "err", err, "path", cfg.DataDir)
		os.Exit(1)
	}
	defer st.Close()
	logger.Info("database opened successfully", "dbSize", st.SizeBytes())

	// 2. Settings Manager
	settingsMgr, err := settings.NewManager(ctx, st)
	if err != nil {
		logger.Error("failed to initialize settings", "err", err)
		os.Exit(1)
	}

	// 3. Auth Manager
	authMgr, err := auth.NewManager(cfg, st, logger)
	if err != nil {
		logger.Error("failed to initialize auth manager", "err", err)
		os.Exit(1)
	}

	// 4. Docker Gateway
	gw, err := docker.NewGateway(cfg.DockerHost)
	if err != nil {
		logger.Error("failed to create docker client", "err", err, "host", cfg.DockerHost)
		os.Exit(1)
	}
	defer gw.Close()

	// Initial Docker ping
	if err := gw.Ping(ctx); err != nil {
		logger.Warn("docker daemon not reachable yet; background monitor will keep retrying", "err", err)
	} else {
		info, _ := gw.Info(ctx)
		logger.Info("connected to Docker Engine", "version", info.ServerVersion, "containers", info.Containers)
	}
	go gw.MonitorConnection(ctx)

	// 5. Inventory
	inv := docker.NewInventory(gw, logger)
	go inv.Run(ctx)

	// 6. Stats Streamer
	streamer := docker.NewStatsStreamer(gw, inv, cfg.MaxStatsStreams, logger)
	go streamer.Run(ctx)

	// 7. Event Watcher
	eventWatcher := docker.NewEventWatcher(gw, inv, st, logger)
	go eventWatcher.Run(ctx)

	// 8. Realtime WebSocket Hub
	hub := realtime.NewHub(cfg, logger)
	go hub.Run(ctx)

	// 9. Alert Engine
	alertEngine := alerts.NewEngine(settingsMgr, st, hub, logger)
	eventWatcher.OnEvent(func(ev docker.DockerEvent) {
		alertEngine.HandleDockerEvent(ctx, ev)
		hub.Broadcast("events", ev)
	})

	// 10. Metrics History Store
	historyStore := metrics.NewHistoryStore(st, settingsMgr, logger)
	go historyStore.Run(ctx)

	// 11. Host Collector & GPU Manager & Netflow Builder
	hostCollector := host.NewCollector(cfg, logger)
	gpuManager := gpu.NewManager(cfg, inv, logger)
	defer gpuManager.Shutdown()
	netflowBuilder := netflow.NewBuilder(cfg, inv, streamer)

	// Hook container stats stream into WebSocket hub, History & Alert Engine
	streamer.Subscribe(func(st docker.ContainerStats) {
		hub.Broadcast("containers", st)
		hub.Broadcast(fmt.Sprintf("container:%s", st.ID), st)
		historyStore.RecordContainer(st)

		if c, ok := inv.Get(st.ID); ok {
			alertEngine.CheckContainer(ctx, c, st)
		}
	})

	// Background collection loops
	go runHostMonitoringLoop(ctx, hostCollector, alertEngine, historyStore, hub, settingsMgr)
	go runGPUMonitoringLoop(ctx, gpuManager, hub, settingsMgr)
	go runNetflowMonitoringLoop(ctx, netflowBuilder, hub, settingsMgr)

	// 12. Setup HTTP Server with API & Frontend SPA
	spaFS, _ := web.DistFS()
	serverAPI := api.NewServer(
		cfg, st, gw, inv, streamer, eventWatcher, hostCollector, gpuManager, netflowBuilder,
		historyStore, alertEngine, settingsMgr, authMgr,
	)
	router := api.SetupRouter(serverAPI, hub, spaFS)

	httpServer := &http.Server{
		Addr:              cfg.Listen,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		logger.Info(fmt.Sprintf("Orbit listening on http://0.0.0.0%s", cfg.Listen))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "err", err)
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	sig := <-sigCh
	logger.Info("shutting down Orbit gracefully", "signal", sig.String())

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	cancel() // cancel background workers
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("http shutdown error", "err", err)
	}

	logger.Info("Orbit shutdown complete")
}

func runHostMonitoringLoop(ctx context.Context, col *host.Collector, alertsEng *alerts.Engine, hist *metrics.HistoryStore, hub *realtime.Hub, sm *settings.Manager) {
	ticker := time.NewTicker(time.Duration(sm.Get().Intervals.HostMs) * time.Millisecond)
	defer ticker.Stop()

	sm.OnChange(func(s settings.Settings) {
		ticker.Reset(time.Duration(s.Intervals.HostMs) * time.Millisecond)
	})

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m := col.Collect(ctx)
			hub.Broadcast("host", m)
			hist.RecordHost(m)
			alertsEng.CheckHost(ctx, m)
		}
	}
}

func runGPUMonitoringLoop(ctx context.Context, mgr *gpu.Manager, hub *realtime.Hub, sm *settings.Manager) {
	ticker := time.NewTicker(time.Duration(sm.Get().Intervals.GPUMs) * time.Millisecond)
	defer ticker.Stop()

	sm.OnChange(func(s settings.Settings) {
		ticker.Reset(time.Duration(s.Intervals.GPUMs) * time.Millisecond)
	})

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t := mgr.Collect(ctx)
			hub.Broadcast("gpu", t)
		}
	}
}

func runNetflowMonitoringLoop(ctx context.Context, bld *netflow.Builder, hub *realtime.Hub, sm *settings.Manager) {
	ticker := time.NewTicker(time.Duration(sm.Get().Intervals.NetflowMs) * time.Millisecond)
	defer ticker.Stop()

	sm.OnChange(func(s settings.Settings) {
		ticker.Reset(time.Duration(s.Intervals.NetflowMs) * time.Millisecond)
	})

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			snapshot := bld.BuildSnapshot(ctx)
			hub.Broadcast("netflow", snapshot)
		}
	}
}
