package gpu

import (
	"context"
	"time"

	"github.com/phili/orbit/internal/model"
)

type GPUProcessInfo struct {
	PID           uint32      `json:"pid"`
	ContainerID   string      `json:"containerId,omitempty"`
	ContainerName string      `json:"containerName,omitempty"`
	ProcessName   string      `json:"processName"`
	UsedMemory    model.Gauge `json:"usedMemoryBytes"`
}

type DeviceMetrics struct {
	Index         int              `json:"index"`
	Model         string           `json:"model"`
	Vendor        string           `json:"vendor"` // nvidia | amd | intel | other
	DriverVersion string           `json:"driverVersion"`
	Utilization   model.Gauge      `json:"utilizationPercent"`
	MemoryUsed    model.Gauge      `json:"memoryUsedBytes"`
	MemoryTotal   model.Gauge      `json:"memoryTotalBytes"`
	MemoryPercent model.Gauge      `json:"memoryPercent"`
	Temperature   model.Gauge      `json:"temperatureCelsius"`
	PowerDraw     model.Gauge      `json:"powerDrawWatts"`
	PowerLimit    model.Gauge      `json:"powerLimitWatts"`
	ClockGraphics model.Gauge      `json:"clockGraphicsMHz"`
	ClockMemory   model.Gauge      `json:"clockMemoryMHz"`
	EncoderUtil   model.Gauge      `json:"encoderUtilPercent"`
	DecoderUtil   model.Gauge      `json:"decoderUtilPercent"`
	Processes     []GPUProcessInfo `json:"processes"`
}

type Telemetry struct {
	Timestamp time.Time       `json:"timestamp"`
	Available bool            `json:"available"`
	Message   string          `json:"message"`
	Devices   []DeviceMetrics `json:"devices"`
}

type Provider interface {
	Name() string
	Available() bool
	Collect(ctx context.Context) ([]DeviceMetrics, error)
	Shutdown()
}

