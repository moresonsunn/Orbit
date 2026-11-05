// Package settings holds runtime-adjustable configuration persisted in SQLite.
package settings

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sync"

	"github.com/phili/orbit/internal/store"
)

// Settings are editable by administrators in the web UI.
type Settings struct {
	Intervals Intervals  `json:"intervals"`
	Retention Retention  `json:"retention"`
	Alerts    AlertRules `json:"alerts"`
	Webhook   Webhook    `json:"webhook"`
}

// Intervals in milliseconds.
type Intervals struct {
	HostMs      int `json:"hostMs"`
	ContainerMs int `json:"containerMs"`
	GPUMs       int `json:"gpuMs"`
	NetflowMs   int `json:"netflowMs"`
	// HistoryStepSec is the resolution of the persisted raw tier.
	HistoryStepSec int `json:"historyStepSec"`
}

type Retention struct {
	RawHours int `json:"rawHours"` // raw tier (HistoryStepSec resolution)
	Days5m   int `json:"days5m"`   // 5 minute tier
	Days1h   int `json:"days1h"`   // 1 hour tier
	// EventDays controls retention of docker events, alerts and audit log.
	EventDays int `json:"eventDays"`
}

type Threshold struct {
	Enabled     bool    `json:"enabled"`
	Value       float64 `json:"value"`
	DurationSec int     `json:"durationSec"`
}

type AlertRules struct {
	HostCPU          Threshold `json:"hostCpu"`          // percent
	HostMemory       Threshold `json:"hostMemory"`       // percent
	DiskUsage        Threshold `json:"diskUsage"`        // percent per filesystem
	ContainerCPU     Threshold `json:"containerCpu"`     // percent of one core * cores (docker semantics, normalized to limit)
	ContainerMemory  Threshold `json:"containerMemory"`  // percent of limit
	NetworkRx        Threshold `json:"networkRx"`        // bytes/s host total
	NetworkTx        Threshold `json:"networkTx"`        // bytes/s host total
	GPUTemp          Threshold `json:"gpuTemp"`          // celsius
	ContainerCrash   Threshold `json:"containerCrash"`   // enabled flag only
	RestartLoop      Threshold `json:"restartLoop"`      // value = restarts within DurationSec
	ContainerHealthy Threshold `json:"containerUnhealthy"`
}

type Webhook struct {
	Enabled     bool   `json:"enabled"`
	URL         string `json:"url"`
	MinSeverity string `json:"minSeverity"` // info | warning | critical
}

func Defaults() Settings {
	return Settings{
		Intervals: Intervals{HostMs: 1000, ContainerMs: 1000, GPUMs: 2000, NetflowMs: 2000, HistoryStepSec: 15},
		Retention: Retention{RawHours: 24, Days5m: 7, Days1h: 30, EventDays: 30},
		Alerts: AlertRules{
			HostCPU:          Threshold{Enabled: true, Value: 90, DurationSec: 120},
			HostMemory:       Threshold{Enabled: true, Value: 90, DurationSec: 120},
			DiskUsage:        Threshold{Enabled: true, Value: 90, DurationSec: 0},
			ContainerCPU:     Threshold{Enabled: true, Value: 90, DurationSec: 120},
			ContainerMemory:  Threshold{Enabled: true, Value: 90, DurationSec: 60},
			NetworkRx:        Threshold{Enabled: false, Value: 100 * 1024 * 1024, DurationSec: 60},
			NetworkTx:        Threshold{Enabled: false, Value: 100 * 1024 * 1024, DurationSec: 60},
			GPUTemp:          Threshold{Enabled: true, Value: 85, DurationSec: 30},
			ContainerCrash:   Threshold{Enabled: true},
			RestartLoop:      Threshold{Enabled: true, Value: 3, DurationSec: 600},
			ContainerHealthy: Threshold{Enabled: true},
		},
		Webhook: Webhook{MinSeverity: "warning"},
	}
}

func (s Settings) Validate() error {
	in := s.Intervals
	if in.HostMs < 500 || in.HostMs > 60000 || in.ContainerMs < 1000 || in.ContainerMs > 60000 ||
		in.GPUMs < 500 || in.GPUMs > 60000 || in.NetflowMs < 1000 || in.NetflowMs > 60000 {
		return errors.New("intervals out of range (host/gpu 500-60000 ms, container/netflow 1000-60000 ms)")
	}
	if in.HistoryStepSec < 5 || in.HistoryStepSec > 300 {
		return errors.New("historyStepSec must be between 5 and 300")
	}
	r := s.Retention
	if r.RawHours < 1 || r.RawHours > 168 || r.Days5m < 1 || r.Days5m > 90 || r.Days1h < 1 || r.Days1h > 730 || r.EventDays < 1 || r.EventDays > 365 {
		return errors.New("retention out of range")
	}
	if s.Webhook.Enabled {
		u, err := url.Parse(s.Webhook.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return errors.New("webhook url must be an absolute http(s) url")
		}
	}
	switch s.Webhook.MinSeverity {
	case "info", "warning", "critical":
	default:
		return errors.New("webhook minSeverity must be info, warning or critical")
	}
	for name, t := range map[string]Threshold{
		"hostCpu": s.Alerts.HostCPU, "hostMemory": s.Alerts.HostMemory, "diskUsage": s.Alerts.DiskUsage,
		"containerCpu": s.Alerts.ContainerCPU, "containerMemory": s.Alerts.ContainerMemory,
		"networkRx": s.Alerts.NetworkRx, "networkTx": s.Alerts.NetworkTx, "gpuTemp": s.Alerts.GPUTemp,
		"restartLoop": s.Alerts.RestartLoop,
	} {
		if t.Value < 0 || t.DurationSec < 0 || t.DurationSec > 86400 {
			return fmt.Errorf("alert %s: invalid threshold", name)
		}
	}
	return nil
}

// Manager caches settings and notifies subscribers on change.
type Manager struct {
	st   *store.Store
	mu   sync.RWMutex
	cur  Settings
	subs []func(Settings)
}

func NewManager(ctx context.Context, st *store.Store) (*Manager, error) {
	m := &Manager{st: st, cur: Defaults()}
	var raw string
	err := st.DB.QueryRowContext(ctx, `SELECT value FROM settings WHERE key='app'`).Scan(&raw)
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return nil, err
	default:
		s := Defaults() // unknown/new fields keep defaults
		if err := json.Unmarshal([]byte(raw), &s); err == nil && s.Validate() == nil {
			m.cur = s
		}
	}
	return m, nil
}

func (m *Manager) Get() Settings {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cur
}

func (m *Manager) OnChange(f func(Settings)) {
	m.mu.Lock()
	m.subs = append(m.subs, f)
	m.mu.Unlock()
}

func (m *Manager) Update(ctx context.Context, s Settings) error {
	if err := s.Validate(); err != nil {
		return err
	}
	b, _ := json.Marshal(s)
	if _, err := m.st.DB.ExecContext(ctx, `INSERT INTO settings(key,value) VALUES('app',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, string(b)); err != nil {
		return err
	}
	m.mu.Lock()
	m.cur = s
	subs := append([]func(Settings){}, m.subs...)
	m.mu.Unlock()
	for _, f := range subs {
		f(s)
	}
	return nil
}

