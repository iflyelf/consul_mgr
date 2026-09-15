package consul

import (
	"sync"

	"github.com/hashicorp/consul/api"
	"github.com/zeromicro/go-zero/core/logx"
)

// DefaultQueryConcurrency Consul 并发查询默认上限
const DefaultQueryConcurrency = 16

// FetchServiceHealth 并行拉取多个服务的健康条目
//
// 背景:
//   服务数量多时，串行逐个调用 Health().Service 会因网络往返累积而极慢（N+1）。
//   例如 300 个服务、单次往返 50ms，串行需约 15s，并发后降至约 1s。
//
// 说明:
//   - 使用固定大小 worker pool 限制并发，避免瞬间打爆 Consul；
//   - 单个服务查询失败只记录日志并跳过，不阻断整体列表（生产环境更健壮）。
//
// 返回:
//   map[string][]*api.ServiceEntry - 服务名 -> 健康条目（查询失败的服务不在结果中）
func FetchServiceHealth(client *api.Client, serviceNames []string) map[string][]*api.ServiceEntry {
	return FetchServiceHealthWithConcurrency(client, serviceNames, DefaultQueryConcurrency)
}

// FetchServiceHealthWithConcurrency 可自定义并发度的并行拉取
func FetchServiceHealthWithConcurrency(client *api.Client, serviceNames []string, concurrency int) map[string][]*api.ServiceEntry {
	result := make(map[string][]*api.ServiceEntry, len(serviceNames))
	if len(serviceNames) == 0 {
		return result
	}
	if concurrency <= 0 {
		concurrency = DefaultQueryConcurrency
	}

	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)
	sem := make(chan struct{}, concurrency)

	for _, name := range serviceNames {
		wg.Add(1)
		go func(n string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			entries, _, err := client.Health().Service(n, "", false, nil)
			if err != nil {
				logx.Errorf("[consul] 查询服务 %s 实例失败，已跳过: %v", n, err)
				return
			}
			mu.Lock()
			result[n] = entries
			mu.Unlock()
		}(name)
	}
	wg.Wait()
	return result
}
