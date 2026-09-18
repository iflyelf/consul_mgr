// Package setting 应用设置：页面可配置，DB 优先 / env 兜底。
//
// 设计同 FlyIAM：设置项以 key/value 存于 app_settings 表；启动时应用到共享的
// *config.Config（指针），使既有读取点自动生效；页面修改后写 DB 并即时写回 Config。
package setting

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"github.com/iflyelf/consul_mgr/internal/config"
)

// Item 可配置项定义
type Item struct {
	Key    string
	Group  string
	Label  string
	Type   string // string / int / bool / list / secret
	Secret bool
	Get    func(*config.Config) string
	Set    func(*config.Config, string)
}

// Registry 全部可页面配置项
var Registry = []Item{
	// 安全 / 跨域
	{Key: "security.cookie_samesite", Group: "安全与跨域", Label: "Cookie SameSite", Type: "string",
		Get: func(c *config.Config) string { return c.Security.CookieSameSite },
		Set: func(c *config.Config, v string) { c.Security.CookieSameSite = v }},
	{Key: "security.cookie_secure", Group: "安全与跨域", Label: "Cookie Secure", Type: "string",
		Get: func(c *config.Config) string { return c.Security.CookieSecure },
		Set: func(c *config.Config, v string) { c.Security.CookieSecure = v }},
	{Key: "security.cookie_domain", Group: "安全与跨域", Label: "Cookie Domain", Type: "string",
		Get: func(c *config.Config) string { return c.Security.CookieDomain },
		Set: func(c *config.Config, v string) { c.Security.CookieDomain = v }},
	{Key: "security.cors_origins", Group: "安全与跨域", Label: "跨域来源（逗号分隔）", Type: "list",
		Get: func(c *config.Config) string { return strings.Join(c.Security.CORSAllowedOrigins, ",") },
		Set: func(c *config.Config, v string) { c.Security.CORSAllowedOrigins = splitTrim(v) }},

	// 审计
	{Key: "audit.enabled", Group: "审计日志", Label: "启用审计日志", Type: "bool",
		Get: func(c *config.Config) string { return strconv.FormatBool(c.Audit.Enabled) },
		Set: func(c *config.Config, v string) { c.Audit.Enabled = v == "true" }},
	{Key: "audit.retention_days", Group: "审计日志", Label: "保留天数", Type: "int",
		Get: func(c *config.Config) string { return strconv.Itoa(c.Audit.RetentionDays) },
		Set: func(c *config.Config, v string) {
			if n, err := strconv.Atoi(v); err == nil {
				c.Audit.RetentionDays = n
			}
		}},

	// 权限
	{Key: "permission.enable_service_group_auth", Group: "权限", Label: "启用服务组级授权", Type: "bool",
		Get: func(c *config.Config) string { return strconv.FormatBool(c.Permission.EnableServiceGroupAuth) },
		Set: func(c *config.Config, v string) { c.Permission.EnableServiceGroupAuth = v == "true" }},
	{Key: "permission.default", Group: "权限", Label: "默认权限（逗号分隔）", Type: "list",
		Get: func(c *config.Config) string { return strings.Join(c.Permission.DefaultPermissions, ",") },
		Set: func(c *config.Config, v string) { c.Permission.DefaultPermissions = splitTrim(v) }},

	// 日志
	{Key: "log.level", Group: "日志", Label: "日志级别", Type: "string",
		Get: func(c *config.Config) string { return c.LogConfig.Level },
		Set: func(c *config.Config, v string) { c.LogConfig.Level = v }},
	{Key: "log.format", Group: "日志", Label: "日志格式", Type: "string",
		Get: func(c *config.Config) string { return c.LogConfig.Format },
		Set: func(c *config.Config, v string) { c.LogConfig.Format = v }},

	// 管理员
	{Key: "admin.username", Group: "管理员", Label: "管理员用户名", Type: "string",
		Get: func(c *config.Config) string { return c.Admin.Username },
		Set: func(c *config.Config, v string) { c.Admin.Username = v }},
	{Key: "admin.password", Group: "管理员", Label: "管理员密码", Type: "secret", Secret: true,
		Get: func(c *config.Config) string { return c.Admin.Password },
		Set: func(c *config.Config, v string) { c.Admin.Password = v }},
	{Key: "admin.email", Group: "管理员", Label: "管理员邮箱", Type: "string",
		Get: func(c *config.Config) string { return c.Admin.Email },
		Set: func(c *config.Config, v string) { c.Admin.Email = v }},

	// JWT
	{Key: "jwt.access_expire", Group: "JWT", Label: "Access 过期(秒)", Type: "int",
		Get: func(c *config.Config) string { return strconv.Itoa(c.JWT.AccessExpire) },
		Set: func(c *config.Config, v string) {
			if n, err := strconv.Atoi(v); err == nil {
				c.JWT.AccessExpire = n
			}
		}},
	{Key: "jwt.refresh_expire", Group: "JWT", Label: "Refresh 过期(秒)", Type: "int",
		Get: func(c *config.Config) string { return strconv.Itoa(c.JWT.RefreshExpire) },
		Set: func(c *config.Config, v string) {
			if n, err := strconv.Atoi(v); err == nil {
				c.JWT.RefreshExpire = n
			}
		}},
	{Key: "jwt.issuer", Group: "JWT", Label: "签发者", Type: "string",
		Get: func(c *config.Config) string { return c.JWT.Issuer },
		Set: func(c *config.Config, v string) { c.JWT.Issuer = v }},

	// Casdoor 连接
	{Key: "casdoor.endpoint", Group: "Casdoor 连接", Label: "后端地址", Type: "string",
		Get: func(c *config.Config) string { return c.Casdoor.Endpoint },
		Set: func(c *config.Config, v string) { c.Casdoor.Endpoint = v }},
	{Key: "casdoor.public_endpoint", Group: "Casdoor 连接", Label: "浏览器地址", Type: "string",
		Get: func(c *config.Config) string { return c.Casdoor.PublicEndpoint },
		Set: func(c *config.Config, v string) { c.Casdoor.PublicEndpoint = v }},
	{Key: "casdoor.organization", Group: "Casdoor 连接", Label: "组织名", Type: "string",
		Get: func(c *config.Config) string { return c.Casdoor.OrganizationName },
		Set: func(c *config.Config, v string) { c.Casdoor.OrganizationName = v }},
	{Key: "casdoor.application", Group: "Casdoor 连接", Label: "应用名", Type: "string",
		Get: func(c *config.Config) string { return c.Casdoor.ApplicationName },
		Set: func(c *config.Config, v string) { c.Casdoor.ApplicationName = v }},
	{Key: "casdoor.certificate", Group: "Casdoor 连接", Label: "证书名", Type: "string",
		Get: func(c *config.Config) string { return c.Casdoor.Certificate },
		Set: func(c *config.Config, v string) { c.Casdoor.Certificate = v }},
	{Key: "casdoor.client_id", Group: "Casdoor 连接", Label: "Client ID", Type: "string",
		Get: func(c *config.Config) string { return c.Casdoor.ClientId },
		Set: func(c *config.Config, v string) { c.Casdoor.ClientId = v }},
	{Key: "casdoor.client_secret", Group: "Casdoor 连接", Label: "Client Secret", Type: "secret", Secret: true,
		Get: func(c *config.Config) string { return c.Casdoor.ClientSecret },
		Set: func(c *config.Config, v string) { c.Casdoor.ClientSecret = v }},
	{Key: "casdoor.user_cache_ttl", Group: "Casdoor 连接", Label: "用户缓存(秒)", Type: "int",
		Get: func(c *config.Config) string { return strconv.Itoa(c.Casdoor.UserCacheTTL) },
		Set: func(c *config.Config, v string) {
			if n, err := strconv.Atoi(v); err == nil {
				c.Casdoor.UserCacheTTL = n
			}
		}},
	{Key: "casdoor.default_password", Group: "Casdoor 连接", Label: "新增用户默认密码", Type: "secret", Secret: true,
		Get: func(c *config.Config) string { return c.Casdoor.DefaultPassword },
		Set: func(c *config.Config, v string) { c.Casdoor.DefaultPassword = v }},

	// 默认 Consul 配置
	{Key: "consul.default_address", Group: "Consul 默认", Label: "默认地址", Type: "string",
		Get: func(c *config.Config) string { return c.Consul.DefaultAddress },
		Set: func(c *config.Config, v string) { c.Consul.DefaultAddress = v }},
	{Key: "consul.default_token", Group: "Consul 默认", Label: "默认 Token", Type: "secret", Secret: true,
		Get: func(c *config.Config) string { return c.Consul.DefaultToken },
		Set: func(c *config.Config, v string) { c.Consul.DefaultToken = v }},
	{Key: "consul.default_datacenter", Group: "Consul 默认", Label: "默认数据中心", Type: "string",
		Get: func(c *config.Config) string { return c.Consul.DefaultDatacenter },
		Set: func(c *config.Config, v string) { c.Consul.DefaultDatacenter = v }},
	{Key: "consul.timeout", Group: "Consul 默认", Label: "操作超时(秒)", Type: "int",
		Get: func(c *config.Config) string { return strconv.Itoa(c.Consul.Timeout) },
		Set: func(c *config.Config, v string) {
			if n, err := strconv.Atoi(v); err == nil {
				c.Consul.Timeout = n
			}
		}},
	{Key: "consul.max_concurrency", Group: "Consul 默认", Label: "最大并发", Type: "int",
		Get: func(c *config.Config) string { return strconv.Itoa(c.Consul.MaxConcurrency) },
		Set: func(c *config.Config, v string) {
			if n, err := strconv.Atoi(v); err == nil {
				c.Consul.MaxConcurrency = n
			}
		}},

	// FlyIAM 集成（同步用户字段定义）
	{Key: "flyiam.api_endpoint", Group: "FlyIAM 集成", Label: "FlyIAM API 地址", Type: "string",
		Get: func(c *config.Config) string { return c.FlyIAM.Endpoint },
		Set: func(c *config.Config, v string) { c.FlyIAM.Endpoint = v }},
	{Key: "flyiam.service_token", Group: "FlyIAM 集成", Label: "API 令牌", Type: "secret", Secret: true,
		Get: func(c *config.Config) string { return c.FlyIAM.ServiceToken },
		Set: func(c *config.Config, v string) { c.FlyIAM.ServiceToken = v }},
	{Key: "flyiam.sync_enabled", Group: "FlyIAM 集成", Label: "启用定时同步", Type: "bool",
		Get: func(c *config.Config) string { return strconv.FormatBool(c.FlyIAM.SyncEnabled) },
		Set: func(c *config.Config, v string) { c.FlyIAM.SyncEnabled = v == "true" }},
	{Key: "flyiam.sync_on_startup", Group: "FlyIAM 集成", Label: "启动时同步", Type: "bool",
		Get: func(c *config.Config) string { return strconv.FormatBool(c.FlyIAM.SyncOnStartup) },
		Set: func(c *config.Config, v string) { c.FlyIAM.SyncOnStartup = v == "true" }},
	{Key: "flyiam.sync_interval", Group: "FlyIAM 集成", Label: "同步间隔（6h/30m/1d）", Type: "string",
		Get: func(c *config.Config) string { return c.FlyIAM.SyncInterval },
		Set: func(c *config.Config, v string) { c.FlyIAM.SyncInterval = v }},
}

