package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// DB 封装 *sql.DB，提供迁移与统一访问入口。
type DB struct {
	*sql.DB
}

// Open 打开（必要时创建）SQLite 数据库并执行迁移。
func Open(path string) (*DB, error) {
	sqldb, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", path, err)
	}
	// SQLite 写并发受限，单连接可避免数据库锁竞争。
	sqldb.SetMaxOpenConns(1)
	if _, err := sqldb.Exec("PRAGMA busy_timeout=5000"); err != nil {
		_ = sqldb.Close()
		return nil, fmt.Errorf("pragma busy_timeout: %w", err)
	}
	if err := sqldb.Ping(); err != nil {
		_ = sqldb.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	db := &DB{sqldb}
	if err := db.migrate(); err != nil {
		_ = sqldb.Close()
		return nil, err
	}
	return db, nil
}

func (db *DB) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS abis (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			version TEXT NOT NULL,
			spec TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS definitions (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			kind TEXT NOT NULL,
			param_spec TEXT NOT NULL DEFAULT '[]',
			source_ref TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS type_args (
			id TEXT PRIMARY KEY,
			def_id TEXT NOT NULL,
			position INTEGER NOT NULL,
			type_expr TEXT NOT NULL,
			alias_of TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS constraints (
			id TEXT PRIMARY KEY,
			def_id TEXT NOT NULL,
			arg_set_hash TEXT NOT NULL DEFAULT '',
			solved_constraints TEXT NOT NULL DEFAULT '[]',
			status TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS batches (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			status TEXT NOT NULL,
			created_at TEXT NOT NULL,
			sealed_at TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS requests (
			id TEXT PRIMARY KEY,
			batch_id TEXT NOT NULL,
			def_id TEXT NOT NULL,
			abi_id TEXT NOT NULL,
			arg_ids TEXT NOT NULL DEFAULT '[]',
			constraint_ids TEXT NOT NULL DEFAULT '[]',
			status TEXT NOT NULL,
			created_at TEXT NOT NULL,
			normalized_at TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS keys (
			id TEXT PRIMARY KEY,
			def_id TEXT NOT NULL,
			request_id TEXT NOT NULL UNIQUE,
			key_string TEXT NOT NULL,
			arg_set_hash TEXT NOT NULL,
			constraint_hash TEXT NOT NULL,
			abi_id TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS cache_entries (
			id TEXT PRIMARY KEY,
			def_id TEXT NOT NULL,
			key_string TEXT NOT NULL,
			arg_set_hash TEXT NOT NULL DEFAULT '',
			request_id TEXT NOT NULL,
			abi_id TEXT NOT NULL,
			status TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS snapshots (
			id TEXT PRIMARY KEY,
			batch_id TEXT NOT NULL,
			status TEXT NOT NULL,
			note TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			published_at TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE INDEX IF NOT EXISTS idx_args_def ON type_args(def_id)`,
		`CREATE INDEX IF NOT EXISTS idx_constraints_def ON constraints(def_id)`,
		`CREATE INDEX IF NOT EXISTS idx_requests_batch ON requests(batch_id)`,
		`CREATE INDEX IF NOT EXISTS idx_keys_string ON keys(key_string)`,
		`CREATE INDEX IF NOT EXISTS idx_cache_key_abi ON cache_entries(key_string, abi_id)`,
		`CREATE INDEX IF NOT EXISTS idx_cache_def ON cache_entries(def_id)`,
		`CREATE INDEX IF NOT EXISTS idx_cache_argset ON cache_entries(arg_set_hash, abi_id)`,
		`CREATE INDEX IF NOT EXISTS idx_snapshots_batch ON snapshots(batch_id)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}

// nowRFC3339 返回当前 UTC 时间戳（RFC3339）。
func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
