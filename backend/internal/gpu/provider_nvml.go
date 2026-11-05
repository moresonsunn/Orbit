//go:build linux && cgo

package gpu

import (
	"context"
	"fmt"
	"log/slog"
	"math"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"github.com/phili/orbit/internal/model"
)

type NVMLProvider struct {
	log       *slog.Logger
	available bool
	version   string
}

func NewNVMLProvider(log *slog.Logger) *NVMLProvider {
	ret := nvml.Init()
	if ret != nvml.SUCCESS {
		log.Debug("NVML not available", "ret", nvml.ErrorString(ret))
		return &NVMLProvider{log: log, available: false}
	}

	version, ret := nvml.SystemGetDriverVersion()
	if ret != nvml.SUCCESS {
		version = "unknown"
	}

	log.Info("NVIDIA NVML initialized successfully", "driverVersion", version)
	return &NVMLProvider{
		log:       log,
		available: true,
		version:   version,
	}
}

func (p *NVMLProvider) Name() string {
	return "nvidia"
}

func (p *NVMLProvider) Available() bool {
	return p.available
}

func (p *NVMLProvider) Shutdown() {
	if p.available {
		nvml.Shutdown()
		p.available = false
	}
}

func (p *NVMLProvider) Collect(ctx context.Context) ([]DeviceMetrics, error) {
	if !p.available {
		return nil, nil
	}

	count, ret := nvml.DeviceGetCount()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("nvml.DeviceGetCount failed: %s", nvml.ErrorString(ret))
	}

	var devices []DeviceMetrics

	for i := 0; i < count; i++ {
		dev, ret := nvml.DeviceGetHandleByIndex(i)
		if ret != nvml.SUCCESS {
			continue
		}

		name, ret := dev.GetName()
		if ret != nvml.SUCCESS {
			name = fmt.Sprintf("NVIDIA GPU #%d", i)
		}

		dm := DeviceMetrics{
			Index:         i,
			Model:         name,
			Vendor:        "nvidia",
			DriverVersion: p.version,
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

		// Utilization
		if rates, ret := dev.GetUtilizationRates(); ret == nvml.SUCCESS {
			dm.Utilization = model.Val(float64(rates.Gpu))
			dm.MemoryPercent = model.Val(float64(rates.Memory))
		}

		// Memory
		if mem, ret := dev.GetMemoryInfo(); ret == nvml.SUCCESS {
			dm.MemoryUsed = model.Val(float64(mem.Used))
			dm.MemoryTotal = model.Val(float64(mem.Total))
			if mem.Total > 0 {
				pct := math.Round(float64(mem.Used)/float64(mem.Total)*10000) / 100
				dm.MemoryPercent = model.Val(pct)
			}
		}

		// Temperature
		if temp, ret := dev.GetTemperature(nvml.TEMPERATURE_GPU); ret == nvml.SUCCESS {
			dm.Temperature = model.Val(float64(temp))
		}

		// Power
		if power, ret := dev.GetPowerUsage(); ret == nvml.SUCCESS {
			dm.PowerDraw = model.Val(float64(power) / 1000.0) // milliwatts to watts
		}
		if powerLimit, ret := dev.GetEnforcedPowerLimit(); ret == nvml.SUCCESS {
			dm.PowerLimit = model.Val(float64(powerLimit) / 1000.0)
		}

		// Clocks
		if clockGpu, ret := dev.GetClockInfo(nvml.CLOCK_GRAPHICS); ret == nvml.SUCCESS {
			dm.ClockGraphics = model.Val(float64(clockGpu))
		}
		if clockMem, ret := dev.GetClockInfo(nvml.CLOCK_MEM); ret == nvml.SUCCESS {
			dm.ClockMemory = model.Val(float64(clockMem))
		}

		// Encoder / Decoder
		if encUtil, _, ret := dev.GetEncoderUtilization(); ret == nvml.SUCCESS {
			dm.EncoderUtil = model.Val(float64(encUtil))
		}
		if decUtil, _, ret := dev.GetDecoderUtilization(); ret == nvml.SUCCESS {
			dm.DecoderUtil = model.Val(float64(decUtil))
		}

		// Running processes
		if procs, ret := dev.GetComputeRunningProcesses(); ret == nvml.SUCCESS {
			for _, proc := range procs {
				dm.Processes = append(dm.Processes, GPUProcessInfo{
					PID:        proc.Pid,
					UsedMemory: model.Val(float64(proc.UsedGpuMemory)),
				})
			}
		}
		if procs, ret := dev.GetGraphicsRunningProcesses(); ret == nvml.SUCCESS {
			seen := make(map[uint32]bool)
			for _, p := range dm.Processes {
				seen[p.PID] = true
			}
			for _, proc := range procs {
				if !seen[proc.Pid] {
					dm.Processes = append(dm.Processes, GPUProcessInfo{
						PID:        proc.Pid,
						UsedMemory: model.Val(float64(proc.UsedGpuMemory)),
					})
				}
			}
		}

		devices = append(devices, dm)
	}

	return devices, nil
}
