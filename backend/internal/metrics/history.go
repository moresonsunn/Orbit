package metrics

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/phili/orbit/internal/docker"
	"github.com/phili/orbit/internal/host"
	"github.com/phili/orbit/internal/settings"
	"github.com/phili/orbit/internal/store"
)

type MetricPoint struct {
	Timestamp int64   `json:"ts"` // unix seconds
	Avg       float64 `json:"avg"`
	Min       float64 `json:"min"`
	Max       float64 `json:"max"`
}

type SeriesHistory struct {
	Key    string        `json:"key"`
	Points []MetricPoint `json:"points"`
}

type sampleQueueItem struct {
	key   string
	ts    int64
	value float64
}

type HistoryStore struct {
	store       *store.Store
	settingsMgr *settings.Manager
	log         *slog.Logger

	mu         sync.Mutex
	seriesMap  map[string]int64
	sampleChan chan sampleQueueItem
}

func NewHistoryStore(st *store.Store, sm *settings.Manager, log *slog.Logger) *HistoryStore {
	hs := &HistoryStore{
		store:       st,
		settingsMgr: sm,
		log:         log,
		seriesMap:   make(map[string]int64),
		sampleChan:  make(chan sampleQueueItem, 2048),
	}
	hs.loadSeriesCache()
	return hs
}

func (hs *HistoryStore) loadSeriesCache() {
	if hs.store == nil {
		return
	}
	rows, err := hs.store.DB.Query(`SELECT id, key FROM series`)
	if err != nil {
		return
	}
	defer rows.Close()

	hs.mu.Lock()
	defer hs.mu.Unlock()
	for rows.Next() {
		var id int64
		var key string
		if err := rows.Scan(&id, &key); err == nil {
			hs.seriesMap[key] = id
		}
	}
}

func (hs *HistoryStore) getOrCreateSeriesID(ctx context.Context, key string) (int64, error) {
	hs.mu.Lock()
	if id, ok := hs.seriesMap[key]; ok {
		hs.mu.Unlock()
		return id, nil
	}
	hs.mu.Unlock()

	var id int64
	err := hs.store.DB.QueryRowContext(ctx,
		`INSERT INTO series (key) VALUES (?) ON CONFLICT(key) DO UPDATE SET key=excluded.key RETURNING id`, key,
	).Scan(&id)
	if err != nil {
		return 0, err
	}

	hs.mu.Lock()
	hs.seriesMap[key] = id
	hs.mu.Unlock()
	return id, nil
}

func (hs *HistoryStore) Record(key string, value float64) {
	select {
	case hs.sampleChan <- sampleQueueItem{key: key, ts: time.Now().Unix(), value: value}:
	default:
		// Queue full, drop rather than blocking realtime loop
	}
}

func (hs *HistoryStore) RecordHost(m host.HostMetrics) {
	if m.CPU.TotalPercent.Value != nil {
		hs.Record("host:cpu", *m.CPU.TotalPercent.Value)
	}
	if m.Memory.UsedPercent.Value != nil {
		hs.Record("host:mem", *m.Memory.UsedPercent.Value)
	}
	if m.Network.RxBps.Value != nil {
		hs.Record("host:net:rx", *m.Network.RxBps.Value)
	}
	if m.Network.TxBps.Value != nil {
		hs.Record("host:net:tx", *m.Network.TxBps.Value)
	}
	if m.Storage.ReadBps.Value != nil {
		hs.Record("host:disk:read", *m.Storage.ReadBps.Value)
	}
	if m.Storage.WriteBps.Value != nil {
		hs.Record("host:disk:write", *m.Storage.WriteBps.Value)
	}
}

func (hs *HistoryStore) RecordContainer(st docker.ContainerStats) {
	if st.CPUPercentOfLimit.Value != nil {
		hs.Record(fmt.Sprintf("container:%s:cpu", st.ID), *st.CPUPercentOfLimit.Value)
	}
	if st.MemPercent.Value != nil {
		hs.Record(fmt.Sprintf("container:%s:mem", st.ID), *st.MemPercent.Value)
	}
	if st.NetRxBps.Value != nil {
		hs.Record(fmt.Sprintf("container:%s:net:rx", st.ID), *st.NetRxBps.Value)
	}
	if st.NetTxBps.Value != nil {
		hs.Record(fmt.Sprintf("container:%s:net:tx", st.ID), *st.NetTxBps.Value)
	}
}

func (hs *HistoryStore) Run(ctx context.Context) {
	flushTicker := time.NewTicker(5 * time.Second)
	rollupTicker := time.NewTicker(5 * time.Minute)
	retentionTicker := time.NewTicker(1 * time.Hour)
	defer flushTicker.Stop()
	defer rollupTicker.Stop()
	defer retentionTicker.Stop()

	var batch []sampleQueueItem

	for {
		select {
		case <-ctx.Done():
			hs.flushBatch(context.Background(), batch)
			return

		case item := <-hs.sampleChan:
			batch = append(batch, item)
			if len(batch) >= 100 {
				hs.flushBatch(ctx, batch)
				batch = batch[:0]
			}

		case <-flushTicker.C:
			if len(batch) > 0 {
				hs.flushBatch(ctx, batch)
				batch = batch[:0]
			}

		case <-rollupTicker.C:
			hs.performRollups(ctx)

		case <-retentionTicker.C:
			hs.enforceRetention(ctx)
		}
	}
}

