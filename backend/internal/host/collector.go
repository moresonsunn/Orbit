package host

import (
	"bufio"
	"context"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/phili/orbit/internal/config"
	"github.com/phili/orbit/internal/model"
)

type cpuTimeSample struct {
	idle  uint64
	total uint64
}

type ifaceSample struct {
	rxBytes   uint64
	txBytes   uint64
	rxPackets uint64
	txPackets uint64
}

type diskSample struct {
	readBytes  uint64
	writeBytes uint64
}

type Collector struct {
	cfg *config.Config
	log *slog.Logger

	mu             sync.Mutex
	prevTime       time.Time
	prevTotalCPU   cpuTimeSample
	prevCoreCPUs   []cpuTimeSample
	prevIfaces     map[string]ifaceSample
	prevDisks      map[string]diskSample
	cachedHostname  string
	cachedOS        string
	cachedKernel    string
	cachedModelName string
	cachedCores     int
	cachedThreads   int
}

func NewCollector(cfg *config.Config, log *slog.Logger) *Collector {
	c := &Collector{
		cfg:        cfg,
		log:        log,
		prevIfaces: make(map[string]ifaceSample),
		prevDisks:  make(map[string]diskSample),
	}
	c.initStaticInfo()
	return c
}

func (c *Collector) procPath(rel string) string {
	if fi, err := os.Stat(c.cfg.HostProc); err == nil && fi.IsDir() {
		return filepath.Join(c.cfg.HostProc, rel)
	}
	return filepath.Join("/proc", rel)
}

func (c *Collector) sysPath(rel string) string {
	if fi, err := os.Stat(c.cfg.HostSys); err == nil && fi.IsDir() {
		return filepath.Join(c.cfg.HostSys, rel)
	}
	return filepath.Join("/sys", rel)
}

func (c *Collector) etcPath(rel string) string {
	if fi, err := os.Stat(c.cfg.HostEtc); err == nil && fi.IsDir() {
		return filepath.Join(c.cfg.HostEtc, rel)
	}
	return filepath.Join("/etc", rel)
}

func (c *Collector) initStaticInfo() {
	if h, err := os.Hostname(); err == nil {
		c.cachedHostname = h
	}

	// Try reading /etc/os-release
	c.cachedOS = runtime.GOOS
	if content, err := os.ReadFile(c.etcPath("os-release")); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(content)))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				c.cachedOS = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), `"`)
				break
			}
		}
	}

	// Try reading /proc/version
	c.cachedKernel = runtime.Version()
	if content, err := os.ReadFile(c.procPath("version")); err == nil {
		parts := strings.Fields(string(content))
		if len(parts) >= 3 {
			c.cachedKernel = parts[2]
		}
	}

	c.cachedThreads = runtime.NumCPU()
	c.cachedCores = c.cachedThreads

	// Try counting physical cores and extracting model name from /proc/cpuinfo
	if content, err := os.ReadFile(c.procPath("cpuinfo")); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(content)))
		coresMap := make(map[string]bool)
		var curPhysical, curCore string
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if c.cachedModelName == "" {
				if strings.HasPrefix(line, "model name") || strings.HasPrefix(line, "Processor") || strings.HasPrefix(line, "Hardware") {
					parts := strings.SplitN(line, ":", 2)
					if len(parts) == 2 {
						c.cachedModelName = strings.TrimSpace(parts[1])
					}
				}
			}
			if strings.HasPrefix(line, "physical id") {
				curPhysical = line
			} else if strings.HasPrefix(line, "core id") {
				curCore = line
				if curPhysical != "" && curCore != "" {
					coresMap[curPhysical+":"+curCore] = true
				}
			}
		}
		if len(coresMap) > 0 {
			c.cachedCores = len(coresMap)
		}
	}
}

func (c *Collector) Collect(ctx context.Context) HostMetrics {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	dt := 1.0
	if !c.prevTime.IsZero() {
		dt = now.Sub(c.prevTime).Seconds()
		if dt <= 0.01 {
			dt = 1.0
		}
	}

	thermal := c.collectThermal()
	cpu := c.collectCPU(dt)
	if thermal.CPUTemp.Status == model.OK {
		cpu.TempC = thermal.CPUTemp
	}

	metrics := HostMetrics{
		Timestamp: now,
		CPU:       cpu,
		Memory:    c.collectMemory(),
		Swap:      c.collectSwap(),
		Storage:   c.collectStorage(dt),
		Network:   c.collectNetwork(dt),
		Thermal:   thermal,
		System:    c.collectSystem(),
	}

	c.prevTime = now
	return metrics
}

