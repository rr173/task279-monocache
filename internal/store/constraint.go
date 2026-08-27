package store

import (
	"database/sql"

	"task279-monocache/internal/model"
)

// CreateConstraint 持久化一个约束解。
func (db *DB) CreateConstraint(c *model.ConstraintSolution) error {
	if c.ID == "" {
		c.ID = model.GenID("con")
	}
	if c.CreatedAt == "" {
		c.CreatedAt = nowRFC3339()
	}
	if c.Status == "" {
		c.Status = "solved"
	}
	if c.SolvedConstraints == "" {
		c.SolvedConstraints = "[]"
	}
	if _, err := db.Exec(
		`INSERT INTO constraints(id,def_id,arg_set_hash,solved_constraints,status,created_at) VALUES(?,?,?,?,?,?)`,
		c.ID, c.DefID, c.ArgSetHash, c.SolvedConstraints, c.Status, c.CreatedAt,
	); err != nil {
		return model.NewError("CreateConstraint", err)
	}
	return nil
}

// GetConstraint 按 ID 读取约束解。
func (db *DB) GetConstraint(id string) (*model.ConstraintSolution, error) {
	row := db.QueryRow(`SELECT id,def_id,arg_set_hash,solved_constraints,status,created_at FROM constraints WHERE id=?`, id)
	c := &model.ConstraintSolution{}
	if err := row.Scan(&c.ID, &c.DefID, &c.ArgSetHash, &c.SolvedConstraints, &c.Status, &c.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, model.NewError("GetConstraint", err)
	}
	return c, nil
}

// ListConstraintsByDef 按定义返回约束解列表。
func (db *DB) ListConstraintsByDef(defID string) ([]*model.ConstraintSolution, error) {
	rows, err := db.Query(`SELECT id,def_id,arg_set_hash,solved_constraints,status,created_at FROM constraints WHERE def_id=? ORDER BY created_at`, defID)
	if err != nil {
		return nil, model.NewError("ListConstraintsByDef", err)
	}
	defer rows.Close()
	var out []*model.ConstraintSolution
	for rows.Next() {
		c := &model.ConstraintSolution{}
		if err := rows.Scan(&c.ID, &c.DefID, &c.ArgSetHash, &c.SolvedConstraints, &c.Status, &c.CreatedAt); err != nil {
			return nil, model.NewError("ListConstraintsByDef", err)
		}
		out = append(out, c)
	}
	return out, nil
}
