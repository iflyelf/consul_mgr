// Package cache 提供基于 Redis 的缓存封装
//
// 功能：
//   - Redis 连接管理（配置来自环境变量，零硬编码）
//   - JSON 对象缓存读写
//   - 优雅降级：Redis 不可用时自动跳过缓存，不影响主流程
//
// 作者: iflyelf
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// Cache 缓存封装
type Cache struct {
	rdb     *redis.Redis
	enabled bool
	ttl     time.Duration
}

// Config 缓存配置
type Config struct {
	Enabled  bool
	Host     string
	Port     int
	Password string
	DB       int
	TTL      int // 秒
}

// New 创建缓存实例
//
// 说明：连接失败时不返回错误，仅禁用缓存并打印告警，保证服务可正常启动。
func New(cfg Config) *Cache {
	c := &Cache{
		enabled: false,
		ttl:     time.Duration(cfg.TTL) * time.Second,
	}

	if !cfg.Enabled {
		logx.Info("[cache] Redis 缓存已禁用")
		return c
	}

	if cfg.TTL <= 0 {
		c.ttl = 30 * time.Second
	}

	if cfg.Host == "" {
		logx.Error("[cache] Redis 地址为空，缓存已禁用")
		return c
	}
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	rdb, err := redis.NewRedis(redis.RedisConf{
		Host: addr,
		Type: "node",
		Pass: cfg.Password,
	})
	if err != nil {
		logx.Errorf("[cache] Redis 初始化失败，缓存已禁用: %v", err)
		return c
	}

	if !rdb.Ping() {
		logx.Errorf("[cache] Redis 连接失败，缓存已禁用: %s", addr)
		return c
	}

	// 切换 DB（如配置）
	if cfg.DB > 0 {
		if _, err := rdb.Do("SELECT", cfg.DB); err != nil {
			logx.Errorf("[cache] 选择 Redis DB %d 失败: %v", cfg.DB, err)
		}
	}

	c.rdb = rdb
	c.enabled = true

	logx.Infof("[cache] Redis 缓存已启用: %s (TTL=%ds)", addr, cfg.TTL)
	return c
}

// Enabled 返回缓存是否可用
func (c *Cache) Enabled() bool {
	return c != nil && c.enabled && c.rdb != nil
}

// Get 读取并反序列化缓存
//
// 返回:
//   bool - 是否命中
func (c *Cache) Get(ctx context.Context, key string, dest interface{}) bool {
	if !c.Enabled() {
		return false
	}
	val, err := c.rdb.GetCtx(ctx, key)
	if err != nil || val == "" {
		return false
	}
	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return false
	}
	return true
}

// Set 序列化并写入缓存
func (c *Cache) Set(ctx context.Context, key string, value interface{}) {
	if !c.Enabled() {
		return
	}
	data, err := json.Marshal(value)
	if err != nil {
		return
	}
	if err := c.rdb.SetexCtx(ctx, key, string(data), int(c.ttl.Seconds())); err != nil {
		logx.Errorf("[cache] 写入缓存失败 key=%s: %v", key, err)
	}
}

// Del 删除缓存
func (c *Cache) Del(ctx context.Context, keys ...string) {
	if !c.Enabled() || len(keys) == 0 {
		return
	}
	if _, err := c.rdb.DelCtx(ctx, keys...); err != nil {
		logx.Errorf("[cache] 删除缓存失败: %v", err)
	}
}

// DelPrefix 按前缀删除缓存（用于数据变更后失效相关缓存）
func (c *Cache) DelPrefix(ctx context.Context, prefix string) {
	if !c.Enabled() {
		return
	}
	var cursor uint64
	for {
		keys, next, err := c.rdb.ScanCtx(ctx, cursor, prefix+"*", 100)
		if err != nil {
			return
		}
		if len(keys) > 0 {
			if _, err := c.rdb.DelCtx(ctx, keys...); err != nil {
				logx.Errorf("[cache] 按前缀删除缓存失败: %v", err)
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
}
