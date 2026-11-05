package alerts

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/phili/orbit/internal/docker"
	"github.com/phili/orbit/internal/host"
	"github.com/phili/orbit/internal/realtime"
	"github.com/phili/orbit/internal/settings"
	"github.com/phili/orbit/internal/store"
)

type Alert struct {
	ID           int64      `json:"id"`
	Rule         string     `json:"rule"`
	Target       string     `json:"target"`
	TargetName   string     `json:"targetName"`
	Severity     string     `json:"severity"` // warning | critical
	Message      string     `json:"message"`
	Value        float64    `json:"value"`
	Threshold    float64    `json:"threshold"`
	StartedAt    time.Time  `json:"startedAt"`
	ResolvedAt   *time.Time `json:"resolvedAt,omitempty"`
	Acknowledged bool       `json:"acknowledged"`
}

type trackState struct {
	firstBreach time.Time
	alertID     int64
	active      bool
}

type Engine struct {
	settingsMgr *settings.Manager
	store       *store.Store
	hub         *realtime.Hub
	log         *slog.Logger
	httpClient  *http.Client

	mu      sync.Mutex
	tracker map[string]*trackState // key: "rule:target"
}

func NewEngine(sm *settings.Manager, st *store.Store, hub *realtime.Hub, log *slog.Logger) *Engine {
	return &Engine{
		settingsMgr: sm,
		store:       st,
		hub:         hub,
		log:         log,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
		tracker:     make(map[string]*trackState),
	}
}

// CheckHost evaluates host metrics against alert thresholds.
func (e *Engine) CheckHost(ctx context.Context, m host.HostMetrics) {
	rules := e.settingsMgr.Get().Alerts

	// Host CPU
	if rules.HostCPU.Enabled && m.CPU.TotalPercent.Value != nil {
		e.evaluateThreshold(ctx, "hostCpu", "host", "Host-System", "warning",
			"Hohe CPU-Auslastung auf dem Host",
			*m.CPU.TotalPercent.Value, rules.HostCPU.Value, rules.HostCPU.DurationSec)
	}

	// Host Memory
	if rules.HostMemory.Enabled && m.Memory.UsedPercent.Value != nil {
		e.evaluateThreshold(ctx, "hostMemory", "host", "Host-System", "warning",
			"Hoher RAM-Verbrauch auf dem Host",
			*m.Memory.UsedPercent.Value, rules.HostMemory.Value, rules.HostMemory.DurationSec)
	}

	// Disk Mounts
	if rules.DiskUsage.Enabled {
		for _, mount := range m.Storage.Mounts {
			if mount.UsedPercent.Value != nil {
				e.evaluateThreshold(ctx, "diskUsage", mount.MountPoint, fmt.Sprintf("Mount %s", mount.MountPoint), "critical",
					fmt.Sprintf("Festplattenspeicherplatz auf %s beinahe erschöpft", mount.MountPoint),
					*mount.UsedPercent.Value, rules.DiskUsage.Value, rules.DiskUsage.DurationSec)
			}
		}
	}
}

// CheckContainer evaluates container stats against alert thresholds.
func (e *Engine) CheckContainer(ctx context.Context, c docker.Container, st docker.ContainerStats) {
	rules := e.settingsMgr.Get().Alerts

	// Container CPU
	if rules.ContainerCPU.Enabled && st.CPUPercentOfLimit.Value != nil {
		e.evaluateThreshold(ctx, "containerCpu", c.ID, c.Name, "warning",
			fmt.Sprintf("Hohe CPU-Auslastung bei Container %s", c.Name),
			*st.CPUPercentOfLimit.Value, rules.ContainerCPU.Value, rules.ContainerCPU.DurationSec)
	}

	// Container Memory
	if rules.ContainerMemory.Enabled && st.MemPercent.Value != nil {
		e.evaluateThreshold(ctx, "containerMemory", c.ID, c.Name, "warning",
			fmt.Sprintf("Hoher RAM-Verbrauch bei Container %s", c.Name),
			*st.MemPercent.Value, rules.ContainerMemory.Value, rules.ContainerMemory.DurationSec)
	}
}

// HandleDockerEvent inspects container crashes or health degradation events.
func (e *Engine) HandleDockerEvent(ctx context.Context, ev docker.DockerEvent) {
	rules := e.settingsMgr.Get().Alerts

	if ev.Type != "container" {
		return
	}

	if rules.ContainerCrash.Enabled && (ev.Action == "die" || ev.Action == "oom") {
		exitCode := ev.Attributes["exitCode"]
		if exitCode != "0" || ev.Action == "oom" {
			msg := fmt.Sprintf("Container %s ist unerwartet beendet worden (Exit %s)", ev.ActorName, exitCode)
			if ev.Action == "oom" {
				msg = fmt.Sprintf("Container %s wurde durch OOM (Out Of Memory) beendet", ev.ActorName)
			}
			e.fireInstantAlert(ctx, "containerCrash", ev.ActorID, ev.ActorName, "critical", msg)
		}
	}

	if rules.ContainerHealthy.Enabled && ev.Action == "health_status: unhealthy" {
		msg := fmt.Sprintf("Container %s ist ungesund (Healthcheck fehlgeschlagen)", ev.ActorName)
		e.fireInstantAlert(ctx, "containerUnhealthy", ev.ActorID, ev.ActorName, "critical", msg)
	}
}

