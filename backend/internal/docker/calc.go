package docker

import (
	"strings"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/phili/orbit/internal/model"
)

// ContainerStats is the live metric snapshot for one container.
type ContainerStats struct {
	ID string `json:"id"`
	TS int64  `json:"ts"` // unix millis of the docker sample

	// CPUPercent uses `docker stats` semantics: 100 % = one fully used core,
	// maximum = OnlineCPUs * 100.
	CPUPercent model.Gauge `json:"cpuPercent"`
	// CPUPercentOfLimit normalizes CPUPercent to the container's CPU limit
	// (or to all online CPUs when unlimited): 0..100.
	CPUPercentOfLimit model.Gauge `json:"cpuPercentOfLimit"`
	OnlineCPUs        uint32      `json:"onlineCpus"`
	CPUThrottledPct   model.Gauge `json:"cpuThrottledPercent"`

	MemUsage   model.Gauge `json:"memUsage"`   // bytes, excluding inactive file cache (docker stats semantics)
	MemLimit   model.Gauge `json:"memLimit"`   // bytes (host total when unlimited)
	MemPercent model.Gauge `json:"memPercent"` // 0..100
	MemCache   model.Gauge `json:"memCache"`   // bytes of file cache
	MemSwap    model.Gauge `json:"memSwap"`    // bytes, cgroup v1 only when available

	NetRxBytes model.Gauge `json:"netRxBytes"` // cumulative since container start
	NetTxBytes model.Gauge `json:"netTxBytes"`
	NetRxBps   model.Gauge `json:"netRxBps"`
	NetTxBps   model.Gauge `json:"netTxBps"`
	NetRxPps   model.Gauge `json:"netRxPps"`
	NetTxPps   model.Gauge `json:"netTxPps"`

	BlkReadBytes  model.Gauge `json:"blkReadBytes"`
	BlkWriteBytes model.Gauge `json:"blkWriteBytes"`
	BlkReadBps    model.Gauge `json:"blkReadBps"`
	BlkWriteBps   model.Gauge `json:"blkWriteBps"`

	Pids model.Gauge `json:"pids"`

	Interfaces []IfaceStats `json:"interfaces"`
}

type IfaceStats struct {
	Name      string  `json:"name"`
	RxBytes   uint64  `json:"rxBytes"`
	TxBytes   uint64  `json:"txBytes"`
	RxPackets uint64  `json:"rxPackets"`
	TxPackets uint64  `json:"txPackets"`
	RxErrors  uint64  `json:"rxErrors"`
	TxErrors  uint64  `json:"txErrors"`
	RxDropped uint64  `json:"rxDropped"`
	TxDropped uint64  `json:"txDropped"`
	RxBps     float64 `json:"rxBps"`
	TxBps     float64 `json:"txBps"`
	RxPps     float64 `json:"rxPps"`
	TxPps     float64 `json:"txPps"`
	Rated     bool    `json:"rated"` // false when no previous sample existed
}

// CPUPercent implements the docker CLI calculation (calculateCPUPercentUnix):
// the container's CPU time delta divided by the host's system CPU time delta,
// multiplied by the number of online CPUs.
func CPUPercent(cur, pre container.CPUStats) (float64, bool) {
	if cur.CPUUsage.TotalUsage < pre.CPUUsage.TotalUsage || cur.SystemUsage <= pre.SystemUsage || pre.SystemUsage == 0 {
		return 0, false
	}
	cpuDelta := float64(cur.CPUUsage.TotalUsage - pre.CPUUsage.TotalUsage)
	sysDelta := float64(cur.SystemUsage - pre.SystemUsage)
	online := float64(OnlineCPUs(cur))
	if online == 0 {
		return 0, false
	}
	return cpuDelta / sysDelta * online * 100.0, true
}

func OnlineCPUs(s container.CPUStats) uint32 {
	if s.OnlineCPUs > 0 {
		return s.OnlineCPUs
	}
	return uint32(len(s.CPUUsage.PercpuUsage))
}

