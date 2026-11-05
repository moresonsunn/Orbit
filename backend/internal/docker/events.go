package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	eventsTypes "github.com/moby/moby/api/types/events"
	"github.com/phili/orbit/internal/store"
)

type EventSeverity string

const (
	SeverityInfo     EventSeverity = "info"
	SeverityWarning  EventSeverity = "warning"
	SeverityCritical EventSeverity = "critical"
)

type DockerEvent struct {
	ID         int64             `json:"id"`
	Timestamp  time.Time         `json:"timestamp"`
	Type       string            `json:"type"`
	Action     string            `json:"action"`
	ActorID    string            `json:"actorId"`
	ActorName  string            `json:"actorName"`
	Severity   EventSeverity     `json:"severity"`
	Attributes map[string]string `json:"attributes"`
}

type EventListener func(DockerEvent)

type EventWatcher struct {
	gw        *Gateway
	inv       *Inventory
	store     *store.Store
	log       *slog.Logger
	mu        sync.RWMutex
	listeners []EventListener
}

func NewEventWatcher(gw *Gateway, inv *Inventory, st *store.Store, log *slog.Logger) *EventWatcher {
	return &EventWatcher{
		gw:    gw,
		inv:   inv,
		store: st,
		log:   log,
	}
}

func (ew *EventWatcher) OnEvent(listener EventListener) {
	ew.mu.Lock()
	defer ew.mu.Unlock()
	ew.listeners = append(ew.listeners, listener)
}

func (ew *EventWatcher) emit(ev DockerEvent) {
	ew.mu.RLock()
	ls := append([]EventListener{}, ew.listeners...)
	ew.mu.RUnlock()
	for _, l := range ls {
		l(ev)
	}
}

func (ew *EventWatcher) Run(ctx context.Context) {
	var lastSince string
	backoff := time.Second

	for {
		if !ew.gw.Connected() {
			select {
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Second):
				continue
			}
		}

		if lastSince == "" {
			lastSince = fmt.Sprintf("%d", time.Now().Add(-1*time.Minute).Unix())
		}

		res := ew.gw.Events(ctx, lastSince)
		ew.log.Info("docker event stream started", "since", lastSince)
		backoff = time.Second

	streamLoop:
		for {
			select {
			case <-ctx.Done():
				return
			case err, ok := <-res.Err:
				if !ok || err != nil {
					if err != nil && ctx.Err() == nil {
						ew.log.Warn("docker events stream error", "err", err)
					}
					break streamLoop
				}
			case msg, ok := <-res.Messages:
				if !ok {
					break streamLoop
				}
				lastSince = fmt.Sprintf("%d", msg.Time)
				ew.handleMessage(ctx, msg)
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
			if backoff < 15*time.Second {
				backoff *= 2
			}
		}
	}
}

func (ew *EventWatcher) handleMessage(ctx context.Context, msg eventsTypes.Message) {
	actorName := msg.Actor.Attributes["name"]
	if actorName == "" {
		actorName = msg.Actor.ID
		if len(actorName) > 12 {
			actorName = actorName[:12]
		}
	}

	severity := SeverityInfo
	switch msg.Action {
	case "die", "kill", "oom":
		severity = SeverityWarning
		if msg.Action == "oom" || msg.Actor.Attributes["exitCode"] != "0" {
			severity = SeverityCritical
		}
	case "health_status: unhealthy":
		severity = SeverityCritical
	case "health_status: healthy":
		severity = SeverityInfo
	case "destroy", "prune":
		severity = SeverityWarning
	}

	event := DockerEvent{
		Timestamp:  time.Unix(0, msg.TimeNano),
		Type:       string(msg.Type),
		Action:     string(msg.Action),
		ActorID:    msg.Actor.ID,
		ActorName:  actorName,
		Severity:   severity,
		Attributes: msg.Actor.Attributes,
	}

	// Persist event in sqlite
	if ew.store != nil {
		attrJSON, _ := json.Marshal(msg.Actor.Attributes)
		res, err := ew.store.DB.ExecContext(ctx,
			`INSERT INTO docker_events (ts_nano, type, action, actor_id, actor_name, severity, attributes)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			msg.TimeNano, string(msg.Type), string(msg.Action), msg.Actor.ID, actorName, string(severity), string(attrJSON),
		)
		if err == nil {
			if id, err := res.LastInsertId(); err == nil {
				event.ID = id
			}
		} else {
			ew.log.Error("failed to insert docker event into sqlite", "err", err)
		}
	}

	// Trigger inventory updates based on event type
	switch msg.Type {
	case eventsTypes.ContainerEventType:
		switch msg.Action {
		case "start", "stop", "die", "pause", "unpause", "restart", "rename", "update":
			ew.inv.Refresh(ctx, msg.Actor.ID)
		case "destroy":
			ew.inv.Refresh(ctx, msg.Actor.ID)
		}
		if len(msg.Action) >= 14 && msg.Action[:14] == "health_status:" {
			ew.inv.Refresh(ctx, msg.Actor.ID)
		}
	case eventsTypes.NetworkEventType:
		ew.inv.RefreshResources(ctx, "networks")
		if msg.Action == "connect" || msg.Action == "disconnect" {
			if containerID := msg.Actor.Attributes["container"]; containerID != "" {
				ew.inv.Refresh(ctx, containerID)
			}
		}
	case eventsTypes.ImageEventType:
		ew.inv.RefreshResources(ctx, "images")
	case eventsTypes.VolumeEventType:
		ew.inv.RefreshResources(ctx, "volumes")
	}

	ew.emit(event)
}

func (ew *EventWatcher) RecentEvents(ctx context.Context, limit int) ([]DockerEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	rows, err := ew.store.DB.QueryContext(ctx,
		`SELECT id, ts_nano, type, action, actor_id, actor_name, severity, attributes
		 FROM docker_events
		 ORDER BY ts_nano DESC
		 LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []DockerEvent
	for rows.Next() {
		var ev DockerEvent
		var tsNano int64
		var attrStr string
		if err := rows.Scan(&ev.ID, &tsNano, &ev.Type, &ev.Action, &ev.ActorID, &ev.ActorName, &ev.Severity, &attrStr); err != nil {
			return nil, err
		}
		ev.Timestamp = time.Unix(0, tsNano)
		if attrStr != "" {
			_ = json.Unmarshal([]byte(attrStr), &ev.Attributes)
		}
		events = append(events, ev)
	}
	return events, rows.Err()
}
