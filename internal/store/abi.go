package store

import (
	"database/sql"

	"task279-monocache/internal/model"
)

// CreateABI 持久化一个目标 ABI 版本。
func (db *DB) CreateABI(a *model.ABIVersion) error {
	if a.ID == "" {
		a.ID = model.GenID("abi")
	}
	if a.CreatedAt == "" {
		a.CreatedAt = nowRFC3339()
	}
	if a.Spec == "" {
		a.Spec = "{}"
	}
	if _, err := db.Exec(
		`INSERT INTO abis(id,name,version,spec,created_at) VALUES(?,?,?,?,?)`,
		a.ID, a.Name, a.Version, a.Spec, a.CreatedAt,
	); err != nil {
		return model.NewError("CreateABI", err)
	}
	return nil
}

// GetABI 按 ID 读取 ABI。
func (db *DB) GetABI(id string) (*model.ABIVersion, error) {
	row := db.QueryRow(`SELECT id,name,version,spec,created_at FROM abis WHERE id=?`, id)
	a := &model.ABIVersion{}
	if err := row.Scan(&a.ID, &a.Name, &a.Version, &a.Spec, &a.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, model.NewError("GetABI", err)
	}
	return a, nil
}

// ListABIs 返回全部 ABI。
func (db *DB) ListABIs() ([]*model.ABIVersion, error) {
	rows, err := db.Query(`SELECT id,name,version,spec,created_at FROM abis ORDER BY created_at`)
	if err != nil {
		return nil, model.NewError("ListABIs", err)
	}
	defer rows.Close()
	var out []*model.ABIVersion
	for rows.Next() {
		a := &model.ABIVersion{}
		if err := rows.Scan(&a.ID, &a.Name, &a.Version, &a.Spec, &a.CreatedAt); err != nil {
			return nil, model.NewError("ListABIs", err)
		}
		out = append(out, a)
	}
	return out, nil
}
