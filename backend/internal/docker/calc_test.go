package docker

import (
	"testing"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/phili/orbit/internal/model"
)

func TestCPUPercent(t *testing.T) {
	pre := container.CPUStats{
		CPUUsage: container.CPUUsage{
			TotalUsage: 1_000_000_000,
		},
		SystemUsage: 10_000_000_000,
		OnlineCPUs:  4,
	}
	cur := container.CPUStats{
		CPUUsage: container.CPUUsage{
			TotalUsage: 2_000_000_000, // 1 sec of CPU
		},
		SystemUsage: 11_000_000_000, // 1 sec elapsed on host
		OnlineCPUs:  4,
	}

	pct, ok := CPUPercent(cur, pre)
	if !ok {
		t.Fatalf("expected ok=true")
	}
	// cpuDelta = 1e9, sysDelta = 1e9, online = 4 => (1/1)*4*100 = 400%
	if pct != 400.0 {
		t.Errorf("expected 400.0, got %f", pct)
	}
}

func TestMemoryUsage(t *testing.T) {
	// Test cgroup v2 inactive_file subtraction
	msV2 := container.MemoryStats{
		Usage: 500 * 1024 * 1024,
		Stats: map[string]uint64{
			"inactive_file": 100 * 1024 * 1024,
			"file":          150 * 1024 * 1024,
		},
	}
	usage, cache, ok := MemoryUsage(msV2)
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if usage != 400*1024*1024 {
		t.Errorf("expected 400MB used, got %d", usage)
	}
	if cache != 150*1024*1024 {
		t.Errorf("expected 150MB cache, got %d", cache)
	}

	// Test cgroup v1 total_inactive_file subtraction
	msV1 := container.MemoryStats{
		Usage: 600 * 1024 * 1024,
		Stats: map[string]uint64{
			"total_inactive_file": 200 * 1024 * 1024,
			"total_cache":         250 * 1024 * 1024,
		},
	}
	usageV1, cacheV1, okV1 := MemoryUsage(msV1)
	if !okV1 {
		t.Fatalf("expected ok=true")
	}
	if usageV1 != 400*1024*1024 {
		t.Errorf("expected 400MB used, got %d", usageV1)
	}
	if cacheV1 != 250*1024*1024 {
		t.Errorf("expected 250MB cache, got %d", cacheV1)
	}
}

func TestCalculatorCompute(t *testing.T) {
	calc := &Calculator{}
	now := time.Now()

	s1 := &container.StatsResponse{
		Read: now,
		CPUStats: container.CPUStats{
			CPUUsage:    container.CPUUsage{TotalUsage: 1_000_000_000},
			SystemUsage: 10_000_000_000,
			OnlineCPUs:  2,
		},
		MemoryStats: container.MemoryStats{
			Usage: 200 * 1024 * 1024,
			Limit: 1024 * 1024 * 1024,
		},
	}

	st1 := calc.Compute("c1", s1, 2.0)
	if st1.CPUPercent.Status != model.Pending {
		t.Errorf("expected pending cpu status on first sample, got %s", st1.CPUPercent.Status)
	}
	if st1.MemUsage.Status != model.OK || *st1.MemUsage.Value != 200*1024*1024 {
		t.Errorf("expected ok mem usage")
	}

	s2 := &container.StatsResponse{
		Read: now.Add(1 * time.Second),
		CPUStats: container.CPUStats{
			CPUUsage:    container.CPUUsage{TotalUsage: 1_500_000_000}, // 0.5s CPU
			SystemUsage: 11_000_000_000,                               // 1.0s system
			OnlineCPUs:  2,
		},
		MemoryStats: container.MemoryStats{
			Usage: 250 * 1024 * 1024,
			Limit: 1024 * 1024 * 1024,
		},
	}

	st2 := calc.Compute("c1", s2, 2.0)
	if st2.CPUPercent.Status != model.OK {
		t.Errorf("expected ok cpu status on second sample, got %s", st2.CPUPercent.Status)
	}
	// 0.5 / 1.0 * 2 cores * 100% = 100%
	if *st2.CPUPercent.Value != 100.0 {
		t.Errorf("expected 100%% cpu, got %f", *st2.CPUPercent.Value)
	}
}

