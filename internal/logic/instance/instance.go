// Package instance 提供实例管理逻辑
package instance

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// InstanceLogic 实例管理逻辑
type InstanceLogic struct {
	ctx    context.Context
	db     sqlx.SqlConn
	logger logx.Logger
}

// NewInstanceLogic 创建实例管理逻辑实例
func NewInstanceLogic(ctx context.Context, db sqlx.SqlConn) *InstanceLogic {
	return &InstanceLogic{
		ctx:    ctx,
		db:     db,
		logger: logx.WithContext(ctx),
	}
}

// Instance 实例模型
type Instance struct {
	ID          int64                  `db:"id" json:"id"`
	InstanceID  string                 `db:"instance_id" json:"instance_id"`
	ServiceName string                 `db:"service_name" json:"service_name"`
	GroupID     int64                  `db:"group_id" json:"group_id"`
	Address     string                 `db:"address" json:"address"`
	Port        int                    `db:"port" json:"port"`
	Tags        []string               `db:"tags" json:"tags"`
	Meta        map[string]interface{} `db:"meta" json:"meta"`
	HealthCheck map[string]interface{} `db:"health_check" json:"health_check,omitempty"`
	Status      string                 `db:"status" json:"status"`
	Datacenter  string                 `db:"datacenter" json:"datacenter"`
	NodeName    string                 `db:"node_name" json:"node_name"`
	CreatedBy   string                 `db:"created_by" json:"created_by"`
	CreatedAt   time.Time              `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time              `db:"updated_at" json:"updated_at"`
}

// RegisterInstance 注册实例
func (l *InstanceLogic) RegisterInstance(
	instanceID, serviceName string,
	groupID int64,
	address string,
	port int,
	tags []string,
	meta, healthCheck map[string]interface{},
	datacenter, nodeName, createdBy string,
) (*Instance, error) {
	
	// 序列化 Meta 和 HealthCheck
	metaJSON, _ := json.Marshal(meta)
	healthCheckJSON, _ := json.Marshal(healthCheck)
	
	query := `
		INSERT INTO consul_instances 
		(instance_id, service_name, group_id, address, port, tags, meta, health_check, 
		 datacenter, node_name, created_by, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 'passing')
		RETURNING id, instance_id, service_name, group_id, address, port, tags, meta, 
		          health_check, status, datacenter, node_name, created_by, created_at, updated_at
	`
	
	var instance Instance
	err := l.db.QueryRowCtx(l.ctx, &instance, query,
		instanceID, serviceName, groupID, address, port, pq.Array(tags),
		metaJSON, healthCheckJSON, datacenter, nodeName, createdBy)
	
	if err != nil {
		l.logger.Errorf("注册实例失败: %v", err)
		return nil, fmt.Errorf("注册实例失败: %w", err)
	}
	
	l.logger.Infof("实例注册成功: %s (ID: %d)", instanceID, instance.ID)
	return &instance, nil
}

// UpdateInstance 更新实例
func (l *InstanceLogic) UpdateInstance(
	id int64,
	address string,
	port int,
	tags []string,
	meta, healthCheck map[string]interface{},
	status, datacenter, nodeName string,
) (*Instance, error) {
	
	metaJSON, _ := json.Marshal(meta)
	healthCheckJSON, _ := json.Marshal(healthCheck)
	
	query := `
		UPDATE consul_instances 
		SET address = $1, port = $2, tags = $3, meta = $4, health_check = $5,
		    status = $6, datacenter = $7, node_name = $8, updated_at = NOW()
		WHERE id = $9
		RETURNING id, instance_id, service_name, group_id, address, port, tags, meta,
		          health_check, status, datacenter, node_name, created_by, created_at, updated_at
	`
	
	var instance Instance
	err := l.db.QueryRowCtx(l.ctx, &instance, query,
		address, port, pq.Array(tags), metaJSON, healthCheckJSON,
		status, datacenter, nodeName, id)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("实例不存在: ID=%d", id)
		}
		l.logger.Errorf("更新实例失败: %v", err)
		return nil, fmt.Errorf("更新实例失败: %w", err)
	}
	
	l.logger.Infof("实例更新成功: %s (ID: %d)", instance.InstanceID, instance.ID)
	return &instance, nil
}

// DeregisterInstance 注销实例
func (l *InstanceLogic) DeregisterInstance(id int64) error {
	query := `DELETE FROM consul_instances WHERE id = $1`
	
	result, err := l.db.ExecCtx(l.ctx, query, id)
	if err != nil {
		l.logger.Errorf("注销实例失败: %v", err)
		return fmt.Errorf("注销实例失败: %w", err)
	}
	
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("实例不存在: ID=%d", id)
	}
	
	l.logger.Infof("实例注销成功: ID=%d", id)
	return nil
}

// GetInstance 获取实例详情
func (l *InstanceLogic) GetInstance(id int64) (*Instance, error) {
	query := `
		SELECT id, instance_id, service_name, group_id, address, port, tags, meta,
		       health_check, status, datacenter, node_name, created_by, created_at, updated_at
		FROM consul_instances
		WHERE id = $1
	`
	
	var instance Instance
	err := l.db.QueryRowCtx(l.ctx, &instance, query, id)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("实例不存在: ID=%d", id)
		}
		l.logger.Errorf("获取实例失败: %v", err)
		return nil, fmt.Errorf("获取实例失败: %w", err)
	}
	
	return &instance, nil
}

// ListInstances 查询实例列表
func (l *InstanceLogic) ListInstances(
	groupID int64,
	serviceName, status, datacenter, keyword string,
	page, pageSize int,
) ([]*Instance, int64, error) {
	
	// 构建查询条件
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1
	
	if groupID > 0 {
		whereClause += fmt.Sprintf(" AND group_id = $%d", argIdx)
		args = append(args, groupID)
		argIdx++
	}
	
	if serviceName != "" {
		whereClause += fmt.Sprintf(" AND service_name = $%d", argIdx)
		args = append(args, serviceName)
		argIdx++
	}
	
	if status != "" {
		whereClause += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	
	if datacenter != "" {
		whereClause += fmt.Sprintf(" AND datacenter = $%d", argIdx)
		args = append(args, datacenter)
		argIdx++
	}
	
	if keyword != "" {
		whereClause += fmt.Sprintf(" AND (instance_id LIKE $%d OR address LIKE $%d OR node_name LIKE $%d)", argIdx, argIdx, argIdx)
		args = append(args, "%"+keyword+"%")
		argIdx++
	}
	
	// 查询总数
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM consul_instances %s", whereClause)
	var total int64
	err := l.db.QueryRowCtx(l.ctx, &total, countQuery, args...)
	if err != nil {
		l.logger.Errorf("查询实例总数失败: %v", err)
		return nil, 0, fmt.Errorf("查询实例总数失败: %w", err)
	}
	
	// 查询列表
	offset := (page - 1) * pageSize
	listQuery := fmt.Sprintf(`
		SELECT id, instance_id, service_name, group_id, address, port, tags, meta,
		       health_check, status, datacenter, node_name, created_by, created_at, updated_at
		FROM consul_instances
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)
	
	args = append(args, pageSize, offset)
	
	var instances []*Instance
	err = l.db.QueryRowsCtx(l.ctx, &instances, listQuery, args...)
	if err != nil {
		l.logger.Errorf("查询实例列表失败: %v", err)
		return nil, 0, fmt.Errorf("查询实例列表失败: %w", err)
	}
	
	return instances, total, nil
}

