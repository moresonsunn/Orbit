package gpu

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/phili/orbit/internal/config"
	"github.com/phili/orbit/internal/model"
)

type DRMProvider struct {
	cfg       *config.Config
	log       *slog.Logger
	cards     []string
	available bool
}

func NewDRMProvider(cfg *config.Config, log *slog.Logger) *DRMProvider {
	drmPath := filepath.Join(cfg.HostSys, "class", "drm")
	if fi, err := os.Stat(drmPath); err != nil || !fi.IsDir() {
		drmPath = "/sys/class/drm"
	}

	entries, err := os.ReadDir(drmPath)
	if err != nil {
		return &DRMProvider{cfg: cfg, log: log, available: false}
	}

	var cards []string
	for _, entry := range entries {
		name := entry.Name()
		// Only primary render cards like card0, card1 (skip card0-DP-1 etc)
		if strings.HasPrefix(name, "card") && !strings.Contains(name, "-") {
			devicePath := filepath.Join(drmPath, name, "device")
			if fi, err := os.Stat(devicePath); err == nil && fi.IsDir() {
				cards = append(cards, devicePath)
			}
		}
	}

	return &DRMProvider{
		cfg:       cfg,
		log:       log,
		cards:     cards,
		available: len(cards) > 0,
	}
}

func (p *DRMProvider) Name() string {
	return "drm"
}

func (p *DRMProvider) Available() bool {
	return p.available
}

func (p *DRMProvider) Shutdown() {}

func (p *DRMProvider) Collect(ctx context.Context) ([]DeviceMetrics, error) {
	if !p.available {
		return nil, nil
	}

	var devices []DeviceMetrics

	for i, devPath := range p.cards {
		vendor := "other"
		modelName := fmt.Sprintf("GPU Card #%d", i)

		// Read vendor ID
		if content, err := os.ReadFile(filepath.Join(devPath, "vendor")); err == nil {
			vID := strings.TrimSpace(string(content))
			switch vID {
			case "0x1002":
				vendor = "amd"
				modelName = "AMD Radeon GPU"
			case "0x8086":
				vendor = "intel"
				modelName = "Intel Graphics"
			case "0x10de":
				// NVIDIA is usually handled by NVML, but mark if seen here
				vendor = "nvidia"
				modelName = "NVIDIA GPU"
			}
		}

		dm := DeviceMetrics{
			Index:         i,
			Model:         modelName,
			Vendor:        vendor,
			DriverVersion: "drm/kernel",
			Utilization:   model.NA(model.Unavailable, "utilization unavailable"),
			MemoryUsed:    model.NA(model.Unavailable, "memory unavailable"),
			MemoryTotal:   model.NA(model.Unavailable, "memory unavailable"),
			MemoryPercent: model.NA(model.Unavailable, "memory unavailable"),
			Temperature:   model.NA(model.Unavailable, "temperature unavailable"),
			PowerDraw:     model.NA(model.Unavailable, "power draw unavailable"),
			PowerLimit:    model.NA(model.Unavailable, "power limit unavailable"),
			ClockGraphics: model.NA(model.Unavailable, "graphics clock unavailable"),
			ClockMemory:   model.NA(model.Unavailable, "memory clock unavailable"),
			EncoderUtil:   model.NA(model.Unavailable, "encoder utilization unavailable"),
			DecoderUtil:   model.NA(model.Unavailable, "decoder utilization unavailable"),
		}

		// Check gpu_busy_percent (AMD)
		if content, err := os.ReadFile(filepath.Join(devPath, "gpu_busy_percent")); err == nil {
			if util, err := strconv.ParseFloat(strings.TrimSpace(string(content)), 64); err == nil {
				dm.Utilization = model.Val(util)
			}
		}

		// Check VRAM (AMD)
		if content, err := os.ReadFile(filepath.Join(devPath, "mem_info_vram_total")); err == nil {
			if tot, err := strconv.ParseUint(strings.TrimSpace(string(content)), 10, 64); err == nil && tot > 0 {
				dm.MemoryTotal = model.Val(float64(tot))
				if usedContent, err := os.ReadFile(filepath.Join(devPath, "mem_info_vram_used")); err == nil {
					if used, err := strconv.ParseUint(strings.TrimSpace(string(usedContent)), 10, 64); err == nil {
						dm.MemoryUsed = model.Val(float64(used))
						pct := mathRound2(float64(used) / float64(tot) * 100.0)
						dm.MemoryPercent = model.Val(pct)
					}
				}
			}
		}

		// Check hwmon inside device for temperature & power
		hwmonDir := filepath.Join(devPath, "hwmon")
		if hwEntries, err := os.ReadDir(hwmonDir); err == nil {
			for _, hw := range hwEntries {
				hwPath := filepath.Join(hwmonDir, hw.Name())
				if tempBytes, err := os.ReadFile(filepath.Join(hwPath, "temp1_input")); err == nil {
					if milliC, err := strconv.ParseFloat(strings.TrimSpace(string(tempBytes)), 64); err == nil {
						dm.Temperature = model.Val(mathRound2(milliC / 1000.0))
					}
				}
				if pwrBytes, err := os.ReadFile(filepath.Join(hwPath, "power1_average")); err == nil {
					if microW, err := strconv.ParseFloat(strings.TrimSpace(string(pwrBytes)), 64); err == nil {
						dm.PowerDraw = model.Val(mathRound2(microW / 1_000_000.0))
					}
				}
			}
		}

		devices = append(devices, dm)
	}

	return devices, nil
}

func mathRound2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}
