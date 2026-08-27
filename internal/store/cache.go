package store

import (
	"database/sql"

	"task279-monocache/internal/model"
)

// SaveCacheEntry 持久化一个实例缓存条目。
func (db *DB) SaveCacheEntry(e *model.CacheEntry) error {
	if e.ID == "" {
		e.ID = model.GenID("cache")
	}
	if e.CreatedAt == "" {
		e.CreatedAt = nowRFC3339()
	}
	if e.Status == "" {
		e.Status = model.CacheCandidate
	}
	if _, err := db.Exec(
		`INSERT INTO cache_entries(id,def_id,key_string,arg_set_hash,request_id,abi_id,status,created_at) VALUES(?,?,?,?,?,?,?,?)`,
		e.ID, e.DefID, e.KeyString, e.ArgSetHash, e.RequestID, e.ABIID, e.Status, e.CreatedAt,
	); err != nil {
		return model.NewError("SaveCacheEntry", err)
	}
	return nil
}

// GetCacheEntry 按 ID 读取缓存条目。
func (db *DB) GetCacheEntry(id string) (*model.CacheEntry, error) {
	row := db.QueryRow(`SELECT id,def_id,key_string,arg_set_hash,request_id,abi_id,status,created_at FROM cache_entries WHERE id=?`, id)
	e := &model.CacheEntry{}
	if err := row.Scan(&e.ID, &e.DefID, &e.KeyString, &e.ArgSetHash, &e.RequestID, &e.ABIID, &e.Status, &e.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, model.NewError("GetCacheEntry", err)
	}
	return e, nil
}

// GetCacheByKeyABI 返回与给定键串、ABI 完全匹配的缓存条目（若存在）。
func (db *DB) GetCacheByKeyABI(keyString, abiID string) (*model.CacheEntry, error) {
	row := db.QueryRow(`SELECT id,def_id,key_string,arg_set_hash,request_id,abi_id,status,created_at FROM cache_entries WHERE key_string=? AND abi_id=? ORDER BY created_at DESC LIMIT 1`, keyString, abiID)
	e := &model.CacheEntry{}
	if err := row.Scan(&e.ID, &e.DefID, &e.KeyString, &e.ArgSetHash, &e.RequestID, &e.ABIID, &e.Status, &e.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, model.NewError("GetCacheByKeyABI", err)
	}
	return e, nil
}

// ListCache 返回全部缓存条目。
func (db *DB) ListCache() ([]*model.CacheEntry, error) {
	rows, err := db.Query(`SELECT id,def_id,key_string,arg_set_hash,request_id,abi_id,status,created_at FROM cache_entries ORDER BY created_at`)
	if err != nil {
		return nil, model.NewError("ListCache", err)
	}
	defer rows.Close()
	var out []*model.CacheEntry
	for rows.Next() {
		e := &model.CacheEntry{}
		if err := rows.Scan(&e.ID, &e.DefID, &e.KeyString, &e.ArgSetHash, &e.RequestID, &e.ABIID, &e.Status, &e.CreatedAt); err != nil {
			return nil, model.NewError("ListCache", err)
		}
		out = append(out, e)
	}
	return out, nil
}

// GetCacheByRequest 返回某实例请求对应的缓存条目（每请求至多一条）。
func (db *DB) GetCacheByRequest(reqID string) (*model.CacheEntry, error) {
	row := db.QueryRow(`SELECT id,def_id,key_string,arg_set_hash,request_id,abi_id,status,created_at FROM cache_entries WHERE request_id=? ORDER BY created_at DESC LIMIT 1`, reqID)
	e := &model.CacheEntry{}
	if err := row.Scan(&e.ID, &e.DefID, &e.KeyString, &e.ArgSetHash, &e.RequestID, &e.ABIID, &e.Status, &e.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, model.NewError("GetCacheByRequest", err)
	}
	return e, nil
}

// ListCacheByBatch 返回属于某批次（经由请求关联）的全部缓存条目。
func (db *DB) ListCacheByBatch(batchID string) ([]*model.CacheEntry, error) {
	rows, err := db.Query(
		`SELECT c.id,c.def_id,c.key_string,c.arg_set_hash,c.request_id,c.abi_id,c.status,c.created_at
		 FROM cache_entries c JOIN requests r ON c.request_id=r.id WHERE r.batch_id=? ORDER BY c.created_at`, batchID)
	if err != nil {
		return nil, model.NewError("ListCacheByBatch", err)
	}
	defer rows.Close()
	var out []*model.CacheEntry
	for rows.Next() {
		e := &model.CacheEntry{}
		if err := rows.Scan(&e.ID, &e.DefID, &e.KeyString, &e.ArgSetHash, &e.RequestID, &e.ABIID, &e.Status, &e.CreatedAt); err != nil {
			return nil, model.NewError("ListCacheByBatch", err)
		}
		out = append(out, e)
	}
	return out, nil
}

// ListCacheByDef 按定义返回缓存条目。
func (db *DB) ListCacheByDef(defID string) ([]*model.CacheEntry, error) {
	rows, err := db.Query(`SELECT id,def_id,key_string,arg_set_hash,request_id,abi_id,status,created_at FROM cache_entries WHERE def_id=? ORDER BY created_at`, defID)
	if err != nil {
		return nil, model.NewError("ListCacheByDef", err)
	}
	defer rows.Close()
	var out []*model.CacheEntry
	for rows.Next() {
		e := &model.CacheEntry{}
		if err := rows.Scan(&e.ID, &e.DefID, &e.KeyString, &e.ArgSetHash, &e.RequestID, &e.ABIID, &e.Status, &e.CreatedAt); err != nil {
			return nil, model.NewError("ListCacheByDef", err)
		}
		out = append(out, e)
	}
	return out, nil
}

// SetCacheStatus 更新缓存条目状态。
func (db *DB) SetCacheStatus(id, status string) error {
	if _, err := db.Exec(`UPDATE cache_entries SET status=? WHERE id=?`, status, id); err != nil {
		return model.NewError("SetCacheStatus", err)
	}
	return nil
}

// UpdateCacheEntry 按 ID 更新缓存条目字段（用于同请求缓存条目 upsert）。
func (db *DB) UpdateCacheEntry(e *model.CacheEntry) error {
	if _, err := db.Exec(
		`UPDATE cache_entries SET def_id=?, key_string=?, arg_set_hash=?, abi_id=?, status=? WHERE id=?`,
		e.DefID, e.KeyString, e.ArgSetHash, e.ABIID, e.Status, e.ID,
	); err != nil {
		return model.NewError("UpdateCacheEntry", err)
	}
	return nil
}