func (c *Collector) collectCPU(dt float64) CPUMetrics {
	m := CPUMetrics{
		ModelName:    c.cachedModelName,
		Cores:        c.cachedCores,
		Threads:      c.cachedThreads,
		TotalPercent: model.NA(model.Unavailable, "no cpu stats available"),
		FreqMHz:      model.NA(model.Unavailable, "cpu frequency not available"),
		TempC:        model.NA(model.Unavailable, "temperature sensor not available"),
		LoadAvg1:     model.NA(model.Unavailable, "load average not available"),
		LoadAvg5:     model.NA(model.Unavailable, "load average not available"),
		LoadAvg15:    model.NA(model.Unavailable, "load average not available"),
	}

	// 1. Read /proc/stat
	statPath := c.procPath("stat")
	if content, err := os.ReadFile(statPath); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(content)))
		var coreSamples []cpuTimeSample

		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "cpu ") {
				sample, ok := parseCPUFields(line[4:])
				if ok {
					if c.prevTotalCPU.total > 0 && sample.total > c.prevTotalCPU.total {
						deltaTotal := float64(sample.total - c.prevTotalCPU.total)
						deltaIdle := float64(sample.idle - c.prevTotalCPU.idle)
						usage := math.Max(0, math.Min(100, (1.0-(deltaIdle/deltaTotal))*100.0))
						m.TotalPercent = model.Val(round2(usage))
					} else {
						m.TotalPercent = model.NA(model.Pending, "awaiting second sample")
					}
					c.prevTotalCPU = sample
				}
			} else if strings.HasPrefix(line, "cpu") && len(line) > 3 && line[3] >= '0' && line[3] <= '9' {
				fields := strings.Fields(line)
				if len(fields) >= 5 {
					sample, ok := parseCPUFields(strings.Join(fields[1:], " "))
					if ok {
						coreSamples = append(coreSamples, sample)
					}
				}
			}
		}

		if len(coreSamples) > 0 {
			m.PerCore = make([]model.Gauge, len(coreSamples))
			for i, sample := range coreSamples {
				if i < len(c.prevCoreCPUs) && c.prevCoreCPUs[i].total > 0 && sample.total > c.prevCoreCPUs[i].total {
					deltaTotal := float64(sample.total - c.prevCoreCPUs[i].total)
					deltaIdle := float64(sample.idle - c.prevCoreCPUs[i].idle)
					usage := math.Max(0, math.Min(100, (1.0-(deltaIdle/deltaTotal))*100.0))
					m.PerCore[i] = model.Val(round2(usage))
				} else {
					m.PerCore[i] = model.NA(model.Pending, "awaiting second sample")
				}
			}
			c.prevCoreCPUs = coreSamples
		}
	}

	// 2. Read /proc/loadavg
	if content, err := os.ReadFile(c.procPath("loadavg")); err == nil {
		fields := strings.Fields(string(content))
		if len(fields) >= 3 {
			if l1, err := strconv.ParseFloat(fields[0], 64); err == nil {
				m.LoadAvg1 = model.Val(round2(l1))
			}
			if l5, err := strconv.ParseFloat(fields[1], 64); err == nil {
				m.LoadAvg5 = model.Val(round2(l5))
			}
			if l15, err := strconv.ParseFloat(fields[2], 64); err == nil {
				m.LoadAvg15 = model.Val(round2(l15))
			}
		}
	}

	// 3. Read CPU Freq from /sys/devices/system/cpu/cpu0/cpufreq/scaling_cur_freq
	freqPath := c.sysPath("devices/system/cpu/cpu0/cpufreq/scaling_cur_freq")
	if content, err := os.ReadFile(freqPath); err == nil {
		if khz, err := strconv.ParseFloat(strings.TrimSpace(string(content)), 64); err == nil && khz > 0 {
			m.FreqMHz = model.Val(round2(khz / 1000.0))
		}
	} else {
		// Fallback for containers/WSL2: read "cpu MHz" from /proc/cpuinfo
		if content, err := os.ReadFile(c.procPath("cpuinfo")); err == nil {
			scanner := bufio.NewScanner(strings.NewReader(string(content)))
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if strings.HasPrefix(line, "cpu MHz") {
					parts := strings.SplitN(line, ":", 2)
					if len(parts) == 2 {
						if mhz, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil && mhz > 0 {
							m.FreqMHz = model.Val(round2(mhz))
							break
						}
					}
				}
			}
		}
	}

	// 4. CPU Temperature is populated by collectThermal() in Collect()
	return m
}