// MemoryUsage implements the docker CLI calculation (calculateMemUsageUnixNoCache):
// the page cache that can be reclaimed (inactive_file) is subtracted.
// cgroup v1 reports "total_inactive_file", cgroup v2 "inactive_file".
func MemoryUsage(m container.MemoryStats) (usage, cache uint64, ok bool) {
	if m.Usage == 0 && m.Limit == 0 && len(m.Stats) == 0 {
		return 0, 0, false
	}
	usage = m.Usage
	if v, has := m.Stats["total_inactive_file"]; has && v < usage { // cgroup v1
		usage -= v
	} else if v, has := m.Stats["inactive_file"]; has && v < usage { // cgroup v2
		usage -= v
	}
	if v, has := m.Stats["total_cache"]; has {
		cache = v
	} else if v, has := m.Stats["file"]; has {
		cache = v
	} else if v, has := m.Stats["cache"]; has {
		cache = v
	}
	return usage, cache, true
}

// BlkioBytes sums read/write bytes. cgroup v1 uses "Read"/"Write", cgroup v2 "read"/"write".
func BlkioBytes(b container.BlkioStats) (read, write uint64, ok bool) {
	if b.IoServiceBytesRecursive == nil {
		return 0, 0, false
	}
	for _, e := range b.IoServiceBytesRecursive {
		switch strings.ToLower(e.Op) {
		case "read":
			read += e.Value
		case "write":
			write += e.Value
		}
	}
	return read, write, true
}

// Calculator turns consecutive raw docker samples of one container into rates.
type Calculator struct {
	prev     *container.StatsResponse
	prevTime time.Time
}

