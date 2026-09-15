package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/consul/api"
	"github.com/iflyelf/consul_mgr/internal/svc"
	"github.com/iflyelf/consul_mgr/internal/types"
)

// getConsulClient 获取 Consul 客户端的辅助函数
func getConsulClient(ctx context.Context, svcCtx *svc.ServiceContext, groupID int64) (*api.Client, error) {
	client, err := svcCtx.GetConsulClient(ctx, groupID)
	if err != nil {
		return nil, err
	}
	return client.GetAPIClient(), nil
}

// ListServicesLogic 服务列表逻辑
type ListServicesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListServicesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListServicesLogic {
	return &ListServicesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListServicesLogic) ListServices(groupID int64, keyword string) ([]types.ConsulServiceInfo, error) {
	client, err := getConsulClient(l.ctx, l.svcCtx, groupID)
	if err != nil {
		return nil, err
	}

	// 查询所有服务
	services, _, err := client.Catalog().Services(nil)
	if err != nil {
		return nil, fmt.Errorf("查询服务列表失败: %w", err)
	}

	var result []types.ConsulServiceInfo
	for serviceName := range services {
		// 如果有关键词过滤
		if keyword != "" && !contains(serviceName, keyword) {
			continue
		}

		// 查询服务的所有实例
		entries, _, err := client.Health().Service(serviceName, "", false, nil)
		if err != nil {
			continue
		}

		if len(entries) == 0 {
			continue
		}

		// 统计健康状态
		healthyCount := 0
		unhealthyCount := 0
		healthStatus := "passing"

		for _, entry := range entries {
			status := aggregateStatus(entry.Checks)
			if status == "passing" {
				healthyCount++
			} else {
				unhealthyCount++
				if status == "critical" {
					healthStatus = "critical"
				} else if status == "warning" && healthStatus != "critical" {
					healthStatus = "warning"
				}
			}
		}

		// 获取第一个实例的信息作为代表
		firstEntry := entries[0]
		service := firstEntry.Service

		result = append(result, types.ConsulServiceInfo{
			ID:             service.ID,
			Service:        serviceName,
			Tags:           service.Tags,
			Meta:           service.Meta,
			Address:        service.Address,
			Port:           service.Port,
			HealthStatus:   healthStatus,
			InstanceCount:  len(entries),
			HealthyCount:   healthyCount,
			UnhealthyCount: unhealthyCount,
		})
	}

	return result, nil
}

// DeleteServiceLogic 删除服务逻辑
type DeleteServiceLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteServiceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteServiceLogic {
	return &DeleteServiceLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteServiceLogic) DeleteService(groupID int64, serviceName string) error {
	client, err := getConsulClient(l.ctx, l.svcCtx, groupID)
	if err != nil {
		return err
	}

	// 获取服务的所有实例
	entries, _, err := client.Health().Service(serviceName, "", false, nil)
	if err != nil {
		return fmt.Errorf("查询服务实例失败: %w", err)
	}

	// 删除所有实例
	for _, entry := range entries {
		if err := client.Agent().ServiceDeregister(entry.Service.ID); err != nil {
			return fmt.Errorf("删除实例 %s 失败: %w", entry.Service.ID, err)
		}
	}

	return nil
}

// BatchDeleteServicesLogic 批量删除服务逻辑
type BatchDeleteServicesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewBatchDeleteServicesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchDeleteServicesLogic {
	return &BatchDeleteServicesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *BatchDeleteServicesLogic) BatchDeleteServices(groupID int64, serviceNames []string) error {
	client, err := getConsulClient(l.ctx, l.svcCtx, groupID)
	if err != nil {
		return err
	}

	var errs []string
	for _, serviceName := range serviceNames {
		entries, _, err := client.Health().Service(serviceName, "", false, nil)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", serviceName, err))
			continue
		}

		for _, entry := range entries {
			if err := client.Agent().ServiceDeregister(entry.Service.ID); err != nil {
				errs = append(errs, fmt.Sprintf("%s/%s: %v", serviceName, entry.Service.ID, err))
			}
		}
	}

	if len(errs) > 0 {
		return errors.New("批量删除部分失败: " + fmt.Sprint(errs))
	}

	return nil
}

// GetServiceDetailLogic 获取服务详情逻辑
type GetServiceDetailLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetServiceDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetServiceDetailLogic {
	return &GetServiceDetailLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetServiceDetailLogic) GetServiceDetail(groupID int64, serviceName string) (*types.ServiceDetail, error) {
	client, err := getConsulClient(l.ctx, l.svcCtx, groupID)
	if err != nil {
		return nil, err
	}

	entries, _, err := client.Health().Service(serviceName, "", false, nil)
	if err != nil {
		return nil, fmt.Errorf("查询服务失败: %w", err)
	}

	if len(entries) == 0 {
		return nil, errors.New("服务不存在")
	}

	var instances []types.ConsulInstanceInfo
	for _, entry := range entries {
		service := entry.Service
		node := entry.Node
		checks := entry.Checks

		instances = append(instances, types.ConsulInstanceInfo{
			ID:           service.ID,
			Service:      service.Service,
			Tags:         service.Tags,
			Meta:         service.Meta,
			Address:      service.Address,
			Port:         service.Port,
			Node:         node.Node,
			NodeAddress:  node.Address,
			HealthStatus: aggregateStatus(checks),
			Checks:       convertChecks(checks),
		})
	}

	return &types.ServiceDetail{
		ServiceName: serviceName,
		Instances:   instances,
	}, nil
}

// 辅助函数
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0)
}

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
