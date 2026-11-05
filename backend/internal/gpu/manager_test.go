package gpu

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/phili/orbit/internal/config"
	"github.com/phili/orbit/internal/docker"
)

func TestGPUManagerNoGPUFallback(t *testing.T) {
	cfg := &config.Config{
		GPUMode:  "auto",
		HostProc: t.TempDir(),
		HostSys:  t.TempDir(),
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	inv := docker.NewInventory(nil, logger)

	mgr := NewManager(cfg, inv, logger)
	defer mgr.Shutdown()

	telemetry := mgr.Collect(context.Background())

	if telemetry.Available {
		t.Errorf("expected no GPU available in test environment")
	}
	if telemetry.Message != "Keine kompatible GPU erkannt." {
		t.Errorf("expected fallback message 'Keine kompatible GPU erkannt.', got '%s'", telemetry.Message)
	}
}
