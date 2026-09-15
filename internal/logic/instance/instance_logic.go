package instance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hashicorp/consul/api"
	"github.com/iflyelf/consul_mgr/internal/svc"
	"github.com/iflyelf/consul_mgr/internal/types"
	"gopkg.in/yaml.v3"
)

// getConsulClient 获取 Consul 客户端的辅助函数
func getConsulClient(ctx context.Context, svcCtx *svc.ServiceContext, groupID int64) (*api.Client, error) {
	client, err := svcCtx.GetConsulClient(ctx, groupID)
	if err != nil {
		return nil, err
	}
	return client.GetAPIClient(), nil
}
type ListInstancesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListInstancesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListInstancesLogic {
	return &ListInstancesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListInstancesLogic) ListInstances(groupID int64, serviceName, status string) ([]types.ConsulInstanceInfo, error) {
	client, err := getConsulClient(l.ctx, l.svcCtx, groupID)
	if err != nil {
		return nil, err
	}

	entries, _, err := client.Health().Service(serviceName, "", false, nil)
	if err != nil {
		return nil, fmt.Errorf("查询实例失败: %w", err)
	}

	var instances []types.ConsulInstanceInfo
	for _, entry := range entries {
		service := entry.Service
		node := entry.Node
		checks := entry.Checks
		healthStatus := aggregateStatus(checks)

		// 状态过滤
		if status != "" && healthStatus != status {
			continue
		}

		instances = append(instances, types.ConsulInstanceInfo{
			ID:           service.ID,
			Service:      service.Service,
			Tags:         service.Tags,
			Meta:         service.Meta,
			Address:      service.Address,
			Port:         service.Port,
			Node:         node.Node,
			NodeAddress:  node.Address,
			HealthStatus: healthStatus,
			Checks:       convertChecks(checks),
		})
	}

	return instances, nil
}

// RegisterInstanceLogic 注册实例逻辑
type RegisterInstanceLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRegisterInstanceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterInstanceLogic {
	return &RegisterInstanceLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RegisterInstanceLogic) RegisterInstance(groupID int64, req *types.RegisterInstanceRequest) error {
	client, err := getConsulClient(l.ctx, l.svcCtx, groupID)
	if err != nil {
		return err
	}

	// 构建服务注册信息
	registration := &api.AgentServiceRegistration{
		ID:      req.ID,
		Name:    req.Service,
		Tags:    req.Tags,
		Meta:    req.Meta,
		Address: req.Address,
		Port:    req.Port,
	}

	// 添加健康检查
	if req.Check != nil {
		check := &api.AgentServiceCheck{
			Interval: req.Check.Interval,
			Timeout:  req.Check.Timeout,
		}
		
		switch req.Check.Type {
		case "http":
			check.HTTP = req.Check.HTTP
		case "tcp":
			check.TCP = req.Check.TCP
		case "ttl":
			check.TTL = req.Check.TTL
		case "grpc":
			check.GRPC = req.Check.GRPC
			check.GRPCUseTLS = req.Check.GRPCUseTLS
		}
		
		registration.Check = check
	}

	// 注册服务
	if err := client.Agent().ServiceRegister(registration); err != nil {
		return fmt.Errorf("注册实例失败: %w", err)
	}

	return nil
}

// UpdateInstanceLogic 更新实例逻辑
type UpdateInstanceLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateInstanceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateInstanceLogic {
	return &UpdateInstanceLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateInstanceLogic) UpdateInstance(groupID int64, instanceID string, req *types.UpdateInstanceRequest) error {
	client, err := getConsulClient(l.ctx, l.svcCtx, groupID)
	if err != nil {
		return err
	}

	// 先获取现有实例信息
	service, _, err := client.Agent().Service(instanceID, nil)
	if err != nil {
		return fmt.Errorf("获取实例失败: %w", err)
	}

	// 更新字段
	if req.Tags != nil {
		service.Tags = req.Tags
	}
	if req.Meta != nil {
		service.Meta = req.Meta
	}
	if req.Address != "" {
		service.Address = req.Address
	}
	if req.Port > 0 {
		service.Port = req.Port
	}

	// 重新注册（Consul 的更新方式）
	registration := &api.AgentServiceRegistration{
		ID:      service.ID,
		Name:    service.Service,
		Tags:    service.Tags,
		Meta:    service.Meta,
		Address: service.Address,
		Port:    service.Port,
	}

	if err := client.Agent().ServiceRegister(registration); err != nil {
		return fmt.Errorf("更新实例失败: %w", err)
	}

	return nil
}

