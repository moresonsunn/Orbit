package gpu

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/phili/orbit/internal/config"
	"github.com/phili/orbit/internal/docker"
)

type Manager struct {
	cfg       *config.Config
	inv       *docker.Inventory
	log       *slog.Logger
	providers []Provider

	mu     sync.RWMutex
	latest Telemetry
}

func NewManager(cfg *config.Config, inv *docker.Inventory, log *slog.Logger) *Manager {
	m := &Manager{
		cfg: cfg,
		inv: inv,
		log: log,
		latest: Telemetry{
			Timestamp: time.Now(),
			Available: false,
			Message:   "Keine kompatible GPU erkannt.",
		},
	}

	if cfg.GPUMode == "off" {
		m.latest.Message = "GPU-Überwachung ist in der Konfiguration deaktiviert."
		return m
	}

	// Try NVML first (preferred for NVIDIA GPUs)
	nvmlProv := NewNVMLProvider(log)
	if nvmlProv.Available() {
		m.providers = append(m.providers, nvmlProv)
	}

	// Try DRM provider (AMD / Intel sysfs)
	drmProv := NewDRMProvider(cfg, log)
	if drmProv.Available() {
		m.providers = append(m.providers, drmProv)
	}

	if len(m.providers) == 0 {
		m.log.Info("no compatible GPU providers found")
	}

	return m
}

func (m *Manager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range m.providers {
		p.Shutdown()
	}
}

func (m *Manager) GetLatest() Telemetry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.latest
}

func (m *Manager) Collect(ctx context.Context) Telemetry {
	if m.cfg.GPUMode == "off" || len(m.providers) == 0 {
		return m.GetLatest()
	}

	var allDevices []DeviceMetrics
	for _, p := range m.providers {
		devices, err := p.Collect(ctx)
		if err != nil {
			m.log.Warn("gpu provider collect failed", "provider", p.Name(), "err", err)
			continue
		}
		for i := range devices {
			m.enrichContainerProcesses(&devices[i])
		}
		allDevices = append(allDevices, devices...)
	}

	t := Telemetry{
		Timestamp: time.Now(),
		Available: len(allDevices) > 0,
		Devices:   allDevices,
	}

	if !t.Available {
		t.Message = "Keine kompatible GPU erkannt."
	} else {
		t.Message = fmt.Sprintf("%d GPU(s) aktiv.", len(allDevices))
	}

	m.mu.Lock()
	m.latest = t
	m.mu.Unlock()

	return t
}

func (m *Manager) enrichContainerProcesses(dm *DeviceMetrics) {
	for i := range dm.Processes {
		proc := &dm.Processes[i]
		containerID := m.findContainerByPID(proc.PID)
		if containerID != "" {
			proc.ContainerID = containerID
			if c, ok := m.inv.Get(containerID); ok {
				proc.ContainerName = c.Name
			}
		}

		// Try to read process name from /proc/<pid>/comm
		commPath := filepath.Join(m.cfg.HostProc, fmt.Sprintf("%d", proc.PID), "comm")
		if comm, err := os.ReadFile(commPath); err == nil {
			proc.ProcessName = strings.TrimSpace(string(comm))
		}
	}
}

func (m *Manager) findContainerByPID(pid uint32) string {
	cgroupPath := filepath.Join(m.cfg.HostProc, fmt.Sprintf("%d", pid), "cgroup")
	content, err := os.ReadFile(cgroupPath)
	if err != nil {
		cgroupPath = filepath.Join("/proc", fmt.Sprintf("%d", pid), "cgroup")
		content, err = os.ReadFile(cgroupPath)
		if err != nil {
			return ""
		}
	}

	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		line := scanner.Text()
		// Patterns:
		// .../docker/<id>
		// .../docker-<id>.scope
		if idx := strings.Index(line, "/docker/"); idx != -1 {
			id := line[idx+8:]
			if slash := strings.Index(id, "/"); slash != -1 {
				id = id[:slash]
			}
			return id
		}
		if idx := strings.Index(line, "docker-"); idx != -1 {
			id := line[idx+7:]
			if dot := strings.Index(id, ".scope"); dot != -1 {
				id = id[:dot]
			}
			return id
		}
	}
	return ""
}
