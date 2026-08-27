package store

import (
	"database/sql"

	"task279-monocache/internal/model"
)

// CreateRequest 持久化一个实例请求。
func (db *DB) CreateRequest(r *model.InstanceRequest) error {
	if r.ID == "" {
		r.ID = model.GenID("req")
	}
	if r.CreatedAt == "" {
		r.CreatedAt = nowRFC3339()
	}
	if r.Status == "" {
		r.Status = model.ReqRaw
	}
	if r.ArgIDs == "" {
		r.ArgIDs = "[]"
	}
	if r.ConstraintIDs == "" {
		r.ConstraintIDs = "[]"
	}
	if _, err := db.Exec(
		`INSERT INTO requests(id,batch_id,def_id,abi_id,arg_ids,constraint_ids,status,created_at,normalized_at)
		 VALUES(?,?,?,?,?,?,?,?,?)`,
		r.ID, r.BatchID, r.DefID, r.ABIID, r.ArgIDs, r.ConstraintIDs, r.Status, r.CreatedAt, r.NormalizedAt,
	); err != nil {
		return model.NewError("CreateRequest", err)
	}
	return nil
}

// GetRequest 按 ID 读取实例请求。
func (db *DB) GetRequest(id string) (*model.InstanceRequest, error) {
	row := db.QueryRow(`SELECT id,batch_id,def_id,abi_id,arg_ids,constraint_ids,status,created_at,normalized_at FROM requests WHERE id=?`, id)
	r := &model.InstanceRequest{}
	var norm string
	if err := row.Scan(&r.ID, &r.BatchID, &r.DefID, &r.ABIID, &r.ArgIDs, &r.ConstraintIDs, &r.Status, &r.CreatedAt, &norm); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, model.NewError("GetRequest", err)
	}
	r.NormalizedAt = norm
	return r, nil
}

// ListRequestsByBatch 按批次返回实例请求。
func (db *DB) ListRequestsByBatch(batchID string) ([]*model.InstanceRequest, error) {
	rows, err := db.Query(`SELECT id,batch_id,def_id,abi_id,arg_ids,constraint_ids,status,created_at,normalized_at FROM requests WHERE batch_id=? ORDER BY created_at`, batchID)
	if err != nil {
		return nil, model.NewError("ListRequestsByBatch", err)
	}
	defer rows.Close()
	var out []*model.InstanceRequest
	for rows.Next() {
		r := &model.InstanceRequest{}
		var norm string
		if err := rows.Scan(&r.ID, &r.BatchID, &r.DefID, &r.ABIID, &r.ArgIDs, &r.ConstraintIDs, &r.Status, &r.CreatedAt, &norm); err != nil {
			return nil, model.NewError("ListRequestsByBatch", err)
		}
		r.NormalizedAt = norm
		out = append(out, r)
	}
	return out, nil
}

// UpdateRequest 按 ID 更新实例请求的可变字段（状态与规范化时间）。
func (db *DB) UpdateRequest(r *model.InstanceRequest) error {
	if !model.ValidRequestStatus(r.Status) {
		return model.NewError("UpdateRequest", model.ErrInvalid)
	}
	if _, err := db.Exec(
		`UPDATE requests SET status=?, normalized_at=? WHERE id=?`,
		r.Status, r.NormalizedAt, r.ID,
	); err != nil {
		return model.NewError("UpdateRequest", err)
	}
	return nil
}
