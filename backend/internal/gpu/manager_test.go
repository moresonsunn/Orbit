package gpu

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/phili/orbit/internal/config"
	"github.com/phili/orbit/internal/docker"
)

func TestGPUManagerNoGPUFallback(t *testing.T) {
	tempSys := t.TempDir()
	// Create empty class/drm directory inside mock sysfs so it doesn't fall back to host /sys/class/drm on Linux CI runners
	_ = os.MkdirAll(filepath.Join(tempSys, "class", "drm"), 0755)

	cfg := &config.Config{
		GPUMode:  "disabled",
		HostProc: t.TempDir(),
		HostSys:  tempSys,
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

