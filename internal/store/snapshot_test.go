package store

import (
	"strings"
	"testing"

	"task279-monocache/internal/model"
)

// TestPublishSnapshotFreezesNote 验证发布瞬间的 note 被冻结写盘，
// 且发布后 GetSnapshot 读回的 note 即发布时写入值，不被默认空 note 覆盖。
func TestPublishSnapshotFreezesNote(t *testing.T) {
	db := newTestDB(t)

	batch := &model.CompilationBatch{Name: "b", Status: model.BatchReceiving}
	if err := db.CreateBatch(batch); err != nil {
		t.Fatalf("create batch: %v", err)
	}
	snap := &model.VerificationSnapshot{BatchID: batch.ID, Status: model.SnapDraft}
	if err := db.CreateSnapshot(snap); err != nil {
		t.Fatalf("create snapshot: %v", err)
	}
	const frozen = "coherent=false total=2 unique=1 duplicate=0 conflict=1 abi_mismatch=0"
	if err := db.PublishSnapshot(snap.ID, frozen); err != nil {
		t.Fatalf("publish: %v", err)
	}
	got, err := db.GetSnapshot(snap.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != model.SnapPublished {
		t.Fatalf("status=%s want published", got.Status)
	}
	if got.Note != frozen {
		t.Fatalf("note drifted: got %q want %q", got.Note, frozen)
	}
}

// TestPublishSnapshotEmptyNoteKeepsDraftNote 验证发布时不传 note 则保留
// 草稿创建时写入的 note（发布不得清空既有 note）。
func TestPublishSnapshotEmptyNoteKeepsDraftNote(t *testing.T) {
	db := newTestDB(t)

	batch := &model.CompilationBatch{Name: "b", Status: model.BatchReceiving}
	if err := db.CreateBatch(batch); err != nil {
		t.Fatalf("create batch: %v", err)
	}
	const draft = "draft-stage-note"
	snap := &model.VerificationSnapshot{BatchID: batch.ID, Status: model.SnapDraft, Note: draft}
	if err := db.CreateSnapshot(snap); err != nil {
		t.Fatalf("create snapshot: %v", err)
	}
	// 同批先发布另一份，确保不影响目标快照。
	other := &model.VerificationSnapshot{BatchID: batch.ID, Status: model.SnapDraft}
	if err := db.CreateSnapshot(other); err != nil {
		t.Fatalf("create other: %v", err)
	}
	_ = db.PublishSnapshot(other.ID, "other")

	if err := db.PublishSnapshot(snap.ID, ""); err != nil {
		t.Fatalf("publish: %v", err)
	}
	got, err := db.GetSnapshot(snap.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !strings.Contains(got.Note, draft) {
		t.Fatalf("empty-note publish wiped draft note: got %q", got.Note)
	}
}