func parseCPUFields(raw string) (cpuTimeSample, bool) {
	fields := strings.Fields(raw)
	if len(fields) < 4 {
		return cpuTimeSample{}, false
	}
	var total, idle uint64
	for i, f := range fields {
		v, err := strconv.ParseUint(f, 10, 64)
		if err != nil {
			return cpuTimeSample{}, false
		}
		total += v
		if i == 3 { // idle field
			idle = v
		} else if i == 4 { // iowait field (often considered idle or wait)
			idle += v
		}
	}
	return cpuTimeSample{idle: idle, total: total}, true
}

func (c *Collector) collectThermal() ThermalMetrics {
	tm := ThermalMetrics{
		CPUTemp: model.NA(model.Unavailable, "no cpu thermal sensor detected"),
		MaxTemp: model.NA(model.Unavailable, "no thermal sensor detected"),
		Sensors: []ThermalSensor{},
	}

	var maxVal float64 = -1
	var cpuFound bool

	// 1. Scan Linux hwmon subsystem: /sys/class/hwmon/hwmon*
	hwmonBase := c.sysPath("class/hwmon")
	if entries, err := os.ReadDir(hwmonBase); err == nil {
		for _, entry := range entries {
			devDir := filepath.Join(hwmonBase, entry.Name())
			devName := entry.Name()
			if nameContent, err := os.ReadFile(filepath.Join(devDir, "name")); err == nil {
				devName = strings.TrimSpace(string(nameContent))
			}

			lowerName := strings.ToLower(devName)
			var sensorType string
			if strings.Contains(lowerName, "k10temp") || strings.Contains(lowerName, "zenpower") || strings.Contains(lowerName, "coretemp") || strings.Contains(lowerName, "cpu") {
				sensorType = "cpu"
			} else if strings.Contains(lowerName, "nvme") {
				sensorType = "nvme"
			} else if strings.Contains(lowerName, "acpitz") || strings.Contains(lowerName, "pch") || strings.Contains(lowerName, "it87") || strings.Contains(lowerName, "nct") {
				sensorType = "motherboard"
			} else {
				sensorType = "system"
			}

			files, err := os.ReadDir(devDir)
			if err != nil {
				continue
			}

			for _, file := range files {
				fileName := file.Name()
				if strings.HasPrefix(fileName, "temp") && strings.HasSuffix(fileName, "_input") {
					prefix := strings.TrimSuffix(fileName, "_input") // e.g. "temp1"
					inputPath := filepath.Join(devDir, fileName)

					valBytes, err := os.ReadFile(inputPath)
					if err != nil {
						continue
					}

					milliC, err := strconv.ParseFloat(strings.TrimSpace(string(valBytes)), 64)
					if err != nil || milliC <= 0 || milliC > 150000 {
						continue
					}

					tempC := round2(milliC / 1000.0)

					// Read label if available
					label := devName
					if labelBytes, err := os.ReadFile(filepath.Join(devDir, prefix+"_label")); err == nil {
						l := strings.TrimSpace(string(labelBytes))
						if l != "" {
							label = l
						}
					}

					// Read crit if available
					var critGauge model.Gauge
					if critBytes, err := os.ReadFile(filepath.Join(devDir, prefix+"_crit")); err == nil {
						if critMilli, err := strconv.ParseFloat(strings.TrimSpace(string(critBytes)), 64); err == nil && critMilli > 0 {
							critGauge = model.Val(round2(critMilli / 1000.0))
						} else {
							critGauge = model.NA(model.Unavailable, "no crit threshold")
						}
					} else {
						critGauge = model.NA(model.Unavailable, "no crit threshold")
					}

					sensor := ThermalSensor{
						Name:       devName,
						Label:      label,
						Device:     entry.Name(),
						TempC:      model.Val(tempC),
						CritC:      critGauge,
						SensorType: sensorType,
					}
					tm.Sensors = append(tm.Sensors, sensor)

					if tempC > maxVal {
						maxVal = tempC
						tm.MaxTemp = model.Val(maxVal)
					}

					if sensorType == "cpu" && (!cpuFound || strings.Contains(strings.ToLower(label), "package") || strings.Contains(strings.ToLower(label), "tctl")) {
						tm.CPUTemp = model.Val(tempC)
						cpuFound = true
					}
				}
			}
		}
	}

	// 2. Fallback to /sys/class/thermal/thermal_zone* (Common on ARM/Raspberry Pi/Debian)
	if len(tm.Sensors) == 0 {
		thermalBase := c.sysPath("class/thermal")
		if tzEntries, err := os.ReadDir(thermalBase); err == nil {
			for _, tz := range tzEntries {
				if strings.HasPrefix(tz.Name(), "thermal_zone") {
					tzDir := filepath.Join(thermalBase, tz.Name())
					typeStr := tz.Name()
					if typeBytes, err := os.ReadFile(filepath.Join(tzDir, "type")); err == nil {
						t := strings.TrimSpace(string(typeBytes))
						if t != "" {
							typeStr = t
						}
					}

					tempBytes, err := os.ReadFile(filepath.Join(tzDir, "temp"))
					if err != nil {
						continue
					}

					milliC, err := strconv.ParseFloat(strings.TrimSpace(string(tempBytes)), 64)
					if err != nil || milliC <= 0 || milliC > 150000 {
						continue
					}

					tempC := round2(milliC / 1000.0)
					sensorType := "system"
					if strings.Contains(strings.ToLower(typeStr), "cpu") || strings.Contains(strings.ToLower(typeStr), "x86_pkg") {
						sensorType = "cpu"
					}

					sensor := ThermalSensor{
						Name:       typeStr,
						Label:      typeStr,
						Device:     tz.Name(),
						TempC:      model.Val(tempC),
						CritC:      model.NA(model.Unavailable, "no crit threshold"),
						SensorType: sensorType,
					}
					tm.Sensors = append(tm.Sensors, sensor)

					if tempC > maxVal {
						maxVal = tempC
						tm.MaxTemp = model.Val(maxVal)
					}

					if sensorType == "cpu" || !cpuFound {
						tm.CPUTemp = model.Val(tempC)
						cpuFound = true
					}
				}
			}
		}
	}

	return tm
}

