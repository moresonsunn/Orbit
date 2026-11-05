package metrics

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/phili/orbit/internal/settings"
	"github.com/phili/orbit/internal/store"
)

func TestHistoryStore_FlushBatch_NoDeadlock(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orbit-history-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	st, err := store.Open(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sm, err := settings.NewManager(ctx, st)
	if err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	hs := NewHistoryStore(st, sm, logger)

	// Queue multiple batches with new series keys
	batch := []sampleQueueItem{
		{key: "host:cpu", ts: time.Now().Unix(), value: 42.5},
		{key: "host:mem", ts: time.Now().Unix(), value: 75.0},
		{key: "host:cpu", ts: time.Now().Unix() + 1, value: 45.0},
		{key: "container:123:cpu", ts: time.Now().Unix(), value: 12.3},
	}

	// This must not hang or deadlock
	hs.flushBatch(ctx, batch)

	// Now query back
	res, err := hs.Query(ctx, "host:cpu", time.Now().Unix()-100, time.Now().Unix()+100, "raw")
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if len(res.Points) == 0 {
		t.Fatalf("expected points for host:cpu, got 0")
	}
}
