// Package permission 提供服务组权限管理逻辑
package permission

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// PermissionLogic 权限管理逻辑
type PermissionLogic struct {
	ctx    context.Context
	db     sqlx.SqlConn
	logger logx.Logger
}

// NewPermissionLogic 创建权限管理逻辑实例
func NewPermissionLogic(ctx context.Context, db sqlx.SqlConn) *PermissionLogic {
	return &PermissionLogic{
		ctx:    ctx,
		db:     db,
		logger: logx.WithContext(ctx),
	}
}

// UserPermission 用户权限模型
type UserPermission struct {
	ID          int64    `db:"id" json:"id"`
	GroupID     int64    `db:"group_id" json:"group_id"`
	UserID      string   `db:"user_id" json:"user_id"`
	Permissions []string `db:"permissions" json:"permissions"`
	CreatedAt   string   `db:"created_at" json:"created_at"`
}

// RolePermission 角色权限模型
type RolePermission struct {
	ID          int64    `db:"id" json:"id"`
	GroupID     int64    `db:"group_id" json:"group_id"`
	RoleName    string   `db:"role_name" json:"role_name"`
	Permissions []string `db:"permissions" json:"permissions"`
	CreatedAt   string   `db:"created_at" json:"created_at"`
}

// GrantUserPermission 授予用户权限
func (l *PermissionLogic) GrantUserPermission(groupID int64, userID string, permissions []string) (*UserPermission, error) {
	query := `
		INSERT INTO service_group_users (group_id, user_id, permissions)
		VALUES ($1, $2, $3)
		ON CONFLICT (group_id, user_id) 
		DO UPDATE SET permissions = $3
		RETURNING id, group_id, user_id, permissions, created_at
	`
	
	var perm UserPermission
	err := l.db.QueryRowCtx(l.ctx, &perm, query, groupID, userID, pq.Array(permissions))
	
	if err != nil {
		l.logger.Errorf("授予用户权限失败: %v", err)
		return nil, fmt.Errorf("授予用户权限失败: %w", err)
	}
	
	l.logger.Infof("用户权限授予成功: group=%d, user=%s", groupID, userID)
	return &perm, nil
}

// RevokeUserPermission 撤销用户权限
func (l *PermissionLogic) RevokeUserPermission(groupID int64, userID string) error {
	query := `DELETE FROM service_group_users WHERE group_id = $1 AND user_id = $2`
	
	result, err := l.db.ExecCtx(l.ctx, query, groupID, userID)
	if err != nil {
		l.logger.Errorf("撤销用户权限失败: %v", err)
		return fmt.Errorf("撤销用户权限失败: %w", err)
	}
	
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("用户权限不存在")
	}
	
	l.logger.Infof("用户权限撤销成功: group=%d, user=%s", groupID, userID)
	return nil
}

// GetUserPermission 获取用户权限
func (l *PermissionLogic) GetUserPermission(groupID int64, userID string) (*UserPermission, error) {
	query := `
		SELECT id, group_id, user_id, permissions, created_at
		FROM service_group_users
		WHERE group_id = $1 AND user_id = $2
	`
	
	var perm UserPermission
	err := l.db.QueryRowCtx(l.ctx, &perm, query, groupID, userID)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("用户权限不存在")
		}
		l.logger.Errorf("获取用户权限失败: %v", err)
		return nil, fmt.Errorf("获取用户权限失败: %w", err)
	}
	
	return &perm, nil
}

// ListUserPermissions 查询服务组的用户权限列表
func (l *PermissionLogic) ListUserPermissions(groupID int64) ([]*UserPermission, error) {
	query := `
		SELECT id, group_id, user_id, permissions, created_at
		FROM service_group_users
		WHERE group_id = $1
		ORDER BY created_at DESC
	`
	
	var perms []*UserPermission
	err := l.db.QueryRowsCtx(l.ctx, &perms, query, groupID)
	if err != nil {
		l.logger.Errorf("查询用户权限列表失败: %v", err)
		return nil, fmt.Errorf("查询用户权限列表失败: %w", err)
	}
	
	return perms, nil
}

// GrantRolePermission 授予角色权限
func (l *PermissionLogic) GrantRolePermission(groupID int64, roleName string, permissions []string) (*RolePermission, error) {
	query := `
		INSERT INTO service_group_roles (group_id, role_name, permissions)
		VALUES ($1, $2, $3)
		ON CONFLICT (group_id, role_name) 
		DO UPDATE SET permissions = $3
		RETURNING id, group_id, role_name, permissions, created_at
	`
	
	var perm RolePermission
	err := l.db.QueryRowCtx(l.ctx, &perm, query, groupID, roleName, pq.Array(permissions))
	
	if err != nil {
		l.logger.Errorf("授予角色权限失败: %v", err)
		return nil, fmt.Errorf("授予角色权限失败: %w", err)
	}
	
	l.logger.Infof("角色权限授予成功: group=%d, role=%s", groupID, roleName)
	return &perm, nil
}

// RevokeRolePermission 撤销角色权限
func (l *PermissionLogic) RevokeRolePermission(groupID int64, roleName string) error {
	query := `DELETE FROM service_group_roles WHERE group_id = $1 AND role_name = $2`
	
	result, err := l.db.ExecCtx(l.ctx, query, groupID, roleName)
	if err != nil {
		l.logger.Errorf("撤销角色权限失败: %v", err)
		return fmt.Errorf("撤销角色权限失败: %w", err)
	}
	
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("角色权限不存在")
	}
	
	l.logger.Infof("角色权限撤销成功: group=%d, role=%s", groupID, roleName)
	return nil
}

// GetRolePermission 获取角色权限
func (l *PermissionLogic) GetRolePermission(groupID int64, roleName string) (*RolePermission, error) {
	query := `
		SELECT id, group_id, role_name, permissions, created_at
		FROM service_group_roles
		WHERE group_id = $1 AND role_name = $2
	`
	
	var perm RolePermission
	err := l.db.QueryRowCtx(l.ctx, &perm, query, groupID, roleName)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("角色权限不存在")
		}
		l.logger.Errorf("获取角色权限失败: %v", err)
		return nil, fmt.Errorf("获取角色权限失败: %w", err)
	}
	
	return &perm, nil
}

// ListRolePermissions 查询服务组的角色权限列表
func (l *PermissionLogic) ListRolePermissions(groupID int64) ([]*RolePermission, error) {
	query := `
		SELECT id, group_id, role_name, permissions, created_at
		FROM service_group_roles
		WHERE group_id = $1
		ORDER BY created_at DESC
	`
	
	var perms []*RolePermission
	err := l.db.QueryRowsCtx(l.ctx, &perms, query, groupID)
	if err != nil {
		l.logger.Errorf("查询角色权限列表失败: %v", err)
		return nil, fmt.Errorf("查询角色权限列表失败: %w", err)
	}
	
	return perms, nil
}

// CheckUserPermission 检查用户是否有指定权限
func (l *PermissionLogic) CheckUserPermission(groupID int64, userID string, requiredPermission string) (bool, error) {
	// 获取用户权限
	userPerm, err := l.GetUserPermission(groupID, userID)
	if err == nil {
		// 检查是否有该权限
		for _, perm := range userPerm.Permissions {
			if perm == "*" || perm == requiredPermission {
				return true, nil
			}
		}
	}
	
	// TODO: 检查用户角色的权限（需要从 Casdoor 获取用户角色）
	
	return false, nil
}