// BatchDelete 批量删除实例
func (l *InstanceLogic) BatchDelete(ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, fmt.Errorf("实例 ID 列表为空")
	}
	
	query := `DELETE FROM consul_instances WHERE id = ANY($1)`
	
	result, err := l.db.ExecCtx(l.ctx, query, pq.Array(ids))
	if err != nil {
		l.logger.Errorf("批量删除实例失败: %v", err)
		return 0, fmt.Errorf("批量删除实例失败: %w", err)
	}
	
	rows, _ := result.RowsAffected()
	l.logger.Infof("批量删除实例成功: %d 条", rows)
	return rows, nil
}

// SyncFromConsul 从 Consul 同步实例（预留接口）
func (l *InstanceLogic) SyncFromConsul(groupID int64, consulAddress, consulToken string) error {
	// TODO: 实现从 Consul 同步实例的逻辑
	// 1. 连接到 Consul
	// 2. 获取所有服务和实例
	// 3. 与数据库对比，更新或插入
	l.logger.Info("从 Consul 同步实例（功能待实现）")
	return nil
}

// GetDatacenters 获取所有数据中心列表
func (l *InstanceLogic) GetDatacenters(groupID int64) ([]string, error) {
	query := `
		SELECT DISTINCT datacenter 
		FROM consul_instances 
		WHERE group_id = $1 
		ORDER BY datacenter
	`
	
	var datacenters []string
	err := l.db.QueryRowsCtx(l.ctx, &datacenters, query, groupID)
	if err != nil {
		l.logger.Errorf("获取数据中心列表失败: %v", err)
		return nil, fmt.Errorf("获取数据中心列表失败: %w", err)
	}
	
	return datacenters, nil
}

// GetServices 获取服务列表
func (l *InstanceLogic) GetServices(groupID int64, datacenter string) ([]string, error) {
	query := `
		SELECT DISTINCT service_name 
		FROM consul_instances 
		WHERE group_id = $1
	`
	args := []interface{}{groupID}
	
	if datacenter != "" {
		query += " AND datacenter = $2"
		args = append(args, datacenter)
	}
	
	query += " ORDER BY service_name"
	
	var services []string
	err := l.db.QueryRowsCtx(l.ctx, &services, query, args...)
	if err != nil {
		l.logger.Errorf("获取服务列表失败: %v", err)
		return nil, fmt.Errorf("获取服务列表失败: %w", err)
	}
	
	return services, nil
}
