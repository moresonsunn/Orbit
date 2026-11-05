// Package store wraps the embedded SQLite database (pure Go driver, no CGO needed).
package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	DB   *sql.DB
	path string
}

// Open opens (and creates) the database in dataDir and applies migrations.
func Open(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0o750); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	path := filepath.Join(dataDir, "orbit.db")
	dsn := "file:" + filepath.ToSlash(path) +
		"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=foreign_keys(1)&_pragma=synchronous(NORMAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetConnMaxIdleTime(5 * time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	s := &Store{DB: db, path: path}
	if err := s.migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.DB.Close() }

func (s *Store) Path() string { return s.path }

// SizeBytes returns the on-disk size of the database including WAL.
func (s *Store) SizeBytes() int64 {
	var total int64
	for _, p := range []string{s.path, s.path + "-wal", s.path + "-shm"} {
		if fi, err := os.Stat(p); err == nil {
			total += fi.Size()
		}
	}
	return total
}

func (s *Store) Ping(ctx context.Context) error { return s.DB.PingContext(ctx) }

var migrations = []string{
	// 1: base schema
	`
CREATE TABLE users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  username TEXT NOT NULL UNIQUE COLLATE NOCASE,
  password_hash TEXT NOT NULL,
  role TEXT NOT NULL CHECK (role IN ('admin','viewer')),
  disabled INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL,
  last_login_at INTEGER
);
CREATE TABLE sessions (
  id_hash TEXT PRIMARY KEY,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  csrf_token TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  last_seen_at INTEGER NOT NULL,
  expires_at INTEGER NOT NULL,
  ip TEXT,
  user_agent TEXT
);
CREATE INDEX sessions_user ON sessions(user_id);
CREATE TABLE settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
CREATE TABLE user_prefs (
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  key TEXT NOT NULL,
  value TEXT NOT NULL,
  PRIMARY KEY (user_id, key)
);
CREATE TABLE audit_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ts INTEGER NOT NULL,
  user_id INTEGER,
  username TEXT,
  action TEXT NOT NULL,
  target TEXT,
  result TEXT NOT NULL,
  ip TEXT,
  details TEXT
);
CREATE INDEX audit_ts ON audit_log(ts);
CREATE TABLE docker_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ts_nano INTEGER NOT NULL,
  type TEXT NOT NULL,
  action TEXT NOT NULL,
  actor_id TEXT,
  actor_name TEXT,
  severity TEXT NOT NULL,
  attributes TEXT
);
CREATE INDEX docker_events_ts ON docker_events(ts_nano);
CREATE INDEX docker_events_actor ON docker_events(actor_id, ts_nano);
CREATE TABLE alerts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  rule TEXT NOT NULL,
  target TEXT NOT NULL,
  target_name TEXT,
  severity TEXT NOT NULL,
  message TEXT NOT NULL,
  value REAL,
  threshold REAL,
  started_at INTEGER NOT NULL,
  resolved_at INTEGER,
  acknowledged INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX alerts_started ON alerts(started_at);
CREATE INDEX alerts_open ON alerts(resolved_at);
CREATE TABLE series (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  key TEXT NOT NULL UNIQUE
);
CREATE TABLE points_raw (
  series_id INTEGER NOT NULL,
  ts INTEGER NOT NULL,
  avg REAL NOT NULL, min REAL NOT NULL, max REAL NOT NULL,
  PRIMARY KEY (series_id, ts)
) WITHOUT ROWID;
CREATE TABLE points_5m (
  series_id INTEGER NOT NULL,
  ts INTEGER NOT NULL,
  avg REAL NOT NULL, min REAL NOT NULL, max REAL NOT NULL,
  PRIMARY KEY (series_id, ts)
) WITHOUT ROWID;
CREATE TABLE points_1h (
  series_id INTEGER NOT NULL,
  ts INTEGER NOT NULL,
  avg REAL NOT NULL, min REAL NOT NULL, max REAL NOT NULL,
  PRIMARY KEY (series_id, ts)
) WITHOUT ROWID;
CREATE INDEX points_raw_ts ON points_raw(ts);
CREATE INDEX points_5m_ts ON points_5m(ts);
CREATE INDEX points_1h_ts ON points_1h(ts);
CREATE TABLE container_names (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  image TEXT,
  last_seen INTEGER NOT NULL
);
`,
}

func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.DB.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, applied_at INTEGER NOT NULL)`); err != nil {
		return fmt.Errorf("migrations table: %w", err)
	}
	var current int
	if err := s.DB.QueryRowContext(ctx, `SELECT COALESCE(MAX(version),0) FROM schema_migrations`).Scan(&current); err != nil {
		return err
	}
	for i := current; i < len(migrations); i++ {
		tx, err := s.DB.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, migrations[i]); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: %w", i+1, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(version, applied_at) VALUES (?, ?)`, i+1, time.Now().Unix()); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

