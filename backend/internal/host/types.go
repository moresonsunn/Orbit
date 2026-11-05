package host

import (
	"time"

	"github.com/phili/orbit/internal/model"
)

type HostMetrics struct {
	Timestamp time.Time      `json:"timestamp"`
	CPU       CPUMetrics     `json:"cpu"`
	Memory    MemoryMetrics  `json:"memory"`
	Swap      SwapMetrics    `json:"swap"`
	Storage   StorageMetrics `json:"storage"`
	Network   NetworkMetrics `json:"network"`
	System    SystemInfo     `json:"system"`
}

type CPUMetrics struct {
	TotalPercent model.Gauge   `json:"totalPercent"`
	Cores        int           `json:"cores"`
	Threads      int           `json:"threads"`
	PerCore      []model.Gauge `json:"perCore"`
	FreqMHz      model.Gauge   `json:"freqMHz"`
	TempC        model.Gauge   `json:"tempC"`
	LoadAvg1     model.Gauge   `json:"loadAvg1"`
	LoadAvg5     model.Gauge   `json:"loadAvg5"`
	LoadAvg15    model.Gauge   `json:"loadAvg15"`
}

type MemoryMetrics struct {
	TotalBytes     uint64      `json:"totalBytes"`
	UsedBytes      uint64      `json:"usedBytes"`
	FreeBytes      uint64      `json:"freeBytes"`
	AvailableBytes uint64      `json:"availableBytes"`
	BuffersBytes   uint64      `json:"buffersBytes"`
	CachedBytes    uint64      `json:"cachedBytes"`
	UsedPercent    model.Gauge `json:"usedPercent"`
}

type SwapMetrics struct {
	TotalBytes  uint64      `json:"totalBytes"`
	UsedBytes   uint64      `json:"usedBytes"`
	FreeBytes   uint64      `json:"freeBytes"`
	UsedPercent model.Gauge `json:"usedPercent"`
}

type DiskMount struct {
	MountPoint   string      `json:"mountPoint"`
	Device       string      `json:"device"`
	FSType       string      `json:"fsType"`
	TotalBytes   uint64      `json:"totalBytes"`
	UsedBytes    uint64      `json:"usedBytes"`
	FreeBytes    uint64      `json:"freeBytes"`
	UsedPercent  model.Gauge `json:"usedPercent"`
	ReadBps      model.Gauge `json:"readBps"`
	WriteBps     model.Gauge `json:"writeBps"`
}

type StorageMetrics struct {
	TotalBytes  uint64      `json:"totalBytes"`
	UsedBytes   uint64      `json:"usedBytes"`
	FreeBytes   uint64      `json:"freeBytes"`
	UsedPercent model.Gauge `json:"usedPercent"`
	ReadBps     model.Gauge `json:"readBps"`
	WriteBps    model.Gauge `json:"writeBps"`
	Mounts      []DiskMount `json:"mounts"`
}

type InterfaceMetrics struct {
	Name      string      `json:"name"`
	RxBytes   uint64      `json:"rxBytes"`
	TxBytes   uint64      `json:"txBytes"`
	RxPackets uint64      `json:"rxPackets"`
	TxPackets uint64      `json:"txPackets"`
	RxErrors  uint64      `json:"rxErrors"`
	TxErrors  uint64      `json:"txErrors"`
	RxDropped uint64      `json:"rxDropped"`
	TxDropped uint64      `json:"txDropped"`
	RxBps     model.Gauge `json:"rxBps"`
	TxBps     model.Gauge `json:"txBps"`
	RxPps     model.Gauge `json:"rxPps"`
	TxPps     model.Gauge `json:"txPps"`
}

type NetworkMetrics struct {
	RxBps      model.Gauge        `json:"rxBps"`
	TxBps      model.Gauge        `json:"txBps"`
	TotalRx    uint64             `json:"totalRxBytes"`
	TotalTx    uint64             `json:"totalTxBytes"`
	Interfaces []InterfaceMetrics `json:"interfaces"`
}

type SystemInfo struct {
	Hostname      string      `json:"hostname"`
	OS            string      `json:"os"`
	KernelVersion string      `json:"kernelVersion"`
	Architecture  string      `json:"architecture"`
	UptimeSeconds model.Gauge `json:"uptimeSeconds"`
}