func (e *Engine) evaluateThreshold(ctx context.Context, rule, target, targetName, severity, message string, currentVal, threshold float64, durationSec int) {
	key := fmt.Sprintf("%s:%s", rule, target)
	now := time.Now()

	e.mu.Lock()
	state, exists := e.tracker[key]
	if !exists {
		state = &trackState{}
		e.tracker[key] = state
	}

	breached := currentVal >= threshold

	if breached {
		if state.firstBreach.IsZero() {
			state.firstBreach = now
		}

		durationMet := durationSec == 0 || now.Sub(state.firstBreach) >= time.Duration(durationSec)*time.Second
		if durationMet && !state.active {
			state.active = true
			e.mu.Unlock()

			// Fire Alert
			alert := Alert{
				Rule:       rule,
				Target:     target,
				TargetName: targetName,
				Severity:   severity,
				Message:    fmt.Sprintf("%s (aktuell: %.1f, Schwellwert: %.1f)", message, currentVal, threshold),
				Value:      currentVal,
				Threshold:  threshold,
				StartedAt:  now,
			}
			e.persistAndBroadcast(ctx, alert, state)
			return
		}
	} else {
		// Recovered
		state.firstBreach = time.Time{}
		if state.active {
			state.active = false
			alertID := state.alertID
			e.mu.Unlock()

			// Resolve Alert
			e.resolveAlert(ctx, alertID, now)
			return
		}
	}
	e.mu.Unlock()
}

func (e *Engine) fireInstantAlert(ctx context.Context, rule, target, targetName, severity, message string) {
	now := time.Now()
	alert := Alert{
		Rule:       rule,
		Target:     target,
		TargetName: targetName,
		Severity:   severity,
		Message:    message,
		StartedAt:  now,
	}
	state := &trackState{active: true}
	e.persistAndBroadcast(ctx, alert, state)
}

func (e *Engine) persistAndBroadcast(ctx context.Context, a Alert, state *trackState) {
	if e.store != nil {
		res, err := e.store.DB.ExecContext(ctx,
			`INSERT INTO alerts (rule, target, target_name, severity, message, value, threshold, started_at, acknowledged)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0)`,
			a.Rule, a.Target, a.TargetName, a.Severity, a.Message, a.Value, a.Threshold, a.StartedAt.Unix(),
		)
		if err == nil {
			if id, err := res.LastInsertId(); err == nil {
				a.ID = id
				e.mu.Lock()
				state.alertID = id
				e.mu.Unlock()
			}
		}
	}

	e.hub.Broadcast("alerts", a)
	e.sendWebhook(a)
}

func (e *Engine) resolveAlert(ctx context.Context, alertID int64, resolvedAt time.Time) {
	if e.store != nil && alertID > 0 {
		_, _ = e.store.DB.ExecContext(ctx, `UPDATE alerts SET resolved_at = ? WHERE id = ?`, resolvedAt.Unix(), alertID)
	}
	e.hub.Broadcast("alerts", map[string]interface{}{
		"id":         alertID,
		"resolved":   true,
		"resolvedAt": resolvedAt,
	})
}

func (e *Engine) Acknowledge(ctx context.Context, alertID int64) error {
	if e.store == nil {
		return nil
	}
	_, err := e.store.DB.ExecContext(ctx, `UPDATE alerts SET acknowledged = 1 WHERE id = ?`, alertID)
	return err
}

func (e *Engine) RecentAlerts(ctx context.Context, limit int) ([]Alert, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := e.store.DB.QueryContext(ctx,
		`SELECT id, rule, target, target_name, severity, message, value, threshold, started_at, resolved_at, acknowledged
		 FROM alerts ORDER BY started_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []Alert
	for rows.Next() {
		var a Alert
		var startedAt int64
		var resolvedAt sql.NullInt64
		var val, thresh sql.NullFloat64

		if err := rows.Scan(&a.ID, &a.Rule, &a.Target, &a.TargetName, &a.Severity, &a.Message, &val, &thresh, &startedAt, &resolvedAt, &a.Acknowledged); err != nil {
			return nil, err
		}
		a.StartedAt = time.Unix(startedAt, 0)
		if resolvedAt.Valid {
			t := time.Unix(resolvedAt.Int64, 0)
			a.ResolvedAt = &t
		}
		if val.Valid {
			a.Value = val.Float64
		}
		if thresh.Valid {
			a.Threshold = thresh.Float64
		}
		alerts = append(alerts, a)
	}
	return alerts, rows.Err()
}

func (e *Engine) sendWebhook(a Alert) {
	wh := e.settingsMgr.Get().Webhook
	if !wh.Enabled || wh.URL == "" {
		return
	}
	// Check min severity
	if wh.MinSeverity == "critical" && a.Severity != "critical" {
		return
	}

	payload, err := json.Marshal(map[string]interface{}{
		"source":    "orbit",
		"event":     "alert",
		"timestamp": a.StartedAt,
		"alert":     a,
	})
	if err != nil {
		return
	}

	go func() {
		req, err := http.NewRequest("POST", wh.URL, bytes.NewReader(payload))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "orbit-alert-webhook")
		resp, err := e.httpClient.Do(req)
		if err != nil {
			e.log.Warn("webhook dispatch failed", "url", wh.URL, "err", err)
			return
		}
		resp.Body.Close()
	}()
}
