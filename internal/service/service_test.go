package service

import (
	"path/filepath"
	"strings"
	"testing"

	"task279-monocache/internal/model"
	"task279-monocache/internal/store"
)

func newSvc(t *testing.T) *Service {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "svc.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return New(db)
}

// seedConflict 在新批次下构造一对语义等价但序列化顺序不同的约束，
// 规范化并比对后处于冲突态（reqB 判 conflict），返回两请求与批次/定义信息。
func seedConflict(t *testing.T, s *Service) (batch, def, reqA, reqB string) {
	t.Helper()
	b, err := s.CreateBatch("conflict-batch")
	if err != nil {
		t.Fatalf("create batch: %v", err)
	}
	d, err := s.CreateDefinition("OrderedMap", "func", `["K","V"]`, "")
	if err != nil {
		t.Fatalf("create def: %v", err)
	}
	abi, err := s.CreateABI("amd64", "v1", "{}")
	if err != nil {
		t.Fatalf("create abi: %v", err)
	}
	arg1, err := s.AddTypeArg(d.ID, 0, "IntList", "List<Int>")
	if err != nil {
		t.Fatalf("add arg1: %v", err)
	}
	arg2, err := s.AddTypeArg(d.ID, 0, "List<Int>", "")
	if err != nil {
		t.Fatalf("add arg2: %v", err)
	}
	con1, err := s.CreateConstraint(d.ID, "", `["Ord k"]`, "solved")
	if err != nil {
		t.Fatalf("create con1: %v", err)
	}
	con2, err := s.CreateConstraint(d.ID, "", `["k Ord"]`, "solved")
	if err != nil {
		t.Fatalf("create con2: %v", err)
	}
	ra, err := s.CreateRequest(b.ID, d.ID, abi.ID, []string{arg1.ID}, []string{con1.ID})
	if err != nil {
		t.Fatalf("create reqA: %v", err)
	}
	rb, err := s.CreateRequest(b.ID, d.ID, abi.ID, []string{arg2.ID}, []string{con2.ID})
	if err != nil {
		t.Fatalf("create reqB: %v", err)
	}
	if _, err := s.NormalizeRequest(ra.ID); err != nil {
		t.Fatalf("normalize A: %v", err)
	}
	if _, _, err := s.CompareCache(ra.ID); err != nil {
		t.Fatalf("compare A: %v", err)
	}
	if _, err := s.NormalizeRequest(rb.ID); err != nil {
		t.Fatalf("normalize B: %v", err)
	}
	if _, _, err := s.CompareCache(rb.ID); err != nil {
		t.Fatalf("compare B: %v", err)
	}
	return b.ID, d.ID, ra.ID, rb.ID
}

// TestPublishedSnapshotFreezesConflictEvidence 是本任务的核心回归保护：
// 在冲突态发布快照，冻结 conflict=1 的证据；随后合并消除冲突，
// 再读已发布快照，note 仍必须是 conflict=1 / coherent=false，不得漂移到合并后一致结果。
func TestPublishedSnapshotFreezesConflictEvidence(t *testing.T) {
	s := newSvc(t)
	batch, _, reqA, reqB := seedConflict(t, s)

	snap, err := s.CreateSnapshot(batch, "")
	if err != nil {
		t.Fatalf("create snapshot: %v", err)
	}
	if err := s.PublishSnapshot(snap.ID, ""); err != nil {
		t.Fatalf("publish snapshot: %v", err)
	}
	frozen, err := s.GetSnapshot(snap.ID)
	if err != nil {
		t.Fatalf("get snapshot pre-merge: %v", err)
	}
	if !strings.Contains(frozen.Note, "conflict=1") || !strings.Contains(frozen.Note, "coherent=false") {
		t.Fatalf("note at conflict time not frozen: %q", frozen.Note)
	}

	// 合并消除冲突，随后缓存应转为一致。
	if err := s.MergeAndResolve([]string{reqA, reqB}); err != nil {
		t.Fatalf("merge: %v", err)
	}
	// 再发布一份一致快照，把冲突快照置为 superseded。
	coherentSnap, err := s.CreateSnapshot(batch, "")
	if err != nil {
		t.Fatalf("create coherent snapshot: %v", err)
	}
	if err := s.PublishSnapshot(coherentSnap.ID, ""); err != nil {
		t.Fatalf("publish coherent snapshot: %v", err)
	}

	// 已 superseded 的冲突快照，冻结证据必须原样保留。
	got, err := s.GetSnapshot(snap.ID)
	if err != nil {
		t.Fatalf("get snapshot post-merge: %v", err)
	}
	if got.Status != model.SnapSuperseded {
		t.Fatalf("status=%s want superseded", got.Status)
	}
	if !strings.Contains(got.Note, "conflict=1") || !strings.Contains(got.Note, "coherent=false") {
		t.Fatalf("published snapshot note drifted after merge: %q", got.Note)
	}

	// 合并后发布的一致快照冻结的是 conflict=0。
	coh, err := s.GetSnapshot(coherentSnap.ID)
	if err != nil {
		t.Fatalf("get coherent snapshot: %v", err)
	}
	if !strings.Contains(coh.Note, "conflict=0") || !strings.Contains(coh.Note, "coherent=true") {
		t.Fatalf("coherent snapshot note not frozen at coherent state: %q", coh.Note)
	}

	// 列出快照，两份 note 都应与各自冻结点一致，不被覆盖。
	all, err := s.ListSnapshots(batch)
	if err != nil {
		t.Fatalf("list snapshots: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(all))
	}
	for _, sn := range all {
		switch sn.ID {
		case snap.ID:
			if !strings.Contains(sn.Note, "conflict=1") {
				t.Fatalf("listed conflict snapshot note drifted: %q", sn.Note)
			}
		case coherentSnap.ID:
			if !strings.Contains(sn.Note, "conflict=0") {
				t.Fatalf("listed coherent snapshot note drifted: %q", sn.Note)
			}
		}
	}
}

// TestPublishedSnapshotFreezesAcrossRestart 验证重启恢复后冻结证据仍不漂移。
func TestPublishedSnapshotFreezesAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "restart.db")
	db, err := store.Open(path)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	s := New(db)
	batch, _, reqA, reqB := seedConflict(t, s)
	snap, err := s.CreateSnapshot(batch, "")
	if err != nil {
		t.Fatalf("create snapshot: %v", err)
	}
	if err := s.PublishSnapshot(snap.ID, ""); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if err := s.MergeAndResolve([]string{reqA, reqB}); err != nil {
		t.Fatalf("merge: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	db2, err := store.Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer db2.Close()
	s2 := New(db2)
	got, err := s2.GetSnapshot(snap.ID)
	if err != nil {
		t.Fatalf("get after restart: %v", err)
	}
	if !strings.Contains(got.Note, "conflict=1") || !strings.Contains(got.Note, "coherent=false") {
		t.Fatalf("frozen note drifted across restart: %q", got.Note)
	}
}
