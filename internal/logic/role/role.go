// Package role 提供角色管理逻辑
//
// 角色 = 可复用的权限集合（read/write/delete），可被团队引用。
package role

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
	"github.com/zeromicro/go-zero/core/logx"
)

// RoleLogic 角色管理逻辑
type RoleLogic struct {
	ctx    context.Context
	db     *sql.DB
	logger logx.Logger
}

// NewRoleLogic 创建角色管理逻辑实例
func NewRoleLogic(ctx context.Context, db *sql.DB) *RoleLogic {
	return &RoleLogic{
		ctx:    ctx,
		db:     db,
		logger: logx.WithContext(ctx),
	}
}

// Role 角色模型
type Role struct {
	ID          int64    `db:"id" json:"id"`
	Name        string   `db:"name" json:"name"`
	Code        string   `db:"code" json:"code"`
	Description string   `db:"description" json:"description"`
	Permissions []string `db:"permissions" json:"permissions"`
	CreatedAt   string   `db:"created_at" json:"created_at"`
	UpdatedAt   string   `db:"updated_at" json:"updated_at"`
}

// CreateRole 创建角色
func (l *RoleLogic) CreateRole(name, code, description string, permissions []string) (*Role, error) {
	if code == "" {
		code = name
	}
	if permissions == nil {
		permissions = []string{}
	}

	query := `
		INSERT INTO roles (name, code, description, permissions, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id, name, code, COALESCE(description,'') AS description, permissions,
		          TO_CHAR(created_at,'YYYY-MM-DD HH24:MI:SS'),
		          TO_CHAR(updated_at,'YYYY-MM-DD HH24:MI:SS')
	`
	var r Role
	var perms []string
	if err := l.db.QueryRowContext(l.ctx, query, name, code, description, pq.Array(permissions)).Scan(
		&r.ID, &r.Name, &r.Code, &r.Description, pq.Array(&perms), &r.CreatedAt, &r.UpdatedAt); err != nil {
		l.logger.Errorf("创建角色失败: %v", err)
		return nil, fmt.Errorf("创建角色失败: %w", err)
	}
	r.Permissions = perms
	l.logger.Infof("角色创建成功: %s (ID: %d)", r.Name, r.ID)
	return &r, nil
}

// UpdateRole 更新角色
func (l *RoleLogic) UpdateRole(id int64, name, description string, permissions []string) (*Role, error) {
	original, err := l.GetRole(id)
	if err != nil {
		return nil, err
	}
	if name == "" {
		name = original.Name
	}
	if description == "" {
		description = original.Description
	}
	if permissions == nil {
		permissions = original.Permissions
	}

	query := `
		UPDATE roles SET name=$1, description=$2, permissions=$3, updated_at=NOW()
		WHERE id=$4
		RETURNING id, name, code, COALESCE(description,'') AS description, permissions,
		          TO_CHAR(created_at,'YYYY-MM-DD HH24:MI:SS'),
		          TO_CHAR(updated_at,'YYYY-MM-DD HH24:MI:SS')
	`
	var r Role
	var perms []string
	if err := l.db.QueryRowContext(l.ctx, query, name, description, pq.Array(permissions), id).Scan(
		&r.ID, &r.Name, &r.Code, &r.Description, pq.Array(&perms), &r.CreatedAt, &r.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("角色不存在: ID=%d", id)
		}
		return nil, fmt.Errorf("更新角色失败: %w", err)
	}
	r.Permissions = perms
	return &r, nil
}

// DeleteRole 删除角色
func (l *RoleLogic) DeleteRole(id int64) error {
	res, err := l.db.ExecContext(l.ctx, `DELETE FROM roles WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("删除角色失败: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("角色不存在: ID=%d", id)
	}
	return nil
}

// GetRole 获取角色
func (l *RoleLogic) GetRole(id int64) (*Role, error) {
	query := `
		SELECT id, name, code, COALESCE(description,'') AS description, permissions,
		       TO_CHAR(created_at,'YYYY-MM-DD HH24:MI:SS'),
		       TO_CHAR(updated_at,'YYYY-MM-DD HH24:MI:SS')
		FROM roles WHERE id=$1
	`
	var r Role
	var perms []string
	if err := l.db.QueryRowContext(l.ctx, query, id).Scan(
		&r.ID, &r.Name, &r.Code, &r.Description, pq.Array(&perms), &r.CreatedAt, &r.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("角色不存在: ID=%d", id)
		}
		return nil, err
	}
	r.Permissions = perms
	return &r, nil
}

// ListRoles 查询角色列表
func (l *RoleLogic) ListRoles(keyword string) ([]*Role, error) {
	query := `
		SELECT id, name, code, COALESCE(description,'') AS description, permissions,
		       TO_CHAR(created_at,'YYYY-MM-DD HH24:MI:SS'),
		       TO_CHAR(updated_at,'YYYY-MM-DD HH24:MI:SS')
		FROM roles
	`
	args := []interface{}{}
	if keyword != "" {
		query += ` WHERE name LIKE $1 OR code LIKE $1`
		args = append(args, "%"+keyword+"%")
	}
	query += ` ORDER BY id ASC`

	rows, err := l.db.QueryContext(l.ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询角色列表失败: %w", err)
	}
	defer rows.Close()

	var list []*Role
	for rows.Next() {
		var r Role
		var perms []string
		if err := rows.Scan(&r.ID, &r.Name, &r.Code, &r.Description, pq.Array(&perms), &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		r.Permissions = perms
		list = append(list, &r)
	}
	return list, rows.Err()
}
