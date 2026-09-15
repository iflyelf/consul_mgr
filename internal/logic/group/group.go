// Package group 提供服务组管理逻辑
package group

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"github.com/iflyelf/consul_mgr/internal/pkg/consul"
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

// Group 服务组模型（对应 service_groups 表）
type Group struct {
	ID               int64  `db:"id" json:"id"`
	Name             string `db:"name" json:"name"`
	Code             string `db:"code" json:"code"`
	Description      string `db:"description" json:"description"`
	ConsulAddress    string `db:"consul_address" json:"consul_address"`
	ConsulToken      string `db:"consul_token" json:"consul_token,omitempty"`
	ConsulDatacenter string `db:"consul_datacenter" json:"consul_datacenter"`
	Status           int    `db:"status" json:"status"`
	CreatedAt        string `db:"created_at" json:"created_at"`
	UpdatedAt        string `db:"updated_at" json:"updated_at"`
}

// CreateGroup 创建服务组
func (l *GroupLogic) CreateGroup(name, code, consulAddress, consulToken, datacenter, description string) (*Group, error) {
	// 若未指定 code，则自动使用 name
	if code == "" {
		code = name
	}
	if datacenter == "" {
		datacenter = "dc1"
	}

	query := `
		INSERT INTO service_groups (name, code, consul_address, consul_token, consul_datacenter, description, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, 1, NOW(), NOW())
		RETURNING id, name, code, consul_address, consul_token, consul_datacenter, description, status,
		          TO_CHAR(created_at, 'YYYY-MM-DD HH24:MI:SS'),
		          TO_CHAR(updated_at, 'YYYY-MM-DD HH24:MI:SS')
	`

	var group Group
	err := l.db.QueryRowCtx(l.ctx, &group, query,
		name, code, consulAddress, consulToken, datacenter, description)

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
		    consul_datacenter = $4, description = $5, updated_at = NOW()
		WHERE id = $6
		RETURNING id, name, code, consul_address, consul_token, consul_datacenter, description, status,
		          TO_CHAR(created_at, 'YYYY-MM-DD HH24:MI:SS'),
		          TO_CHAR(updated_at, 'YYYY-MM-DD HH24:MI:SS')
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
		SELECT id, name, code, consul_address, consul_token, consul_datacenter, description, status,
		       TO_CHAR(created_at, 'YYYY-MM-DD HH24:MI:SS'),
		       TO_CHAR(updated_at, 'YYYY-MM-DD HH24:MI:SS')
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
		whereClause += fmt.Sprintf(" AND (name LIKE $%d OR code LIKE $%d OR description LIKE $%d)", argIdx, argIdx, argIdx)
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
		SELECT id, name, code, consul_address, consul_token, consul_datacenter, description, status,
		       TO_CHAR(created_at, 'YYYY-MM-DD HH24:MI:SS'),
		       TO_CHAR(updated_at, 'YYYY-MM-DD HH24:MI:SS')
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

// TestConnection 测试服务组的 Consul 连接
func (l *GroupLogic) TestConnection(id int64, manager *consul.Manager) error {
	group, err := l.GetGroup(id)
	if err != nil {
		return err
	}

	cfg := &consul.Config{
		Address:    group.ConsulAddress,
		Token:      group.ConsulToken,
		Datacenter: group.ConsulDatacenter,
	}

	if err := manager.TestConnection(id, cfg); err != nil {
		l.logger.Errorf("Consul 连接测试失败: %v", err)
		return fmt.Errorf("Consul 连接测试失败: %w", err)
	}

	l.logger.Infof("Consul 连接测试成功: %s", group.ConsulAddress)
	return nil
}

// DetectDatacenter 探测指定 Consul 地址的数据中心与节点名称
//
// 参数:
//   address - Consul 地址
//   token   - Consul Token（可选）
//
// 返回:
//   string - 数据中心
//   string - 节点名称
//   error  - 错误信息
func (l *GroupLogic) DetectDatacenter(address, token string) (string, string, error) {
	if address == "" {
		return "", "", fmt.Errorf("Consul 地址不能为空")
	}

	client, err := consul.NewClient(&consul.Config{
		Address: address,
		Token:   token,
		Timeout: 5 * time.Second,
	})
	if err != nil {
		return "", "", fmt.Errorf("创建 Consul 客户端失败: %w", err)
	}

	// 内网偶发抖动时重试（快速失败，避免用户长时间等待）
	var dc, nodeName string
	var lastErr error
	for i := 0; i < 2; i++ {
		dc, nodeName, lastErr = client.DetectDatacenter()
		if lastErr == nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if lastErr != nil {
		l.logger.Errorf("探测数据中心失败: %v", lastErr)
		return "", "", consul.FriendlyError(address, lastErr)
	}

	l.logger.Infof("数据中心探测成功: %s (节点: %s)", dc, nodeName)
	return dc, nodeName, nil
}
