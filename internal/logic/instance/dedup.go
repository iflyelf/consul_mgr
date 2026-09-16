package instance

import (
	"fmt"
	"sync"

	"github.com/hashicorp/consul/api"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/iflyelf/consul_mgr/internal/pkg/consul"
	"github.com/iflyelf/consul_mgr/internal/pkg/parallel"
	"github.com/iflyelf/consul_mgr/internal/types"
)

// localNodeName 获取本地 Consul 节点名
//
// 说明：客户端通过 Agent 注册，服务归属该 Agent 所在节点。
func localNodeName(client *api.Client) string {
	self, err := client.Agent().Self()
	if err != nil || self == nil {
		return ""
	}
	cfg, ok := self["Config"]
	if !ok {
		return ""
	}
	if v, ok := cfg["NodeName"].(string); ok {
		return v
	}
	return ""
}

// dedupClusterRegistrations 注册前的集群级去重
//
// 背景:
//   Consul 集群中，Agent 注册是「节点本地」的；若同一 ServiceID 通过不同 Agent
//   注册，会在 Catalog 中出现多条记录（不同 Node），即同一实例被多个节点重复注册。
//
// 方案:
//   注册前扫描集群 Catalog（而非仅本地 Agent），发现目标 ServiceID 已存在于
//   「其他节点」时，先通过 Catalog 注销远端记录，保证同一实例在集群中只归属一个节点。
//
// 参数:
//   client  - Consul 客户端
//   targets - 即将注册的实例列表
//
// 返回:
//   []string - 被清理的远端重复项（格式 serviceID@node）
//   error    - 查询/注销失败
func dedupClusterRegistrations(client *api.Client, targets []types.RegisterInstanceRequest) ([]string, error) {
	if len(targets) == 0 {
		return nil, nil
	}

	localNode := localNodeName(client)

	wanted := make(map[string]bool) // 目标 ServiceID
	names := make(map[string]bool)  // 涉及的服务名
	for i := range targets {
		id := targets[i].ID
		if id == "" {
			id = targets[i].Service
		}
		wanted[id] = true
		if targets[i].Service != "" {
			names[targets[i].Service] = true
		}
	}

	nameList := make([]string, 0, len(names))
	for name := range names {
		nameList = append(nameList, name)
	}

	// 第一步：并行查询各服务的集群条目，汇总待清理的远端重复项
	type found struct{ node, id string }
	perSvc := parallel.Map(nameList, parallel.DefaultConcurrency, func(name string) []found {
		entries, _, err := client.Catalog().Service(name, "", nil)
		if err != nil {
			// 查询失败不阻断注册，仅记录
			logx.Errorf("[dedup] 查询集群服务失败 service=%s: %v", name, err)
			return nil
		}
		var local []found
		for _, e := range entries {
			if !wanted[e.ServiceID] {
				continue
			}
			// 本节点的交由注册逻辑处理（覆盖/跳过）
			if localNode != "" && e.Node == localNode {
				continue
			}
			local = append(local, found{e.Node, e.ServiceID})
		}
		return local
	})

	var dups []found
	for _, list := range perSvc {
		dups = append(dups, list...)
	}

	// 第二步：统一并行清理（单一并发池，避免嵌套并发）
	var mu sync.Mutex
	var removed []string
	_ = parallel.ForEach(dups, parallel.DefaultConcurrency, func(_ int, f found) error {
		if derr := consul.DeregisterRemote(client, f.node, f.id); derr != nil {
			logx.Errorf("[dedup] 清理远端重复失败 %s@%s: %v", f.id, f.node, derr)
			return nil
		}
		mu.Lock()
		removed = append(removed, fmt.Sprintf("%s@%s", f.id, f.node))
		mu.Unlock()
		return nil
	})

	if len(removed) > 0 {
		logx.Infof("[dedup] 已清理集群中 %d 个重复实例: %v", len(removed), removed)
	}
	return removed, nil
}

// dedupSingle 单实例注册去重（便捷封装）
func dedupSingle(client *api.Client, req *types.RegisterInstanceRequest) {
	_, _ = dedupClusterRegistrations(client, []types.RegisterInstanceRequest{*req})
}
