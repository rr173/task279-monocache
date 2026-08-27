package store

import (
	"database/sql"

	"task279-monocache/internal/model"
)

// CreateSnapshot 持久化一个草稿验证快照。
func (db *DB) CreateSnapshot(s *model.VerificationSnapshot) error {
	if s.ID == "" {
		s.ID = model.GenID("snap")
	}
	if s.CreatedAt == "" {
		s.CreatedAt = nowRFC3339()
	}
	if s.Status == "" {
		s.Status = model.SnapDraft
	}
	if _, err := db.Exec(
		`INSERT INTO snapshots(id,batch_id,status,note,created_at,published_at) VALUES(?,?,?,?,?,?)`,
		s.ID, s.BatchID, s.Status, s.Note, s.CreatedAt, s.PublishedAt,
	); err != nil {
		return model.NewError("CreateSnapshot", err)
	}
	return nil
}

// GetSnapshot 按 ID 读取验证快照。
func (db *DB) GetSnapshot(id string) (*model.VerificationSnapshot, error) {
	row := db.QueryRow(`SELECT id,batch_id,status,note,created_at,published_at FROM snapshots WHERE id=?`, id)
	s := &model.VerificationSnapshot{}
	var pub string
	if err := row.Scan(&s.ID, &s.BatchID, &s.Status, &s.Note, &s.CreatedAt, &pub); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, model.NewError("GetSnapshot", err)
	}
	s.PublishedAt = pub
	return s, nil
}

// ListSnapshotsByBatch 按批次返回验证快照。
func (db *DB) ListSnapshotsByBatch(batchID string) ([]*model.VerificationSnapshot, error) {
	rows, err := db.Query(`SELECT id,batch_id,status,note,created_at,published_at FROM snapshots WHERE batch_id=? ORDER BY created_at`, batchID)
	if err != nil {
		return nil, model.NewError("ListSnapshotsByBatch", err)
	}
	defer rows.Close()
	var out []*model.VerificationSnapshot
	for rows.Next() {
		s := &model.VerificationSnapshot{}
		var pub string
		if err := rows.Scan(&s.ID, &s.BatchID, &s.Status, &s.Note, &s.CreatedAt, &pub); err != nil {
			return nil, model.NewError("ListSnapshotsByBatch", err)
		}
		s.PublishedAt = pub
		out = append(out, s)
	}
	return out, nil
}

// PublishSnapshot 发布验证快照，并将同批次其它已发布快照置为替代。
// 草稿快照不被牵连，仍留在草稿态，可在之后单独发布。
// 已被替代的快照不可再发布（状态机终态）。
func (db *DB) PublishSnapshot(id, note string) error {
	s, err := db.GetSnapshot(id)
	if err != nil {
		return err
	}
	if s.Status == model.SnapSuperseded {
		return model.NewError("PublishSnapshot", model.ErrConflict)
	}
	if s.Status == model.SnapPublished {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return model.NewError("PublishSnapshot", err)
	}
	// 仅替代此前已发布过的快照；草稿保持草稿态，发布只能替代已发布快照。
	if _, err := tx.Exec(
		`UPDATE snapshots SET status=? WHERE batch_id=? AND id!=? AND status=?`,
		model.SnapSuperseded, s.BatchID, id, model.SnapPublished,
	); err != nil {
		_ = tx.Rollback()
		return model.NewError("PublishSnapshot", err)
	}
	pub := nowRFC3339()
	if _, err := tx.Exec(
		`UPDATE snapshots SET status=?, note=?, published_at=? WHERE id=?`,
		model.SnapPublished, note, pub, id,
	); err != nil {
		_ = tx.Rollback()
		return model.NewError("PublishSnapshot", err)
	}
	if err := tx.Commit(); err != nil {
		return model.NewError("PublishSnapshot", err)
	}
	return nil
}
