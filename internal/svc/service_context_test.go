package svc

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/iflyelf/consul_mgr/internal/config"
	"github.com/iflyelf/consul_mgr/internal/pkg/casdoor"
)

// newTestClient 构造一个无需网络的 Casdoor 客户端替身（NewClient 仅做字段校验）。
func newTestClient(t *testing.T, endpoint string) *casdoor.Client {
	t.Helper()
	c, err := casdoor.NewClient(&casdoor.Config{
		Endpoint:         endpoint,
		ClientId:         "id",
		ClientSecret:     "secret",
		OrganizationName: "org",
		ApplicationName:  "app",
	})
	if err != nil {
		t.Fatalf("构造测试客户端失败: %v", err)
	}
	return c
}

// TestReloadCasdoor_ConcurrentSwap 验证热重载期间并发读取不会拿到 nil，
// 且每次 ReloadCasdoor 都会原子替换为新客户端（配合 -race 检测数据竞争）。
func TestReloadCasdoor_ConcurrentSwap(t *testing.T) {
	svcCtx := &ServiceContext{cfgStore: config.NewStore(&config.Config{})}
	svcCtx.casdoorRef.Store(newTestClient(t, "http://init"))

	var seq int64
	svcCtx.casdoorBuild = func(config.Config) (*casdoor.Client, error) {
		i := atomic.AddInt64(&seq, 1)
		return newTestClient(t, fmt.Sprintf("http://casdoor-%d", i)), nil
	}

	const readers = 8
	stop := make(chan struct{})
	var readWG sync.WaitGroup
	for i := 0; i < readers; i++ {
		readWG.Add(1)
		go func() {
			defer readWG.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				if c := svcCtx.Casdoor(); c == nil {
					t.Error("并发读取期间 Casdoor() 返回 nil")
					return
				}
			}
		}()
	}

	var writeWG sync.WaitGroup
	writeWG.Add(1)
	go func() {
		defer writeWG.Done()
		for i := 0; i < 300; i++ {
			if err := svcCtx.ReloadCasdoor(context.Background()); err != nil {
				t.Errorf("热重载失败: %v", err)
				return
			}
		}
	}()

	writeWG.Wait()
	close(stop)
	readWG.Wait()

	if got := atomic.LoadInt64(&seq); got != 300 {
		t.Errorf("构建次数 = %d，期望 300", got)
	}
	if svcCtx.Casdoor() == nil {
		t.Fatal("全部重载后 Casdoor() 仍为 nil")
	}
}

// TestReloadCasdoor_ErrorKeepsOldClient 验证重建失败时保留旧客户端。
func TestReloadCasdoor_ErrorKeepsOldClient(t *testing.T) {
	svcCtx := &ServiceContext{cfgStore: config.NewStore(&config.Config{})}
	old := newTestClient(t, "http://old")
	svcCtx.casdoorRef.Store(old)
	svcCtx.casdoorBuild = func(config.Config) (*casdoor.Client, error) {
		return nil, errors.New("端点不可达")
	}

	if err := svcCtx.ReloadCasdoor(context.Background()); err == nil {
		t.Fatal("期望重建失败返回错误")
	}
	if svcCtx.Casdoor() != old {
		t.Fatal("重建失败后旧客户端被替换")
	}
}
