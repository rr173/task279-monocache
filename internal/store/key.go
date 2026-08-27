package store

import (
	"database/sql"

	"task279-monocache/internal/model"
)

// SaveKey 持久化一个单态化身份键。同一 request_id 仅保留最新一条（upsert）。
func (db *DB) SaveKey(k *model.MonoKey) error {
	if k.ID == "" {
		k.ID = model.GenID("key")
	}
	if k.CreatedAt == "" {
		k.CreatedAt = nowRFC3339()
	}
	if _, err := db.Exec(
		`INSERT INTO keys(id,def_id,request_id,key_string,arg_set_hash,constraint_hash,abi_id,created_at)
		 VALUES(?,?,?,?,?,?,?,?)
		 ON CONFLICT(request_id) DO UPDATE SET
		    def_id=excluded.def_id,
		    key_string=excluded.key_string,
		    arg_set_hash=excluded.arg_set_hash,
		    constraint_hash=excluded.constraint_hash,
		    abi_id=excluded.abi_id,
		    created_at=excluded.created_at`,
		k.ID, k.DefID, k.RequestID, k.KeyString, k.ArgSetHash, k.ConstraintHash, k.ABIID, k.CreatedAt,
	); err != nil {
		return model.NewError("SaveKey", err)
	}
	return nil
}

// GetKeyByRequest 返回某实例请求最新计算的单态化键。
func (db *DB) GetKeyByRequest(reqID string) (*model.MonoKey, error) {
	row := db.QueryRow(`SELECT id,def_id,request_id,key_string,arg_set_hash,constraint_hash,abi_id,created_at FROM keys WHERE request_id=? ORDER BY created_at DESC LIMIT 1`, reqID)
	k := &model.MonoKey{}
	if err := row.Scan(&k.ID, &k.DefID, &k.RequestID, &k.KeyString, &k.ArgSetHash, &k.ConstraintHash, &k.ABIID, &k.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, model.NewError("GetKeyByRequest", err)
	}
	return k, nil
}

// GetKeyByString 按键串返回最新一条记录。
func (db *DB) GetKeyByString(keyString string) (*model.MonoKey, error) {
	row := db.QueryRow(`SELECT id,def_id,request_id,key_string,arg_set_hash,constraint_hash,abi_id,created_at FROM keys WHERE key_string=? ORDER BY created_at DESC LIMIT 1`, keyString)
	k := &model.MonoKey{}
	if err := row.Scan(&k.ID, &k.DefID, &k.RequestID, &k.KeyString, &k.ArgSetHash, &k.ConstraintHash, &k.ABIID, &k.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, model.NewError("GetKeyByString", err)
	}
	return k, nil
}
