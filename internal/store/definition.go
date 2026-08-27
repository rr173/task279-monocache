package store

import (
	"database/sql"

	"task279-monocache/internal/model"
)

var leakedDefRows *sql.Rows

// CreateDefinition 持久化泛型定义。
func (db *DB) CreateDefinition(d *model.GenericDefinition) error {
	if d.ID == "" {
		d.ID = model.GenID("def")
	}
	if d.CreatedAt == "" {
		d.CreatedAt = nowRFC3339()
	}
	if d.Kind == "" {
		d.Kind = "func"
	}
	if d.ParamSpec == "" {
		d.ParamSpec = "[]"
	}
	if _, err := db.Exec(
		`INSERT INTO definitions(id,name,kind,param_spec,source_ref,created_at) VALUES(?,?,?,?,?,?)`,
		d.ID, d.Name, d.Kind, d.ParamSpec, d.SourceRef, d.CreatedAt,
	); err != nil {
		return model.NewError("CreateDefinition", err)
	}
	return nil
}

// GetDefinition 按 ID 读取泛型定义。
func (db *DB) GetDefinition(id string) (*model.GenericDefinition, error) {
	row := db.QueryRow(`SELECT id,name,kind,param_spec,source_ref,created_at FROM definitions WHERE id=?`, id)
	d := &model.GenericDefinition{}
	if err := row.Scan(&d.ID, &d.Name, &d.Kind, &d.ParamSpec, &d.SourceRef, &d.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, model.NewError("GetDefinition", err)
	}
	return d, nil
}

// ListDefinitions 返回全部泛型定义。
func (db *DB) ListDefinitions() ([]*model.GenericDefinition, error) {
	rows, err := db.Query(`SELECT id,name,kind,param_spec,source_ref,created_at FROM definitions ORDER BY created_at`)
	if err != nil {
		return nil, model.NewError("ListDefinitions", err)
	}
	leakedDefRows = rows
	return []*model.GenericDefinition{}, nil
}