var registryMap = func() map[string]Item {
	m := make(map[string]Item, len(Registry))
	for _, it := range Registry {
		m[it.Key] = it
	}
	return m
}()

// Service 设置服务
//
// 并发模型：配置以 config.Store（atomic 快照）承载。修改时先 Clone 副本、
// 在副本上应用，再原子替换，读者始终看到不可变快照，避免读写数据竞争。
// mu 仅用于串行化「副本修改 + 替换」过程，防止并发保存相互覆盖。
type Service struct {
	mu    sync.Mutex
	db    sqlx.SqlConn
	store *config.Store
	items map[string]Item
}

// NewService 创建设置服务（store 为共享配置快照存储）
func NewService(db sqlx.SqlConn, store *config.Store) *Service {
	return &Service{db: db, store: store, items: registryMap}
}

// Load 从数据库加载设置并应用到配置
func (s *Service) Load(ctx context.Context) error {
	var rows []struct {
		Key   string `db:"key"`
		Value string `db:"value"`
	}
	if err := s.db.QueryRowsCtx(ctx, &rows, `SELECT key, value FROM app_settings`); err != nil {
		return fmt.Errorf("读取应用设置失败: %w", err)
	}
	if len(rows) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	snap := s.store.Get().Clone()
	for _, r := range rows {
		if it, ok := s.items[r.Key]; ok {
			it.Set(snap, r.Value)
		}
	}
	s.store.Store(snap)
	return nil
}

