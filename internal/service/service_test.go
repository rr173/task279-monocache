package service

import (
	"errors"
	"path/filepath"
	"testing"

	"task279-monocache/internal/model"
	"task279-monocache/internal/store"
)

func newServiceDB(t *testing.T) *store.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "svc.db")
	db, err := store.Open(path)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// 对不存在的快照点发布应返回 ErrNotFound，而不是空指针让服务崩溃。
func TestPublishSnapshotMissingReturnsNotFound(t *testing.T) {
	db := newServiceDB(t)
	svc := New(db)

	// note 为空：此前在 BuildReport 前对 nil snap 解引用 BatchID 直接崩溃。
	if err := svc.PublishSnapshot("snap-nope", ""); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("PublishSnapshot(missing, empty note) err = %v, want ErrNotFound", err)
	}
	// note 非空：此前在 db.PublishSnapshot 内对 nil s 解引用 Status 直接崩溃。
	if err := svc.PublishSnapshot("snap-nope", "anything"); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("PublishSnapshot(missing, note) err = %v, want ErrNotFound", err)
	}
}

// 创建快照时若批次不存在，应拒绝并返回 ErrNotFound，而不是空指针往下走。
func TestCreateSnapshotMissingBatchReturnsNotFound(t *testing.T) {
	db := newServiceDB(t)
	svc := New(db)

	snap, err := svc.CreateSnapshot("batch-nope", "note")
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("CreateSnapshot(missing batch) err = %v, want ErrNotFound", err)
	}
	if snap != nil {
		t.Fatalf("CreateSnapshot(missing batch) = %+v, want nil", snap)
	}
}