func (c *Collector) collectMemory() MemoryMetrics {
	m := MemoryMetrics{
		UsedPercent: model.NA(model.Unavailable, "no memory stats available"),
	}

	content, err := os.ReadFile(c.procPath("meminfo"))
	if err != nil {
		return m
	}

	mem := make(map[string]uint64)
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			vStr := strings.TrimSpace(strings.TrimSuffix(parts[1], " kB"))
			fields := strings.Fields(vStr)
			if len(fields) > 0 {
				if val, err := strconv.ParseUint(fields[0], 10, 64); err == nil {
					mem[k] = val * 1024 // convert kB to bytes
				}
			}
		}
	}

	m.TotalBytes = mem["MemTotal"]
	m.FreeBytes = mem["MemFree"]
	m.BuffersBytes = mem["Buffers"]
	m.CachedBytes = mem["Cached"] + mem["SReclaimable"]

	if avail, has := mem["MemAvailable"]; has {
		m.AvailableBytes = avail
		if m.TotalBytes > 0 && avail <= m.TotalBytes {
			m.UsedBytes = m.TotalBytes - avail
			m.UsedPercent = model.Val(round2(float64(m.UsedBytes) / float64(m.TotalBytes) * 100.0))
		}
	} else if m.TotalBytes > 0 {
		m.UsedBytes = m.TotalBytes - m.FreeBytes - m.BuffersBytes - m.CachedBytes
		m.AvailableBytes = m.FreeBytes + m.BuffersBytes + m.CachedBytes
		m.UsedPercent = model.Val(round2(float64(m.UsedBytes) / float64(m.TotalBytes) * 100.0))
	}

	return m
}