func (hs *HistoryStore) flushBatch(ctx context.Context, batch []sampleQueueItem) {
	if len(batch) == 0 || hs.store == nil {
		return
	}

	// 1. Resolve all series IDs in advance BEFORE opening the transaction.
	// This ensures no queries are executed against hs.store.DB while tx holds the connection,
	// preventing connection pool deadlocks when MaxOpenConns is limited.
	type preparedItem struct {
		seriesID int64
		ts       int64
		value    float64
	}
	items := make([]preparedItem, 0, len(batch))
	for _, item := range batch {
		sID, err := hs.getOrCreateSeriesID(ctx, item.key)
		if err != nil {
			continue
		}
		items = append(items, preparedItem{
			seriesID: sID,
			ts:       item.ts,
			value:    item.value,
		})
	}

	if len(items) == 0 {
		return
	}

	tx, err := hs.store.DB.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO points_raw (series_id, ts, avg, min, max)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(series_id, ts) DO UPDATE SET
		   avg = (points_raw.avg + excluded.avg) / 2.0,
		   min = MIN(points_raw.min, excluded.min),
		   max = MAX(points_raw.max, excluded.max)`,
	)
	if err != nil {
		return
	}
	defer stmt.Close()

	for _, item := range items {
		_, _ = stmt.ExecContext(ctx, item.seriesID, item.ts, item.value, item.value, item.value)
	}

	_ = tx.Commit()
}

func (hs *HistoryStore) performRollups(ctx context.Context) {
	if hs.store == nil {
		return
	}
	now := time.Now().Unix()
	fiveMinCutoff := now - 300 // 5 min ago

	// Rollup points_raw -> points_5m (grouped by 300-second windows)
	_, _ = hs.store.DB.ExecContext(ctx,
		`INSERT INTO points_5m (series_id, ts, avg, min, max)
		 SELECT series_id, (ts / 300) * 300, AVG(avg), MIN(min), MAX(max)
		 FROM points_raw
		 WHERE ts <= ?
		 GROUP BY series_id, (ts / 300)
		 ON CONFLICT(series_id, ts) DO UPDATE SET
		   avg = excluded.avg, min = excluded.min, max = excluded.max`,
		fiveMinCutoff,
	)

	oneHourCutoff := now - 3600 // 1 hour ago
	// Rollup points_5m -> points_1h (grouped by 3600-second windows)
	_, _ = hs.store.DB.ExecContext(ctx,
		`INSERT INTO points_1h (series_id, ts, avg, min, max)
		 SELECT series_id, (ts / 3600) * 3600, AVG(avg), MIN(min), MAX(max)
		 FROM points_5m
		 WHERE ts <= ?
		 GROUP BY series_id, (ts / 3600)
		 ON CONFLICT(series_id, ts) DO UPDATE SET
		   avg = excluded.avg, min = excluded.min, max = excluded.max`,
		oneHourCutoff,
	)
}

func (hs *HistoryStore) enforceRetention(ctx context.Context) {
	if hs.store == nil {
		return
	}
	ret := hs.settingsMgr.Get().Retention
	now := time.Now().Unix()

	rawCutoff := now - int64(ret.RawHours*3600)
	cut5m := now - int64(ret.Days5m*86400)
	cut1h := now - int64(ret.Days1h*86400)
	cutEvents := now - int64(ret.EventDays*86400)

	_, _ = hs.store.DB.ExecContext(ctx, `DELETE FROM points_raw WHERE ts < ?`, rawCutoff)
	_, _ = hs.store.DB.ExecContext(ctx, `DELETE FROM points_5m WHERE ts < ?`, cut5m)
	_, _ = hs.store.DB.ExecContext(ctx, `DELETE FROM points_1h WHERE ts < ?`, cut1h)
	_, _ = hs.store.DB.ExecContext(ctx, `DELETE FROM docker_events WHERE ts_nano < ?`, cutEvents*1_000_000_000)
	_, _ = hs.store.DB.ExecContext(ctx, `DELETE FROM alerts WHERE started_at < ? AND resolved_at IS NOT NULL`, cutEvents)
	_, _ = hs.store.DB.ExecContext(ctx, `DELETE FROM audit_log WHERE ts < ?`, cutEvents)
}

func (hs *HistoryStore) Query(ctx context.Context, seriesKey string, from, to int64, resolution string) (*SeriesHistory, error) {
	if hs.store == nil {
		return nil, sql.ErrNoRows
	}

	hs.mu.Lock()
	sID, exists := hs.seriesMap[seriesKey]
	hs.mu.Unlock()

	if !exists {
		return &SeriesHistory{Key: seriesKey, Points: []MetricPoint{}}, nil
	}

	table := "points_raw"
	switch resolution {
	case "5m":
		table = "points_5m"
	case "1h":
		table = "points_1h"
	default:
		// Automatically choose resolution based on time range if resolution is "auto" or empty
		diff := to - from
		if diff > 7*86400 {
			table = "points_1h"
		} else if diff > 24*3600 {
			table = "points_5m"
		}
	}

	query := fmt.Sprintf(`SELECT ts, avg, min, max FROM %s WHERE series_id = ? AND ts >= ? AND ts <= ? ORDER BY ts ASC`, table)
	rows, err := hs.store.DB.QueryContext(ctx, query, sID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []MetricPoint
	for rows.Next() {
		var p MetricPoint
		if err := rows.Scan(&p.Timestamp, &p.Avg, &p.Min, &p.Max); err == nil {
			points = append(points, p)
		}
	}

	return &SeriesHistory{
		Key:    seriesKey,
		Points: points,
	}, nil
}