// Apply 写入 DB 并即时应用到内存配置。
//
// 返回实际生效的 key 列表（供调用方判断是否需要热重载相关组件，如 Casdoor 客户端）。
func (s *Service) Apply(ctx context.Context, kv map[string]string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap := s.store.Get().Clone()
	// 无论中途是否出错都提交已应用的键：保证内存快照与已写入 DB 的部分保持一致。
	defer s.store.Store(snap)
	applied := make([]string, 0, len(kv))
	for k, v := range kv {
		it, ok := s.items[k]
		if !ok {
			continue
		}
		if _, err := s.db.ExecCtx(ctx,
			`INSERT INTO app_settings (key, value, updated_at) VALUES ($1,$2,NOW())
			 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`,
			k, v); err != nil {
			return applied, fmt.Errorf("保存设置 %s 失败: %w", k, err)
		}
		it.Set(snap, v)
		applied = append(applied, k)
	}
	return applied, nil
}

// NeedsCasdoorReload 判断本次变更是否涉及 Casdoor 连接配置（需重建客户端）
func NeedsCasdoorReload(applied []string) bool {
	for _, k := range applied {
		if strings.HasPrefix(k, "casdoor.") {
			return true
		}
	}
	return false
}

// View 返回全部可配置项（secret 以占位符返回）
func (s *Service) View() []map[string]interface{} {
	cfg := s.store.Get()
	out := make([]map[string]interface{}, 0, len(Registry))
	for _, it := range Registry {
		val := it.Get(cfg)
		if it.Secret && val != "" {
			val = "******"
		}
		out = append(out, map[string]interface{}{
			"key": it.Key, "group": it.Group, "label": it.Label,
			"type": it.Type, "secret": it.Secret, "value": val,
		})
	}
	return out
}

func splitTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// MaskedSecret 占位符
const MaskedSecret = "******"

// IsMasked 判断是否为占位符
func IsMasked(v string) bool { return v == MaskedSecret }
