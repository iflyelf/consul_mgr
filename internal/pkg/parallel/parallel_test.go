package parallel

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestForEachCollectsErrors(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}
	errs := ForEach(items, 2, func(i int, v int) error {
		if v%2 == 0 {
			return errors.New("even")
		}
		return nil
	})
	// 2、4 失败
	if len(errs) != 2 {
		t.Fatalf("want 2 errors, got %d", len(errs))
	}
	if _, ok := errs[1]; !ok {
		t.Errorf("index 1 should fail")
	}
	if _, ok := errs[3]; !ok {
		t.Errorf("index 3 should fail")
	}
}

func TestForEachLimitConcurrency(t *testing.T) {
	var cur, max int32
	// 记录并发峰值
	ForEachIndex(50, 4, func(i int) error {
		c := atomic.AddInt32(&cur, 1)
		for {
			m := atomic.LoadInt32(&max)
			if c <= m || atomic.CompareAndSwapInt32(&max, m, c) {
				break
			}
		}
		time.Sleep(5 * time.Millisecond)
		atomic.AddInt32(&cur, -1)
		return nil
	})
	if max > 4 {
		t.Errorf("concurrency exceeded limit: %d > 4", max)
	}
}

func TestMapPreservesOrder(t *testing.T) {
	in := []int{10, 20, 30}
	out := Map(in, 3, func(v int) int { return v * 2 })
	want := []int{20, 40, 60}
	for i := range want {
		if out[i] != want[i] {
			t.Fatalf("index %d: got %d want %d", i, out[i], want[i])
		}
	}
}

// TestForEachRecoversPanic 验证单项 panic 被捕获为该项错误，且不影响其它项、
// 不终止进程。
func TestForEachRecoversPanic(t *testing.T) {
	items := []int{0, 1, 2, 3, 4}
	errs := ForEach(items, 3, func(i int, v int) error {
		if v == 2 {
			panic("boom")
		}
		return nil
	})
	if len(errs) != 1 {
		t.Fatalf("want 1 error (panic), got %d: %v", len(errs), errs)
	}
	if _, ok := errs[2]; !ok {
		t.Fatalf("panic 项（index 2）应被记录为错误: %v", errs)
	}
}
