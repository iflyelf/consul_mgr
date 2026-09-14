// Package group 提供服务组管理逻辑
package group

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// GroupLogic 服务组管理逻辑
type GroupLogic struct {
	ctx    context.Context
	db     sqlx.SqlConn
	logger logx.Logger
}

// NewGroupLogic 创建服务组管理逻辑实例
func NewGroupLogic(ctx context.Context, db sqlx.SqlConn) *GroupLogic {
	return &GroupLogic{
		ctx:    ctx,
		db:     db,
		logger: logx.WithContext(ctx),
	}
}

// Group 服务组模型
type Group struct {
	ID            int64  `db:"id" json:"id"`
	Name          string `db:"name" json:"name"`
	ConsulAddress string `db:"consul_address" json:"consul_address"`
	ConsulToken   string `db:"consul_token" json:"consul_token,omitempty"`
	Datacenter    string `db:"datacenter" json:"datacenter"`
	Description   string `db:"description" json:"description"`
	CreatedAt     string `db:"created_at" json:"created_at"`
	UpdatedAt     string `db:"updated_at" json:"updated_at"`
}

// CreateGroup 创建服务组
func (l *GroupLogic) CreateGroup(name, consulAddress, consulToken, datacenter, description string) (*Group, error) {
	query := `
		INSERT INTO service_groups (name, consul_address, consul_token, datacenter, description)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, consul_address, consul_token, datacenter, description, 
		          created_at, updated_at
	`
	
	var group Group
	err := l.db.QueryRowCtx(l.ctx, &group, query, 
		name, consulAddress, consulToken, datacenter, description)
	
	if err != nil {
		l.logger.Errorf("创建服务组失败: %v", err)
		return nil, fmt.Errorf("创建服务组失败: %w", err)
	}
	
	l.logger.Infof("服务组创建成功: %s (ID: %d)", group.Name, group.ID)
	return &group, nil
}

// UpdateGroup 更新服务组
func (l *GroupLogic) UpdateGroup(id int64, name, consulAddress, consulToken, datacenter, description string) (*Group, error) {
	query := `
		UPDATE service_groups 
		SET name = $1, consul_address = $2, consul_token = $3, 
		    datacenter = $4, description = $5, updated_at = NOW()
		WHERE id = $6
		RETURNING id, name, consul_address, consul_token, datacenter, description, 
		          created_at, updated_at
	`
	
	var group Group
	err := l.db.QueryRowCtx(l.ctx, &group, query,
		name, consulAddress, consulToken, datacenter, description, id)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("服务组不存在: ID=%d", id)
		}
		l.logger.Errorf("更新服务组失败: %v", err)
		return nil, fmt.Errorf("更新服务组失败: %w", err)
	}
	
	l.logger.Infof("服务组更新成功: %s (ID: %d)", group.Name, group.ID)
	return &group, nil
}

// DeleteGroup 删除服务组
func (l *GroupLogic) DeleteGroup(id int64) error {
	query := `DELETE FROM service_groups WHERE id = $1`
	
	result, err := l.db.ExecCtx(l.ctx, query, id)
	if err != nil {
		l.logger.Errorf("删除服务组失败: %v", err)
		return fmt.Errorf("删除服务组失败: %w", err)
	}
	
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("服务组不存在: ID=%d", id)
	}
	
	l.logger.Infof("服务组删除成功: ID=%d", id)
	return nil
}

// GetGroup 获取服务组详情
func (l *GroupLogic) GetGroup(id int64) (*Group, error) {
	query := `
		SELECT id, name, consul_address, consul_token, datacenter, description, 
		       created_at, updated_at
		FROM service_groups
		WHERE id = $1
	`
	
	var group Group
	err := l.db.QueryRowCtx(l.ctx, &group, query, id)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("服务组不存在: ID=%d", id)
		}
		l.logger.Errorf("获取服务组失败: %v", err)
		return nil, fmt.Errorf("获取服务组失败: %w", err)
	}
	
	return &group, nil
}

// ListGroups 查询服务组列表
func (l *GroupLogic) ListGroups(keyword string, page, pageSize int) ([]*Group, int64, error) {
	// 构建查询条件
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1
	
	if keyword != "" {
		whereClause += fmt.Sprintf(" AND (name LIKE $%d OR description LIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+keyword+"%")
		argIdx++
	}
	
	// 查询总数
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM service_groups %s", whereClause)
	var total int64
	err := l.db.QueryRowCtx(l.ctx, &total, countQuery, args...)
	if err != nil {
		l.logger.Errorf("查询服务组总数失败: %v", err)
		return nil, 0, fmt.Errorf("查询服务组总数失败: %w", err)
	}
	
	// 查询列表
	offset := (page - 1) * pageSize
	listQuery := fmt.Sprintf(`
		SELECT id, name, consul_address, consul_token, datacenter, description, 
		       created_at, updated_at
		FROM service_groups
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)
	
	args = append(args, pageSize, offset)
	
	var groups []*Group
	err = l.db.QueryRowsCtx(l.ctx, &groups, listQuery, args...)
	if err != nil {
		l.logger.Errorf("查询服务组列表失败: %v", err)
		return nil, 0, fmt.Errorf("查询服务组列表失败: %w", err)
	}
	
	return groups, total, nil
}

// GroupDetailStats 服务组详情统计
type GroupDetailStats struct {
	ServiceCount  int `json:"service_count"`
	InstanceCount int `json:"instance_count"`
	UserCount     int `json:"user_count"`
	RoleCount     int `json:"role_count"`
}

// GetGroupDetail 获取服务组详情（包含统计）
func (l *GroupLogic) GetGroupDetail(id int64) (*Group, *GroupDetailStats, error) {
	// 获取基本信息
	group, err := l.GetGroup(id)
	if err != nil {
		return nil, nil, err
	}
	
	// 统计实例数量
	var stats GroupDetailStats
	
	instanceQuery := `SELECT COUNT(*) FROM consul_instances WHERE group_id = $1`
	l.db.QueryRowCtx(l.ctx, &stats.InstanceCount, instanceQuery, id)
	
	// 统计服务数量（去重）
	serviceQuery := `SELECT COUNT(DISTINCT service_name) FROM consul_instances WHERE group_id = $1`
	l.db.QueryRowCtx(l.ctx, &stats.ServiceCount, serviceQuery, id)
	
	// 统计授权用户数量
	userQuery := `SELECT COUNT(*) FROM service_group_users WHERE group_id = $1`
	l.db.QueryRowCtx(l.ctx, &stats.UserCount, userQuery, id)
	
	// 统计授权角色数量
	roleQuery := `SELECT COUNT(*) FROM service_group_roles WHERE group_id = $1`
	l.db.QueryRowCtx(l.ctx, &stats.RoleCount, roleQuery, id)
	
	return group, &stats, nil
}