func (c *Collector) collectSwap() SwapMetrics {
	m := SwapMetrics{
		UsedPercent: model.NA(model.Unavailable, "no swap stats available"),
	}

	content, err := os.ReadFile(c.procPath("meminfo"))
	if err != nil {
		return m
	}

	mem := make(map[string]uint64)
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			vStr := strings.TrimSpace(strings.TrimSuffix(parts[1], " kB"))
			fields := strings.Fields(vStr)
			if len(fields) > 0 {
				if val, err := strconv.ParseUint(fields[0], 10, 64); err == nil {
					mem[k] = val * 1024
				}
			}
		}
	}

	m.TotalBytes = mem["SwapTotal"]
	m.FreeBytes = mem["SwapFree"]
	if m.TotalBytes > 0 {
		m.UsedBytes = m.TotalBytes - m.FreeBytes
		m.UsedPercent = model.Val(round2(float64(m.UsedBytes) / float64(m.TotalBytes) * 100.0))
	} else {
		m.UsedPercent = model.NA(model.OK, "no swap configured")
	}

	return m
}

func (c *Collector) collectNetwork(dt float64) NetworkMetrics {
	m := NetworkMetrics{
		RxBps: model.NA(model.Pending, "awaiting second sample"),
		TxBps: model.NA(model.Pending, "awaiting second sample"),
	}

	content, err := os.ReadFile(c.procPath("net/dev"))
	if err != nil {
		m.RxBps = model.NA(model.Unavailable, "net/dev not accessible")
		m.TxBps = model.NA(model.Unavailable, "net/dev not accessible")
		return m
	}

	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	var totalRxBps, totalTxBps float64
	var totalRxBytes, totalTxBytes uint64
	hasRate := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.Contains(line, ":") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		ifaceName := strings.TrimSpace(parts[0])
		fields := strings.Fields(parts[1])
		if len(fields) < 16 {
			continue
		}

		rxBytes, _ := strconv.ParseUint(fields[0], 10, 64)
		rxPackets, _ := strconv.ParseUint(fields[1], 10, 64)
		rxErrors, _ := strconv.ParseUint(fields[2], 10, 64)
		rxDropped, _ := strconv.ParseUint(fields[3], 10, 64)

		txBytes, _ := strconv.ParseUint(fields[8], 10, 64)
		txPackets, _ := strconv.ParseUint(fields[9], 10, 64)
		txErrors, _ := strconv.ParseUint(fields[10], 10, 64)
		txDropped, _ := strconv.ParseUint(fields[11], 10, 64)

		im := InterfaceMetrics{
			Name:      ifaceName,
			RxBytes:   rxBytes,
			TxBytes:   txBytes,
			RxPackets: rxPackets,
			TxPackets: txPackets,
			RxErrors:  rxErrors,
			TxErrors:  txErrors,
			RxDropped: rxDropped,
			TxDropped: txDropped,
			RxBps:     model.NA(model.Pending, "awaiting second sample"),
			TxBps:     model.NA(model.Pending, "awaiting second sample"),
			RxPps:     model.NA(model.Pending, "awaiting second sample"),
			TxPps:     model.NA(model.Pending, "awaiting second sample"),
		}

		if prev, ok := c.prevIfaces[ifaceName]; ok && dt > 0 {
			if rxBytes >= prev.rxBytes {
				rxBps := round2(float64(rxBytes-prev.rxBytes) / dt)
				im.RxBps = model.Val(rxBps)
				if ifaceName != "lo" {
					totalRxBps += rxBps
				}
			}
			if txBytes >= prev.txBytes {
				txBps := round2(float64(txBytes-prev.txBytes) / dt)
				im.TxBps = model.Val(txBps)
				if ifaceName != "lo" {
					totalTxBps += txBps
				}
			}
			if rxPackets >= prev.rxPackets {
				im.RxPps = model.Val(round2(float64(rxPackets-prev.rxPackets) / dt))
			}
			if txPackets >= prev.txPackets {
				im.TxPps = model.Val(round2(float64(txPackets-prev.txPackets) / dt))
			}
			hasRate = true
		}

		c.prevIfaces[ifaceName] = ifaceSample{
			rxBytes:   rxBytes,
			txBytes:   txBytes,
			rxPackets: rxPackets,
			txPackets: txPackets,
		}

		if ifaceName != "lo" {
			totalRxBytes += rxBytes
			totalTxBytes += txBytes
		}

		m.Interfaces = append(m.Interfaces, im)
	}

	m.TotalRx = totalRxBytes
	m.TotalTx = totalTxBytes

	if hasRate {
		m.RxBps = model.Val(totalRxBps)
		m.TxBps = model.Val(totalTxBps)
	}

	return m
}