// Compute derives a snapshot from a raw sample. hostMem is used as limit fallback;
// cpuLimit is the configured CPU limit in cores (0 = unlimited).
func (c *Calculator) Compute(id string, s *container.StatsResponse, cpuLimit float64) ContainerStats {
	out := ContainerStats{ID: id, TS: s.Read.UnixMilli(), OnlineCPUs: OnlineCPUs(s.CPUStats)}
	pending := model.NA(model.Pending, "waiting for second sample")

	// CPU: prefer the precpu sample docker embeds; fall back to our previous sample.
	pre := s.PreCPUStats
	if pre.SystemUsage == 0 && c.prev != nil {
		pre = c.prev.CPUStats
	}
	if pct, ok := CPUPercent(s.CPUStats, pre); ok {
		out.CPUPercent = model.Val(round2(pct))
		capacity := float64(out.OnlineCPUs)
		if cpuLimit > 0 && cpuLimit < capacity {
			capacity = cpuLimit
		}
		if capacity > 0 {
			out.CPUPercentOfLimit = model.Val(round2(clamp(pct/capacity, 0, 100)))
		}
		if dp := s.CPUStats.ThrottlingData.Periods - pre.ThrottlingData.Periods; dp > 0 && s.CPUStats.ThrottlingData.Periods >= pre.ThrottlingData.Periods {
			dt := s.CPUStats.ThrottlingData.ThrottledPeriods - pre.ThrottlingData.ThrottledPeriods
			out.CPUThrottledPct = model.Val(round2(float64(dt) / float64(dp) * 100))
		} else {
			out.CPUThrottledPct = model.NA(model.Unavailable, "no CFS quota configured")
		}
	} else if s.CPUStats.CPUUsage.TotalUsage == 0 && s.CPUStats.SystemUsage == 0 {
		out.CPUPercent = model.NA(model.Unavailable, "docker reported no cpu statistics")
		out.CPUPercentOfLimit = out.CPUPercent
		out.CPUThrottledPct = out.CPUPercent
	} else {
		out.CPUPercent, out.CPUPercentOfLimit, out.CPUThrottledPct = pending, pending, pending
	}

	// Memory
	if used, cache, ok := MemoryUsage(s.MemoryStats); ok {
		out.MemUsage = model.Val(float64(used))
		out.MemCache = model.Val(float64(cache))
		if s.MemoryStats.Limit > 0 {
			out.MemLimit = model.Val(float64(s.MemoryStats.Limit))
			out.MemPercent = model.Val(round2(float64(used) / float64(s.MemoryStats.Limit) * 100))
		} else {
			out.MemLimit = model.NA(model.Unavailable, "no memory limit reported")
			out.MemPercent = out.MemLimit
		}
		if sw, has := s.MemoryStats.Stats["swap"]; has {
			out.MemSwap = model.Val(float64(sw))
		} else {
			out.MemSwap = model.NA(model.Unsupported, "swap accounting not reported by cgroup")
		}
	} else {
		na := model.NA(model.Unavailable, "docker reported no memory statistics")
		out.MemUsage, out.MemLimit, out.MemPercent, out.MemCache, out.MemSwap = na, na, na, na, na
	}

	// Pids
	if s.PidsStats.Current > 0 {
		out.Pids = model.Val(float64(s.PidsStats.Current))
	} else {
		out.Pids = model.NA(model.Unavailable, "")
	}

	dt := 0.0
	if c.prev != nil {
		dt = s.Read.Sub(c.prev.Read).Seconds()
	}
	rated := dt > 0.05

	// Network
	if s.Networks == nil {
		na := model.NA(model.Unavailable, "container has no own network namespace (e.g. host or none network mode)")
		out.NetRxBytes, out.NetTxBytes, out.NetRxBps, out.NetTxBps, out.NetRxPps, out.NetTxPps = na, na, na, na, na, na
	} else {
		var rx, tx, rxp, txp, prx, ptx, prxp, ptxp uint64
		for name, n := range s.Networks {
			ifs := IfaceStats{Name: name, RxBytes: n.RxBytes, TxBytes: n.TxBytes, RxPackets: n.RxPackets, TxPackets: n.TxPackets,
				RxErrors: n.RxErrors, TxErrors: n.TxErrors, RxDropped: n.RxDropped, TxDropped: n.TxDropped}
			rx += n.RxBytes
			tx += n.TxBytes
			rxp += n.RxPackets
			txp += n.TxPackets
			if rated {
				if p, ok := c.prev.Networks[name]; ok {
					ifs.RxBps = rate(n.RxBytes, p.RxBytes, dt)
					ifs.TxBps = rate(n.TxBytes, p.TxBytes, dt)
					ifs.RxPps = rate(n.RxPackets, p.RxPackets, dt)
					ifs.TxPps = rate(n.TxPackets, p.TxPackets, dt)
					ifs.Rated = true
				}
			}
			out.Interfaces = append(out.Interfaces, ifs)
		}
		out.NetRxBytes, out.NetTxBytes = model.Val(float64(rx)), model.Val(float64(tx))
		if rated && c.prev.Networks != nil {
			for _, p := range c.prev.Networks {
				prx += p.RxBytes
				ptx += p.TxBytes
				prxp += p.RxPackets
				ptxp += p.TxPackets
			}
			out.NetRxBps, out.NetTxBps = model.Val(rate(rx, prx, dt)), model.Val(rate(tx, ptx, dt))
			out.NetRxPps, out.NetTxPps = model.Val(rate(rxp, prxp, dt)), model.Val(rate(txp, ptxp, dt))
		} else {
			out.NetRxBps, out.NetTxBps, out.NetRxPps, out.NetTxPps = pending, pending, pending, pending
		}
		sortIfaces(out.Interfaces)
	}

	// Block I/O
	if r, w, ok := BlkioBytes(s.BlkioStats); ok {
		out.BlkReadBytes, out.BlkWriteBytes = model.Val(float64(r)), model.Val(float64(w))
		if rated {
			if pr, pw, ok := BlkioBytes(c.prev.BlkioStats); ok {
				out.BlkReadBps, out.BlkWriteBps = model.Val(rate(r, pr, dt)), model.Val(rate(w, pw, dt))
			} else {
				out.BlkReadBps, out.BlkWriteBps = pending, pending
			}
		} else {
			out.BlkReadBps, out.BlkWriteBps = pending, pending
		}
	} else {
		na := model.NA(model.Unavailable, "block I/O accounting not reported (cgroup/io controller or storage driver)")
		out.BlkReadBytes, out.BlkWriteBytes, out.BlkReadBps, out.BlkWriteBps = na, na, na, na
	}

	cp := *s
	c.prev = &cp
	c.prevTime = s.Read
	return out
}

// rate returns a per-second rate; counter resets yield 0 for that interval.
func rate(cur, prev uint64, dt float64) float64 {
	if cur < prev || dt <= 0 {
		return 0
	}
	return round2(float64(cur-prev) / dt)
}

func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func sortIfaces(s []IfaceStats) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j].Name < s[j-1].Name; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

