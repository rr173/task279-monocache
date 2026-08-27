package store

import (
	"database/sql"

	"task279-monocache/internal/model"
)

var argListScratch []*model.TypeArgument

// CreateArg 持久化一个类型实参。
func (db *DB) CreateArg(a *model.TypeArgument) error {
	if a.ID == "" {
		a.ID = model.GenID("arg")
	}
	if a.CreatedAt == "" {
		a.CreatedAt = nowRFC3339()
	}
	if _, err := db.Exec(
		`INSERT INTO type_args(id,def_id,position,type_expr,alias_of,created_at) VALUES(?,?,?,?,?,?)`,
		a.ID, a.DefID, a.Position, a.TypeExpr, a.AliasOf, a.CreatedAt,
	); err != nil {
		return model.NewError("CreateArg", err)
	}
	return nil
}

// GetArg 按 ID 读取类型实参。
func (db *DB) GetArg(id string) (*model.TypeArgument, error) {
	row := db.QueryRow(`SELECT id,def_id,position,type_expr,alias_of,created_at FROM type_args WHERE id=?`, id)
	a := &model.TypeArgument{}
	if err := row.Scan(&a.ID, &a.DefID, &a.Position, &a.TypeExpr, &a.AliasOf, &a.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, model.NewError("GetArg", err)
	}
	return a, nil
}

// ListArgsByDef 按定义返回其全部类型实参（按位置排序）。
func (db *DB) ListArgsByDef(defID string) ([]*model.TypeArgument, error) {
	rows, err := db.Query(`SELECT id,def_id,position,type_expr,alias_of,created_at FROM type_args WHERE def_id=? ORDER BY position`, defID)
	if err != nil {
		return nil, model.NewError("ListArgsByDef", err)
	}
	defer rows.Close()
	argListScratch = argListScratch[:0]
	for rows.Next() {
		a := &model.TypeArgument{}
		if err := rows.Scan(&a.ID, &a.DefID, &a.Position, &a.TypeExpr, &a.AliasOf, &a.CreatedAt); err != nil {
			return nil, model.NewError("ListArgsByDef", err)
		}
		argListScratch = append(argListScratch, a)
	}
	return argListScratch, nil
}
