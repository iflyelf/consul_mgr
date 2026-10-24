// Package instance 的主机导出：面向服务间调用（如 upgrade_mgr）的只读视图。
//
// 与实例列表的区别：
//   - 实例是「服务 + 端口」粒度，同一台机器会出现多条；
//   - 主机是「IP」粒度，把同一 IP 上的多个服务实例聚合成一条。
//
// 数据实时来自各服务组配置的 Consul Catalog，不落本地表，
// 保证与 Consul 状态一致（与 ListInstances 的数据来源相同）。
package instance

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/iflyelf/consul_mgr/internal/svc"
	"github.com/iflyelf/consul_mgr/internal/types"
)

// ExportedHost 一台主机的聚合视图（按 IP 聚合）。
//
// Meta 为该 IP 上所有服务实例 ServiceMeta 的并集：同名键以首个非空值为准，
// 调用方（upgrade_mgr）按自己的映射规则决定写入哪个字段，这里不做任何键的取舍。
type ExportedHost struct {
	IP         string            `json:"ip"`
	Node       string            `json:"node"`
	Datacenter string            `json:"datacenter"`
	GroupID    int64             `json:"group_id"`
	GroupName  string            `json:"group_name"`
	GroupCode  string            `json:"group_code"`
	Services   []string          `json:"services"`
	Tags       []string          `json:"tags"`
	Meta       map[string]string `json:"meta"`
	// Health 该 IP 上所有实例的最差健康状态：critical > warning > passing。
	Health string `json:"health"`
}

// ExportHostsResult 导出结果。
type ExportHostsResult struct {
	Hosts []ExportedHost `json:"hosts"`
	// Groups 实际参与导出的服务组数；FailedGroups 为连接失败的组。
	Groups       int      `json:"groups"`
	FailedGroups []string `json:"failed_groups,omitempty"`
	// Partial 为 true 表示部分服务组查询失败，清单不完整。
	// 调用方据此跳过「清理不存在主机」等破坏性操作。
	Partial bool `json:"partial"`
}

// healthRank 健康状态的严重程度，用于取最差值。
var healthRank = map[string]int{"passing": 0, "warning": 1, "critical": 2}

// ExportHosts 聚合指定服务组（groupID<=0 表示全部启用的组）的主机清单。
func ExportHosts(ctx context.Context, svcCtx *svc.ServiceContext, groupID int64) (*ExportHostsResult, error) {
	groups, err := listEnabledGroups(ctx, svcCtx, groupID)
	if err != nil {
		return nil, err
	}

	res := &ExportHostsResult{Hosts: []ExportedHost{}}
	// byIP 跨服务组聚合：同一 IP 只保留首个服务组的归属信息，
	// 但服务名/标签/Meta 仍然合并，避免多组共用机器时丢数据。
	byIP := map[string]*ExportedHost{}

	for _, g := range groups {
		insts, err := listGroupInstances(ctx, svcCtx, g.ID)
		if err != nil {
			// 单个组失败不影响其它组，但标记为不完整。
			res.FailedGroups = append(res.FailedGroups, g.Name)
			res.Partial = true
			continue
		}
		res.Groups++
		for _, in := range insts {
			ip := strings.TrimSpace(in.Address)
			if ip == "" {
				// ServiceAddress 为空时回退到节点地址（与 Consul 解析规则一致）。
				ip = strings.TrimSpace(in.NodeAddress)
			}
			if ip == "" {
				continue
			}
			h := byIP[ip]
			if h == nil {
				h = &ExportedHost{
					IP: ip, GroupID: g.ID, GroupName: g.Name, GroupCode: g.Code,
					// 服务组配置的数据中心作为兜底：Catalog 健康条目里不带该字段。
					Datacenter: g.Datacenter,
					Meta:       map[string]string{}, Health: "passing",
				}
				byIP[ip] = h
			}
			mergeInstance(h, in)
		}
	}

	for _, h := range byIP {
		sort.Strings(h.Services)
		sort.Strings(h.Tags)
		res.Hosts = append(res.Hosts, *h)
	}
	// 固定顺序，便于调用方比对与分页。
	sort.Slice(res.Hosts, func(i, j int) bool { return res.Hosts[i].IP < res.Hosts[j].IP })
	return res, nil
}

// mergeInstance 把一个服务实例合并进主机聚合视图。
func mergeInstance(h *ExportedHost, in types.ConsulInstanceInfo) {
	if h.Node == "" {
		h.Node = in.Node
	}
	if h.Datacenter == "" {
		h.Datacenter = in.Datacenter
	}
	if in.Service != "" && !containsStr(h.Services, in.Service) {
		h.Services = append(h.Services, in.Service)
	}
	for _, t := range in.Tags {
		if t != "" && !containsStr(h.Tags, t) {
			h.Tags = append(h.Tags, t)
		}
	}
	// Meta 并集：首个非空值优先，不覆盖已有值。
	for k, v := range in.Meta {
		k = strings.TrimSpace(k)
		if k == "" || strings.TrimSpace(v) == "" {
			continue
		}
		if _, ok := h.Meta[k]; !ok {
			h.Meta[k] = v
		}
	}
	if healthRank[in.HealthStatus] > healthRank[h.Health] {
		h.Health = in.HealthStatus
	}
}

func containsStr(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// exportGroup 导出所需的服务组信息。
type exportGroup struct {
	ID         int64  `db:"id"`
	Name       string `db:"name"`
	Code       string `db:"code"`
	Datacenter string `db:"consul_datacenter"`
}

// listEnabledGroups 列出启用中的服务组；groupID>0 时只取该组。
//
// 只返回已配置 Consul 地址的启用组（status=1），避免把半配置的组算作失败。
func listEnabledGroups(ctx context.Context, svcCtx *svc.ServiceContext, groupID int64) ([]exportGroup, error) {
	query := `
		SELECT id, name, code, COALESCE(consul_datacenter,'') AS consul_datacenter
		FROM service_groups
		WHERE status = 1 AND COALESCE(consul_address,'') <> ''`
	args := []any{}
	if groupID > 0 {
		query += ` AND id = $1`
		args = append(args, groupID)
	}
	query += ` ORDER BY id`

	var groups []exportGroup
	if err := svcCtx.DB.QueryRowsCtx(ctx, &groups, query, args...); err != nil {
		return nil, fmt.Errorf("查询服务组失败: %w", err)
	}
	return groups, nil
}

// listGroupInstances 取某服务组下全部服务的实例（复用实例列表的缓存路径）。
func listGroupInstances(ctx context.Context, svcCtx *svc.ServiceContext, groupID int64) ([]types.ConsulInstanceInfo, error) {
	l := NewListInstancesLogic(ctx, svcCtx)
	// status 传空：导出不做健康过滤，健康状态作为字段返回给调用方。
	return l.ListInstances(groupID, "", "")
}