// DeleteInstanceLogic 删除实例逻辑
type DeleteInstanceLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteInstanceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteInstanceLogic {
	return &DeleteInstanceLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteInstanceLogic) DeleteInstance(groupID int64, instanceID string) error {
	client, err := getConsulClient(l.ctx, l.svcCtx, groupID)
	if err != nil {
		return err
	}

	if err := client.Agent().ServiceDeregister(instanceID); err != nil {
		return fmt.Errorf("删除实例失败: %w", err)
	}

	return nil
}

// BatchDeleteInstancesLogic 批量删除实例逻辑
type BatchDeleteInstancesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewBatchDeleteInstancesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchDeleteInstancesLogic {
	return &BatchDeleteInstancesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *BatchDeleteInstancesLogic) BatchDeleteInstances(groupID int64, instanceIDs []string) error {
	client, err := getConsulClient(l.ctx, l.svcCtx, groupID)
	if err != nil {
		return err
	}

	var errs []string
	for _, instanceID := range instanceIDs {
		if err := client.Agent().ServiceDeregister(instanceID); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", instanceID, err))
		}
	}

	if len(errs) > 0 {
		return errors.New("批量删除部分失败: " + fmt.Sprint(errs))
	}

	return nil
}

// ExportInstancesLogic 导出实例逻辑
type ExportInstancesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewExportInstancesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ExportInstancesLogic {
	return &ExportInstancesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ExportInstancesLogic) ExportInstances(groupID int64, serviceName, format string) ([]byte, error) {
	client, err := getConsulClient(l.ctx, l.svcCtx, groupID)
	if err != nil {
		return nil, err
	}

	entries, _, err := client.Health().Service(serviceName, "", false, nil)
	if err != nil {
		return nil, fmt.Errorf("查询实例失败: %w", err)
	}

	var instances []types.RegisterInstanceRequest
	for _, entry := range entries {
		service := entry.Service
		instances = append(instances, types.RegisterInstanceRequest{
			GroupID: groupID,
			ID:      service.ID,
			Service: service.Service,
			Tags:    service.Tags,
			Meta:    service.Meta,
			Address: service.Address,
			Port:    service.Port,
		})
	}

	switch format {
	case "json":
		return json.MarshalIndent(instances, "", "  ")
	case "yaml":
		return yaml.Marshal(instances)
	case "csv":
		return exportToCSV(instances)
	default:
		return nil, errors.New("不支持的格式")
	}
}

// ImportInstancesLogic 导入实例逻辑
type ImportInstancesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewImportInstancesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ImportInstancesLogic {
	return &ImportInstancesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ImportInstancesLogic) ImportInstances(groupID int64, format string, data []byte) error {
	client, err := getConsulClient(l.ctx, l.svcCtx, groupID)
	if err != nil {
		return err
	}

	var instances []types.RegisterInstanceRequest

	switch format {
	case "json":
		if err := json.Unmarshal(data, &instances); err != nil {
			return fmt.Errorf("解析 JSON 失败: %w", err)
		}
	case "yaml":
		if err := yaml.Unmarshal(data, &instances); err != nil {
			return fmt.Errorf("解析 YAML 失败: %w", err)
		}
	case "csv":
		var err error
		instances, err = importFromCSV(data)
		if err != nil {
			return fmt.Errorf("解析 CSV 失败: %w", err)
		}
	default:
		return errors.New("不支持的格式")
	}

	// 批量注册
	var errs []string
	for _, instance := range instances {
		registration := &api.AgentServiceRegistration{
			ID:      instance.ID,
			Name:    instance.Service,
			Tags:    instance.Tags,
			Meta:    instance.Meta,
			Address: instance.Address,
			Port:    instance.Port,
		}

		if err := client.Agent().ServiceRegister(registration); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", instance.ID, err))
		}
	}

	if len(errs) > 0 {
		return errors.New("批量导入部分失败: " + fmt.Sprint(errs))
	}

	return nil
}

// 辅助函数
func aggregateStatus(checks []*api.HealthCheck) string {
	status := "passing"
	for _, check := range checks {
		if check.Status == api.HealthCritical {
			return "critical"
		}
		if check.Status == api.HealthWarning {
			status = "warning"
		}
	}
	return status
}

func convertChecks(checks []*api.HealthCheck) []map[string]interface{} {
	var result []map[string]interface{}
	for _, check := range checks {
		result = append(result, map[string]interface{}{
			"check_id": check.CheckID,
			"name":     check.Name,
			"status":   check.Status,
			"notes":    check.Notes,
			"output":   check.Output,
		})
	}
	return result
}

func exportToCSV(instances []types.RegisterInstanceRequest) ([]byte, error) {
	// 实现 CSV 导出
	return []byte("id,service,address,port,tags,meta\n"), nil
}

func importFromCSV(data []byte) ([]types.RegisterInstanceRequest, error) {
	// 实现 CSV 导入
	return nil, errors.New("CSV 导入暂未实现")
}
