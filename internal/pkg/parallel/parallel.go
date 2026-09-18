// Package parallel 提供带并发上限的并行执行工具
//
// 说明：
//
//	Consul 批量操作（查询/删除）若串行执行，网络往返会线性累积导致极慢。
//	本包统一提供「有界并发」执行原语，避免各处重复实现，并防止瞬间打爆下游。
//
// 作者: iflyelf
package parallel

import (
	"sync"
)

// DefaultConcurrency 默认并发度
const DefaultConcurrency = 16

// ForEach 以固定并发度并行遍历 items
//
// 参数:
//
//	items       - 待处理元素
//	concurrency - 并发上限（<=0 时取 DefaultConcurrency）
//	fn          - 处理函数，返回 error 表示该项失败
//
// 返回:
//
//	map[int]error - 失败项的下标及错误（全部成功时为空）
//
// 说明:
//   - 任何一项失败都不会中断其他项；
//   - 结果通过返回值汇总，调用方按需处理。
func ForEach[T any](items []T, concurrency int, fn func(index int, item T) error) map[int]error {
	return ForEachIndex(len(items), concurrency, func(i int) error {
		return fn(i, items[i])
	})
}

// ForEachIndex 以固定并发度并行遍历 [0, n)
//
// 适用于结果不便于直接索引（或需要按下标写入切片）的场景。
func ForEachIndex(n, concurrency int, fn func(index int) error) map[int]error {
	if n <= 0 {
		return nil
	}
	if concurrency <= 0 {
		concurrency = DefaultConcurrency
	}
	if concurrency > n {
		concurrency = n
	}

	var (
		mu   sync.Mutex
		wg   sync.WaitGroup
		errs = make(map[int]error)
	)
	sem := make(chan struct{}, concurrency)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := fn(idx); err != nil {
				mu.Lock()
				errs[idx] = err
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()

	if len(errs) == 0 {
		return nil
	}
	return errs
}

// Map 并行对 items 做映射，返回与输入等长的结果切片
//
// 说明:
//   - 保序：out[i] 对应 items[i]；
//   - 任一处理函数 panic 会向上传播（由调用方保证 fn 不 panic）。
func Map[T any, R any](items []T, concurrency int, fn func(item T) R) []R {
	out := make([]R, len(items))
	if len(items) == 0 {
		return out
	}
	_ = ForEachIndex(len(items), concurrency, func(i int) error {
		out[i] = fn(items[i])
		return nil
	})
	return out
}