func (c *Collector) collectStorage(dt float64) StorageMetrics {
	m := StorageMetrics{
		UsedPercent: model.NA(model.Unavailable, "no storage info available"),
		ReadBps:     model.NA(model.Pending, "awaiting second sample"),
		WriteBps:    model.NA(model.Pending, "awaiting second sample"),
	}

	// Read /proc/diskstats for read/write I/O
	if content, err := os.ReadFile(c.procPath("diskstats")); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(content)))
		var totalReadBps, totalWriteBps float64
		hasRate := false

		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) < 14 {
				continue
			}
			devName := fields[2]
			// Filter to physical disks (sdX, nvmeXnY, vdX, mmcblkX)
			if !isPhysicalDisk(devName) {
				continue
			}

			// Field 5: sectors read (sector = 512 bytes)
			sectorsRead, _ := strconv.ParseUint(fields[5], 10, 64)
			// Field 9: sectors written
			sectorsWritten, _ := strconv.ParseUint(fields[9], 10, 64)

			readBytes := sectorsRead * 512
			writeBytes := sectorsWritten * 512

			if prev, ok := c.prevDisks[devName]; ok && dt > 0 {
				if readBytes >= prev.readBytes {
					totalReadBps += float64(readBytes-prev.readBytes) / dt
				}
				if writeBytes >= prev.writeBytes {
					totalWriteBps += float64(writeBytes-prev.writeBytes) / dt
				}
				hasRate = true
			}

			c.prevDisks[devName] = diskSample{
				readBytes:  readBytes,
				writeBytes: writeBytes,
			}
		}

		if hasRate {
			m.ReadBps = model.Val(round2(totalReadBps))
			m.WriteBps = model.Val(round2(totalWriteBps))
		}
	}

	// Detect mounts using /proc/mounts or mountinfo
	mounts := c.detectMounts()
	m.Mounts = mounts

	for _, mount := range mounts {
		m.TotalBytes += mount.TotalBytes
		m.UsedBytes += mount.UsedBytes
		m.FreeBytes += mount.FreeBytes
	}

	if m.TotalBytes > 0 {
		m.UsedPercent = model.Val(round2(float64(m.UsedBytes) / float64(m.TotalBytes) * 100.0))
	}

	return m
}

func isPhysicalDisk(name string) bool {
	if strings.HasPrefix(name, "sd") && len(name) == 3 { // sda, sdb
		return true
	}
	if strings.HasPrefix(name, "vd") && len(name) == 3 { // vda, vdb
		return true
	}
	if strings.HasPrefix(name, "nvme") && strings.Contains(name, "n1") && !strings.Contains(name, "p") { // nvme0n1
		return true
	}
	return false
}

func (c *Collector) collectSystem() SystemInfo {
	s := SystemInfo{
		Hostname:      c.cachedHostname,
		OS:            c.cachedOS,
		KernelVersion: c.cachedKernel,
		Architecture:  runtime.GOARCH,
		UptimeSeconds: model.NA(model.Unavailable, "uptime not available"),
	}

	if content, err := os.ReadFile(c.procPath("uptime")); err == nil {
		fields := strings.Fields(string(content))
		if len(fields) > 0 {
			if up, err := strconv.ParseFloat(fields[0], 64); err == nil {
				s.UptimeSeconds = model.Val(round2(up))
			}
		}
	}

	return s
}
