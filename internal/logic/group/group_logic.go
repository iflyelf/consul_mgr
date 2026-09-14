package group

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/iflyelf/consul_mgr/internal/middleware"
	"github.com/iflyelf/consul_mgr/internal/model"
	"github.com/iflyelf/consul_mgr/internal/pkg/consul"
	"github.com/iflyelf/consul_mgr/internal/svc"
	"github.com/iflyelf/consul_mgr/internal/types"
)

// ListGroupsLogic 服务组列表逻辑
type ListGroupsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListGroupsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListGroupsLogic {
	return &ListGroupsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListGroupsLogic) ListGroups(req *types.QueryRequest) (*types.PageResponse, error) {
	// 构建查询条件
	query := `SELECT id, name, code, description, consul_address, consul_datacenter, status, created_at 
	          FROM service_groups WHERE 1=1`
	args := []interface{}{}
	argPos := 1
	
	if req.Keyword != "" {
		query += ` AND (name LIKE $` + string(rune(argPos)) + ` OR code LIKE $` + string(rune(argPos)) + `)`
		args = append(args, "%"+req.Keyword+"%")
		argPos++
	}
	
	if req.Status != nil {
		query += ` AND status = $` + string(rune(argPos))
		args = append(args, *req.Status)
		argPos++
	}
	
	query += ` ORDER BY created_at DESC`
	
	// 查询列表
	var groups []model.ServiceGroup
	err := l.svcCtx.DB.QueryRowsPartialCtx(l.ctx, &groups, query, args...)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	
	return &types.PageResponse{
		Total:    int64(len(groups)),
		Page:     req.Page,
		PageSize: req.PageSize,
		List:     groups,
	}, nil
}

// CreateGroupLogic 创建服务组逻辑
type CreateGroupLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateGroupLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateGroupLogic {
	return &CreateGroupLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateGroupLogic) CreateGroup(req *types.CreateGroupRequest) (*model.ServiceGroup, error) {
	userID := middleware.GetUserID(l.ctx)
	
	// 检查 code 是否已存在
	var count int
	checkQuery := `SELECT COUNT(*) FROM service_groups WHERE code = $1`
	err := l.svcCtx.DB.QueryRowCtx(l.ctx, &count, checkQuery, req.Code)
	if err == nil && count > 0 {
		return nil, errors.New("服务组代码已存在")
	}
	
	// 设置默认值
	datacenter := req.ConsulDatacenter
	if datacenter == "" {
		datacenter = "dc1"
	}
	
	// 插入服务组
	query := `INSERT INTO service_groups 
	          (name, code, description, consul_address, consul_token, consul_datacenter, status, created_by, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, 1, $7, $8, $8)
	          RETURNING id, name, code, description, consul_address, consul_datacenter, status, created_at`
	
	var group model.ServiceGroup
	now := time.Now()
	err = l.svcCtx.DB.QueryRowPartialCtx(l.ctx, &group, query,
		req.Name, req.Code, req.Description, req.ConsulAddress, req.ConsulToken, datacenter, userID, now)
	if err != nil {
		return nil, err
	}
	
	return &group, nil
}

// GetGroupLogic 获取服务组详情逻辑
type GetGroupLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetGroupLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetGroupLogic {
	return &GetGroupLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetGroupLogic) GetGroup(id int64) (*model.ServiceGroup, error) {
	var group model.ServiceGroup
	query := `SELECT id, name, code, description, consul_address, consul_token, consul_datacenter, status, created_at, updated_at
	          FROM service_groups WHERE id = $1`
	err := l.svcCtx.DB.QueryRowPartialCtx(l.ctx, &group, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("服务组不存在")
		}
		return nil, err
	}
	
	return &group, nil
}

// UpdateGroupLogic 更新服务组逻辑
type UpdateGroupLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateGroupLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateGroupLogic {
	return &UpdateGroupLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateGroupLogic) UpdateGroup(id int64, req *types.UpdateGroupRequest) error {
	// 检查服务组是否存在
	var count int
	checkQuery := `SELECT COUNT(*) FROM service_groups WHERE id = $1`
	err := l.svcCtx.DB.QueryRowCtx(l.ctx, &count, checkQuery, id)
	if err != nil || count == 0 {
		return errors.New("服务组不存在")
	}
	
	// 构建更新语句
	query := `UPDATE service_groups SET updated_at = $1`
	args := []interface{}{time.Now()}
	argPos := 2
	
	if req.Name != "" {
		query += `, name = $` + strconv.Itoa(argPos)
		args = append(args, req.Name)
		argPos++
	}
	
	if req.Description != "" {
		query += `, description = $` + strconv.Itoa(argPos)
		args = append(args, req.Description)
		argPos++
	}
	
	if req.ConsulAddress != "" {
		query += `, consul_address = $` + strconv.Itoa(argPos)
		args = append(args, req.ConsulAddress)
		argPos++
	}
	
	if req.ConsulToken != "" {
		query += `, consul_token = $` + strconv.Itoa(argPos)
		args = append(args, req.ConsulToken)
		argPos++
	}
	
	if req.ConsulDatacenter != "" {
		query += `, consul_datacenter = $` + strconv.Itoa(argPos)
		args = append(args, req.ConsulDatacenter)
		argPos++
	}
	
	if req.Status != nil {
		query += `, status = $` + strconv.Itoa(argPos)
		args = append(args, *req.Status)
		argPos++
	}
	
	query += ` WHERE id = $` + strconv.Itoa(argPos)
	args = append(args, id)
	
	_, err = l.svcCtx.DB.ExecCtx(l.ctx, query, args...)
	if err != nil {
		return err
	}
	
	// 清除缓存的 Consul 客户端
	l.svcCtx.ConsulManager.RemoveClient(id)
	
	return nil
}

// DeleteGroupLogic 删除服务组逻辑
type DeleteGroupLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteGroupLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteGroupLogic {
	return &DeleteGroupLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteGroupLogic) DeleteGroup(id int64) error {
	query := `DELETE FROM service_groups WHERE id = $1`
	result, err := l.svcCtx.DB.ExecCtx(l.ctx, query, id)
	if err != nil {
		return err
	}
	
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.New("服务组不存在")
	}
	
	// 清除缓存的 Consul 客户端
	l.svcCtx.ConsulManager.RemoveClient(id)
	
	return nil
}

// TestConnectionLogic 测试连接逻辑
type TestConnectionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewTestConnectionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TestConnectionLogic {
	return &TestConnectionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *TestConnectionLogic) TestConnection(id int64) error {
	// 获取服务组信息
	var group model.ServiceGroup
	query := `SELECT consul_address, consul_token, consul_datacenter FROM service_groups WHERE id = $1`
	err := l.svcCtx.DB.QueryRowPartialCtx(l.ctx, &group, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("服务组不存在")
		}
		return err
	}
	
	// 测试连接
	cfg := &consul.Config{
		Address:    group.ConsulAddress,
		Token:      group.ConsulToken,
		Datacenter: group.ConsulDatacenter,
	}
	
	return l.svcCtx.ConsulManager.TestConnection(id, cfg)
}
