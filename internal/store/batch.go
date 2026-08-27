package store

import (
	"database/sql"
	"fmt"

	"task279-monocache/internal/model"
)

// CreateBatch 持久化一个编译批次，缺失字段以默认值补齐。
func (db *DB) CreateBatch(b *model.CompilationBatch) error {
	if b.ID == "" {
		b.ID = model.GenID("batch")
	}
	if b.Status == "" {
		b.Status = model.BatchReceiving
	}
	if b.CreatedAt == "" {
		b.CreatedAt = nowRFC3339()
	}
	if _, err := db.Exec(
		`INSERT INTO batches(id,name,status,created_at,sealed_at) VALUES(?,?,?,?,?)`,
		b.ID, b.Name, b.Status, b.CreatedAt, b.SealedAt,
	); err != nil {
		return model.NewError("CreateBatch", err)
	}
	return nil
}

// GetBatch 按 ID 读取编译批次。
func (db *DB) GetBatch(id string) (*model.CompilationBatch, error) {
	row := db.QueryRow(`SELECT id,name,status,created_at,sealed_at FROM batches WHERE id=?`, id)
	b := &model.CompilationBatch{}
	var sealed string
	if err := row.Scan(&b.ID, &b.Name, &b.Status, &b.CreatedAt, &sealed); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, model.NewError("GetBatch", err)
	}
	b.SealedAt = sealed
	return b, nil
}

// SealBatch 将批次封存；封存后状态不可变。
func (db *DB) SealBatch(id string) error {
	b, err := db.GetBatch(id)
	if err != nil {
		return err
	}
	if b.Status == model.BatchSealed {
		return nil
	}
	if _, err := db.Exec(
		`UPDATE batches SET status=?, sealed_at=? WHERE id=?`,
		model.BatchSealed, nowRFC3339(), id,
	); err != nil {
		return model.NewError("SealBatch", err)
	}
	return nil
}

// SetBatchStatus 设置批次状态（封存的批次不可再变更）。
func (db *DB) SetBatchStatus(id, status string) error {
	if !model.ValidBatchStatus(status) {
		return model.NewError("SetBatchStatus", model.ErrInvalid)
	}
	b, err := db.GetBatch(id)
	if err != nil {
		return err
	}
	if b.Status == model.BatchSealed {
		return fmt.Errorf("SetBatchStatus: %v", model.ErrSealed)
	}
	if _, err := db.Exec(`UPDATE batches SET status=? WHERE id=?`, status, id); err != nil {
		return model.NewError("SetBatchStatus", err)
	}
	return nil
}

// ListBatches 返回全部编译批次。
func (db *DB) ListBatches() ([]*model.CompilationBatch, error) {
	rows, err := db.Query(`SELECT id,name,status,created_at,sealed_at FROM batches ORDER BY created_at`)
	if err != nil {
		return nil, model.NewError("ListBatches", err)
	}
	defer rows.Close()
	var out []*model.CompilationBatch
	for rows.Next() {
		b := &model.CompilationBatch{}
		var sealed string
		if err := rows.Scan(&b.ID, &b.Name, &b.Status, &b.CreatedAt, &sealed); err != nil {
			return nil, model.NewError("ListBatches", err)
		}
		b.SealedAt = sealed
		out = append(out, b)
	}
	return out, nil
}
