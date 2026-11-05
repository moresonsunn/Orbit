package host

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/phili/orbit/internal/config"
	"github.com/phili/orbit/internal/model"
)

func TestCollectorWithMockProc(t *testing.T) {
	tmpDir := t.TempDir()
	procDir := filepath.Join(tmpDir, "proc")
	sysDir := filepath.Join(tmpDir, "sys")
	etcDir := filepath.Join(tmpDir, "etc")
	_ = os.MkdirAll(filepath.Join(procDir, "net"), 0o755)
	_ = os.MkdirAll(sysDir, 0o755)
	_ = os.MkdirAll(etcDir, 0o755)

	// Mock /proc/stat
	statContent := `cpu  1000 0 1000 8000 0 0 0 0 0 0
cpu0 500 0 500 4000 0 0 0 0 0 0
cpu1 500 0 500 4000 0 0 0 0 0 0
`
	_ = os.WriteFile(filepath.Join(procDir, "stat"), []byte(statContent), 0o644)

	// Mock /proc/meminfo
	memContent := `MemTotal:       16000000 kB
MemFree:         4000000 kB
MemAvailable:   10000000 kB
Buffers:          500000 kB
Cached:          5500000 kB
SwapTotal:       2000000 kB
SwapFree:        1500000 kB
`
	_ = os.WriteFile(filepath.Join(procDir, "meminfo"), []byte(memContent), 0o644)

	// Mock /proc/loadavg
	_ = os.WriteFile(filepath.Join(procDir, "loadavg"), []byte("1.25 0.85 0.55 2/450 1234\n"), 0o644)

	// Mock /proc/uptime
	_ = os.WriteFile(filepath.Join(procDir, "uptime"), []byte("3600.50 7200.00\n"), 0o644)

	cfg := &config.Config{
		HostProc: procDir,
		HostSys:  sysDir,
		HostEtc:  etcDir,
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	collector := NewCollector(cfg, logger)

	// First collection (pending delta metrics)
	m1 := collector.Collect(context.Background())

	if m1.Memory.TotalBytes != 16000000*1024 {
		t.Errorf("expected 16GB total memory, got %d", m1.Memory.TotalBytes)
	}
	if m1.Memory.UsedPercent.Status != model.OK || *m1.Memory.UsedPercent.Value != 37.5 {
		t.Errorf("expected 37.5%% used memory, got %v", m1.Memory.UsedPercent.Value)
	}
	if *m1.CPU.LoadAvg1.Value != 1.25 {
		t.Errorf("expected loadavg1 1.25, got %v", m1.CPU.LoadAvg1.Value)
	}
	if *m1.System.UptimeSeconds.Value != 3600.5 {
		t.Errorf("expected uptime 3600.5, got %v", m1.System.UptimeSeconds.Value)
	}

	// Update stat to simulate 50% CPU usage
	statContent2 := `cpu  1500 0 1500 9000 0 0 0 0 0 0
cpu0 750 0 750 4500 0 0 0 0 0 0
cpu1 750 0 750 4500 0 0 0 0 0 0
`
	_ = os.WriteFile(filepath.Join(procDir, "stat"), []byte(statContent2), 0o644)

	m2 := collector.Collect(context.Background())
	if m2.CPU.TotalPercent.Status != model.OK {
		t.Fatalf("expected ok cpu percent on second collection, got %s", m2.CPU.TotalPercent.Status)
	}
	// total = 1000 + 1000 = 2000 delta. idle = 1000 delta => (1 - 1000/2000) * 100 = 50%
	if *m2.CPU.TotalPercent.Value != 50.0 {
		t.Errorf("expected 50%% cpu, got %f", *m2.CPU.TotalPercent.Value)
	}
}
