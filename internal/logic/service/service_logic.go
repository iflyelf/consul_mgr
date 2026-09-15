package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/iflyelf/consul_mgr/internal/pkg/consul"
	"github.com/iflyelf/consul_mgr/internal/svc"
	"github.com/iflyelf/consul_mgr/internal/types"
)

// getConsulClient 获取 Consul 客户端的辅助函数
func getConsulClient(ctx context.Context, svcCtx *svc.ServiceContext, groupID int64) (*api.Client, error) {
	client, _, err := getConsulClientWithAddr(ctx, svcCtx, groupID)
	return client, err
}

// getConsulClientWithAddr 获取 Consul 客户端及其地址
func getConsulClientWithAddr(ctx context.Context, svcCtx *svc.ServiceContext, groupID int64) (*api.Client, string, error) {
	client, err := svcCtx.GetConsulClient(ctx, groupID)
	if err != nil {
		return nil, "", err
	}
	return client.GetAPIClient(), client.Address(), nil
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
	// 缓存键（服务列表按服务组缓存，关键词在内存中过滤）
	cacheKey := l.svcCtx.CacheKeyServices(groupID)
	var all []types.ConsulServiceInfo
	if !l.svcCtx.Cache.Get(l.ctx, cacheKey, &all) {
		client, addr, err := getConsulClientWithAddr(l.ctx, l.svcCtx, groupID)
		if err != nil {
			return nil, err
		}

		// 查询所有服务
		services, _, err := client.Catalog().Services(nil)
		if err != nil {
			return nil, fmt.Errorf("查询服务列表失败: %w", consul.FriendlyError(addr, err))
		}

		for serviceName := range services {
			// 查询服务的所有实例
			entries, _, err := client.Health().Service(serviceName, "", false, nil)
			if err != nil {
				continue
			}
			if len(entries) == 0 {
				continue
			}

			all = append(all, buildServiceInfo(serviceName, entries))
		}

		l.svcCtx.Cache.Set(l.ctx, cacheKey, all)
	}

	// 关键词过滤（缓存的是全量，过滤在内存完成）
	if keyword == "" {
		return all, nil
	}
	result := make([]types.ConsulServiceInfo, 0, len(all))
	for _, s := range all {
		if contains(s.Service, keyword) {
			result = append(result, s)
		}
	}
	return result, nil
}

// buildServiceInfo 由 Consul 健康条目构建服务汇总信息
func buildServiceInfo(serviceName string, entries []*api.ServiceEntry) types.ConsulServiceInfo {
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

	service := entries[0].Service
	return types.ConsulServiceInfo{
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
	}
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

	invalidateServiceCache(l.ctx, l.svcCtx, groupID)
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
		invalidateServiceCache(l.ctx, l.svcCtx, groupID)
		return errors.New("批量删除部分失败: " + fmt.Sprint(errs))
	}

	invalidateServiceCache(l.ctx, l.svcCtx, groupID)
	return nil
}

// invalidateServiceCache 失效指定服务组的服务列表与服务详情缓存
func invalidateServiceCache(ctx context.Context, svcCtx *svc.ServiceContext, groupID int64) {
	svcCtx.InvalidateServices(ctx, groupID)
	svcCtx.InvalidateServiceDetails(ctx, groupID)
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

// GetServiceDetail 获取服务详情
//
// 参数:
//   groupID     - 服务组 ID
//   serviceName - 服务名
//   keyword     - 关键字（过滤实例：实例ID/地址/节点/Tags/Meta）
func (l *GetServiceDetailLogic) GetServiceDetail(groupID int64, serviceName, keyword string) (*types.ServiceDetail, error) {
	// 缓存键（详情按服务缓存，关键字在内存过滤）
	cacheKey := l.svcCtx.CacheKeyServiceDetail(groupID, serviceName)
	var detail types.ServiceDetail
	if !l.svcCtx.Cache.Get(l.ctx, cacheKey, &detail) {
		client, addr, err := getConsulClientWithAddr(l.ctx, l.svcCtx, groupID)
		if err != nil {
			return nil, err
		}

		entries, _, err := client.Health().Service(serviceName, "", false, nil)
		if err != nil {
			return nil, fmt.Errorf("查询服务失败: %w", consul.FriendlyError(addr, err))
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

		detail = types.ServiceDetail{
			ServiceName: serviceName,
			Instances:   instances,
		}
		l.svcCtx.Cache.Set(l.ctx, cacheKey, detail)
	}

	// 关键字过滤（实例ID / 地址 / 节点 / Tags / Meta）
	if keyword == "" {
		return &detail, nil
	}

	kw := strings.ToLower(keyword)
	filtered := make([]types.ConsulInstanceInfo, 0, len(detail.Instances))
	for _, ins := range detail.Instances {
		if matchInstanceKeyword(ins, kw) {
			filtered = append(filtered, ins)
		}
	}
	return &types.ServiceDetail{
		ServiceName: detail.ServiceName,
		Instances:   filtered,
	}, nil
}

// matchInstanceKeyword 判断实例是否匹配关键字
func matchInstanceKeyword(ins types.ConsulInstanceInfo, kw string) bool {
	if strings.Contains(strings.ToLower(ins.ID), kw) ||
		strings.Contains(strings.ToLower(ins.Address), kw) ||
		strings.Contains(strings.ToLower(ins.Node), kw) ||
		strings.Contains(strings.ToLower(ins.NodeAddress), kw) {
		return true
	}
	for _, t := range ins.Tags {
		if strings.Contains(strings.ToLower(t), kw) {
			return true
		}
	}
	for k, v := range ins.Meta {
		if strings.Contains(strings.ToLower(k), kw) || strings.Contains(strings.ToLower(v), kw) {
			return true
		}
	}
	return false
}

// 辅助函数
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
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
