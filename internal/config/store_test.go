package config

import (
	"reflect"
	"sync"
	"testing"

	"github.com/zeromicro/go-zero/core/conf"
)

// TestStoreConcurrentReadWrite 验证快照存储的并发读写下无数据竞争（配合 -race）。
func TestStoreConcurrentReadWrite(t *testing.T) {
	c := &Config{}
	c.Admin.Username = "admin"
	c.Permission.DefaultPermissions = []string{"read"}
	c.Security.CORSAllowedOrigins = []string{"https://a.example.com"}

	store := NewStore(c)

	var wg sync.WaitGroup
	stop := make(chan struct{})

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				snap := store.Get()
				_ = snap.Admin.Username
				_ = snap.Permission.DefaultPermissions
				_ = snap.Security.CORSAllowedOrigins
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			snap := store.Get().Clone()
			snap.Admin.Username = "admin"
			snap.Permission.DefaultPermissions = []string{"read", "write"}
			snap.Security.CORSAllowedOrigins = []string{"https://a.example.com", "https://b.example.com"}
			store.Store(snap)
		}
		close(stop)
	}()

	wg.Wait()

	if got := store.Get(); len(got.Permission.DefaultPermissions) != 2 {
		t.Fatalf("最终快照 DefaultPermissions = %v, 期望 2 项", got.Permission.DefaultPermissions)
	}
}

// TestCloneDeepCopiesSlices 验证 Clone 深拷贝切片，修改副本不影响原快照。
func TestCloneDeepCopiesSlices(t *testing.T) {
	c := &Config{}
	c.Permission.DefaultPermissions = []string{"read"}
	c.Security.CORSAllowedOrigins = []string{"https://a"}

	cp := c.Clone()
	cp.Permission.DefaultPermissions[0] = "changed"
	cp.Security.CORSAllowedOrigins[0] = "changed"

	if c.Permission.DefaultPermissions[0] != "read" {
		t.Error("Clone 未深拷贝 Permission.DefaultPermissions")
	}
	if c.Security.CORSAllowedOrigins[0] != "https://a" {
		t.Error("Clone 未深拷贝 Security.CORSAllowedOrigins")
	}
}

// TestCloneRoundTripLossless 验证 Clone 往返后配置与原始完全一致（无字段丢失）。
//
// 采用 JSON 往返实现深拷贝，若 Config 中混入不可序列化字段会在此暴露。
// 配合 conf.FillDefault 覆盖全部默认字段，使比对更完整。
func TestCloneRoundTripLossless(t *testing.T) {
	c := &Config{}
	if err := conf.FillDefault(c); err != nil {
		t.Fatalf("填充默认值失败: %v", err)
	}
	c.Admin.Username = "admin"
	c.Permission.DefaultPermissions = []string{"read", "write"}
	c.Security.CORSAllowedOrigins = []string{"https://a", "https://b"}

	cp := c.Clone()
	if !reflect.DeepEqual(c, cp) {
		t.Fatal("Clone 往返后与原配置不一致（疑似字段丢失）")
	}
}
