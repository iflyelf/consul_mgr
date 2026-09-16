package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/iflyelf/consul_mgr/internal/logic/instance"
	"github.com/iflyelf/consul_mgr/internal/pkg/consul"
	"github.com/iflyelf/consul_mgr/internal/pkg/parallel"
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

		names := make([]string, 0, len(services))
		for name := range services {
			names = append(names, name)
		}
		sort.Strings(names)

		// 并行按服务取实例：复用实例模块的按服务缓存，
		// 用户先看过实例列表时此处可直接命中缓存，无需再次请求 Consul。
		perService := parallel.Map(names, l.svcCtx.ConsulConcurrency(), func(name string) []types.ConsulInstanceInfo {
			return instance.ListInstancesCached(l.ctx, l.svcCtx, client, groupID, name)
		})
		for i, list := range perService {
			if len(list) == 0 {
				continue
			}
			all = append(all, buildServiceInfoFromInstances(names[i], list))
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

// buildServiceInfoFromInstances 由已获取的实例列表构建服务汇总信息
func buildServiceInfoFromInstances(serviceName string, list []types.ConsulInstanceInfo) types.ConsulServiceInfo {
	healthyCount := 0
	unhealthyCount := 0
	healthStatus := "passing"

	for _, it := range list {
		switch it.HealthStatus {
		case "passing":
			healthyCount++
		case "critical":
			unhealthyCount++
			healthStatus = "critical"
		default:
			unhealthyCount++
			if healthStatus != "critical" {
				healthStatus = "warning"
			}
		}
	}

	first := list[0]
	return types.ConsulServiceInfo{
		ID:             first.ID,
		Service:        serviceName,
		Tags:           first.Tags,
		Meta:           first.Meta,
		Address:        first.Address,
		Port:           first.Port,
		HealthStatus:   healthStatus,
		InstanceCount:  len(list),
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

	// 并行删除所有实例（支持集群跨节点注销）
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.Service.ID)
	}
	if errs := parallel.ForEach(ids, l.svcCtx.ConsulConcurrency(), func(_ int, id string) error {
		return consul.DeregisterService(client, id)
	}); len(errs) > 0 {
		invalidateServiceCache(l.ctx, l.svcCtx, groupID)
		return fmt.Errorf("删除服务 %s 时 %d 个实例失败: %v", serviceName, len(errs), errs)
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

	if len(serviceNames) == 0 {
		return errors.New("未选择任何服务")
	}

	concurrency := l.svcCtx.ConsulConcurrency()

	// 第一步：并行收集各服务下的实例 ID（每个服务一次查询）
	type svcEntries struct {
		name string
		ids  []string
		err  error
	}
	perSvc := parallel.Map(serviceNames, concurrency, func(name string) svcEntries {
		entries, _, e := client.Health().Service(name, "", false, nil)
		if e != nil {
			return svcEntries{name: name, err: e}
		}
		ids := make([]string, 0, len(entries))
		for _, entry := range entries {
			ids = append(ids, entry.Service.ID)
		}
		return svcEntries{name: name, ids: ids}
	})

	// 第二步：汇总所有实例，统一并行注销（单一并发池，避免嵌套并发导致 goroutine 爆炸）
	var errs []string
	type target struct{ service, id string }
	var targets []target
	for _, s := range perSvc {
		if s.err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", s.name, s.err))
			continue
		}
		for _, id := range s.ids {
			targets = append(targets, target{s.name, id})
		}
	}

	if derrs := parallel.ForEach(targets, concurrency, func(_ int, t target) error {
		return consul.DeregisterService(client, t.id)
	}); len(derrs) > 0 {
		for i, de := range derrs {
			errs = append(errs, fmt.Sprintf("%s/%s: %v", targets[i].service, targets[i].id, de))
		}
	}

	invalidateServiceCache(l.ctx, l.svcCtx, groupID)

	if len(errs) > 0 {
		sort.Strings(errs)
		return errors.New("批量删除部分失败: " + fmt.Sprint(errs))
	}
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
		client, _, err := getConsulClientWithAddr(l.ctx, l.svcCtx, groupID)
		if err != nil {
			return nil, err
		}

		// 复用实例模块的「按服务缓存」：与实例列表/服务列表共享同一份数据，
		// 避免同一服务被重复请求 Consul。
		instances := instance.ListInstancesCached(l.ctx, l.svcCtx, client, groupID, serviceName)
		if len(instances) == 0 {
			return nil, errors.New("服务不存在")
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
