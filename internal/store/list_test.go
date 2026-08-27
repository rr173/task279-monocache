package store

import (
	"path/filepath"
	"testing"
	"time"

	"task279-monocache/internal/model"
)

// newTestDB 打开一个临时文件 SQLite，用于隔离测试。
func newTestDB(t *testing.T) *DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// TestListThenWriteDoesNotBlock 复现"列出全部 ABI / 按批次列出快照后，
// 后续写入被卡死"的连接泄漏回归：ListXxx 必须归还数据库连接，
// 使紧随其后的 CreateBatch 等写入立即可执行。
//
// 修复前 ListABIs / ListSnapshotsByBatch / ListDefinitions 会把 *sql.Rows
// 存进包级全局变量且从不关闭；由于 SetMaxOpenConns(1)，这条唯一连接
// 被打开的 Rows 独占，后续写入会一直等待空闲连接而卡死。这里用带超时
// 的 goroutine 包住写入，若连接未归还则以超时失败而非挂死收尾。
func TestListThenWriteDoesNotBlock(t *testing.T) {
	db := newTestDB(t)

	// 先写入若干 ABI 与一条快照，作为列表返回内容。
	for i := 0; i < 3; i++ {
		if err := db.CreateABI(&model.ABIVersion{Name: "a", Version: "v"}); err != nil {
			t.Fatalf("CreateABI: %v", err)
		}
	}
	b := &model.CompilationBatch{Name: "b"}
	if err := db.CreateBatch(b); err != nil {
		t.Fatalf("CreateBatch: %v", err)
	}
	if err := db.CreateSnapshot(&model.VerificationSnapshot{BatchID: b.ID}); err != nil {
		t.Fatalf("CreateSnapshot: %v", err)
	}

	// 列出全部目标 ABI 与按批次列出快照——历史上这两步会泄漏 *sql.Rows。
	abis, err := db.ListABIs()
	if err != nil {
		t.Fatalf("ListABIs: %v", err)
	}
	if len(abis) != 3 {
		t.Fatalf("ListABIs returned %d abis, want 3", len(abis))
	}
	snaps, err := db.ListSnapshotsByBatch(b.ID)
	if err != nil {
		t.Fatalf("ListSnapshotsByBatch: %v", err)
	}
	if len(snaps) != 1 {
		t.Fatalf("ListSnapshotsByBatch returned %d, want 1", len(snaps))
	}

	// 列出后，后续写入必须立刻能执行：若连接未归还，此处会卡死，
	// done 通道在超时前不会收到结果。
	done := make(chan error, 1)
	go func() {
		done <- db.CreateBatch(&model.CompilationBatch{Name: "after-list"})
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("CreateBatch after ListABIs/ListSnapshots: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("CreateBatch after list hung: DB connection leaked by List*")
	}

	// 额外确认 ListDefinitions 同样不泄漏连接，且确实返回落盘数据。
	d := &model.GenericDefinition{Name: "Map", Kind: "func", ParamSpec: "[]"}
	if err := db.CreateDefinition(d); err != nil {
		t.Fatalf("CreateDefinition: %v", err)
	}
	defs, err := db.ListDefinitions()
	if err != nil {
		t.Fatalf("ListDefinitions: %v", err)
	}
	if len(defs) != 1 || defs[0].ID != d.ID {
		t.Fatalf("ListDefinitions = %+v, want single def %s", defs, d.ID)
	}
	select {
	case err := <-done:
		_ = err
	default:
	}
	done2 := make(chan error, 1)
	go func() {
		done2 <- db.CreateBatch(&model.CompilationBatch{Name: "after-defs"})
	}()
	select {
	case err := <-done2:
		if err != nil {
			t.Fatalf("CreateBatch after ListDefinitions: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("CreateBatch after ListDefinitions hung: DB connection leaked")
	}
}
