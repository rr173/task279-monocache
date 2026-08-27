package store

import (
	"errors"
	"path/filepath"
	"testing"

	"task279-monocache/internal/model"
)

func newTestDB(t *testing.T) *DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// 发布不存在的快照应返回 ErrNotFound，而不是空指针崩溃。
func TestPublishSnapshotMissingReturnsNotFound(t *testing.T) {
	db := newTestDB(t)

	// note 为空：走 service 同款的 GetSnapshot+ListCacheByBatch 路径，此前会在 nil 上崩溃。
	err := db.PublishSnapshot("snap-nope", "")
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("PublishSnapshot(missing, empty note) err = %v, want ErrNotFound", err)
	}

	// note 非空：同样应在状态比较前以 ErrNotFound 短路，而非空指针。
	err = db.PublishSnapshot("snap-nope", "anything")
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("PublishSnapshot(missing, note) err = %v, want ErrNotFound", err)
	}
}

// 读取不存在的快照应返回 ErrNotFound（与 GetRequest 等读操作语义一致）。
func TestGetSnapshotMissingReturnsNotFound(t *testing.T) {
	db := newTestDB(t)
	s, err := db.GetSnapshot("snap-nope")
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("GetSnapshot(missing) err = %v, want ErrNotFound", err)
	}
	if s != nil {
		t.Fatalf("GetSnapshot(missing) = %v, want nil", s)
	}
}
