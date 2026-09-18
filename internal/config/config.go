package config

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/zeromicro/go-zero/rest"
)

// Config 应用配置
//
// 说明：所有可调参数均支持环境变量注入（env=XXX / YAML 占位符 ${XXX:default}），
// 代码中不硬编码任何地址、端口或凭据，便于容器化与多环境部署。
type Config struct {
	rest.RestConf

	Database struct {
		DSN             string `json:",optional,env=DATABASE_URL"`
		Host            string `json:",optional,default=localhost,env=DB_HOST"`
		Port            int    `json:",optional,default=5432,env=DB_PORT"`
		User            string `json:",optional,default=postgres,env=DB_USER"`
		Password        string `json:",optional,env=DB_PASSWORD"`
		DBName          string `json:",optional,default=consul_mgr,env=DB_NAME"`
		SSLMode         string `json:",optional,default=disable,env=DB_SSLMODE"`
		MaxOpenConns    int    `json:",optional,default=100,env=DB_MAX_OPEN_CONNS"`
		MaxIdleConns    int    `json:",optional,default=10,env=DB_MAX_IDLE_CONNS"`
		ConnMaxLifetime int    `json:",optional,default=3600,env=DB_CONN_MAX_LIFETIME"` // seconds
	}

	JWT struct {
		Secret string `json:",env=JWT_SECRET"`
		// 注意：go-zero 会把带 env 标签的 int64 字段当作 time.Duration 解析，
		// 导致 JWT_ACCESS_EXPIRE=7200 报 "missing unit in duration"。
		// 此处使用 int（单位：秒），env 传纯数字即可。
		AccessExpire  int    `json:",default=7200,env=JWT_ACCESS_EXPIRE"`    // 2 hours
		RefreshExpire int    `json:",default=604800,env=JWT_REFRESH_EXPIRE"` // 7 days
		Issuer        string `json:",default=consul_mgr,env=JWT_ISSUER"`
	}

	Admin struct {
		Username string `json:",default=admin,env=ADMIN_USERNAME"`
		Password string `json:",env=ADMIN_PASSWORD"`
		Email    string `json:",optional,env=ADMIN_EMAIL"`
	}

	Consul struct {
		DefaultAddress    string `json:",optional,env=CONSUL_ADDRESS"`
		DefaultToken      string `json:",optional,env=CONSUL_TOKEN"`
		DefaultDatacenter string `json:",default=dc1,env=CONSUL_DATACENTER"`
		Timeout           int    `json:",default=10,env=CONSUL_TIMEOUT"`           // seconds
		MaxConcurrency    int    `json:",default=16,env=CONSUL_MAX_CONCURRENCY"`   // 批量查询/操作并发上限
	}

	Web struct {
		Embedded  bool   `json:",default=true,env=WEB_EMBEDDED"`
		StaticDir string `json:",default=./web/dist,env=WEB_STATIC_DIR"`
		Port      int    `json:",default=5173,env=WEB_PORT"`
		Host      string `json:",default=0.0.0.0,env=WEB_HOST"`
	}

	LogConfig struct {
		Level  string `json:",default=info,env=LOG_LEVEL"`
		Format string `json:",default=json,env=LOG_FORMAT"`
	}

	Audit struct {
		Enabled       bool `json:",default=true,env=AUDIT_ENABLED"`
		RetentionDays int  `json:",default=90,env=AUDIT_RETENTION_DAYS"`
	}

	Redis struct {
		Enabled  bool   `json:",default=true,env=REDIS_ENABLED"`
		Host     string `json:",default=localhost,env=REDIS_HOST"`
		Port     int    `json:",default=6379,env=REDIS_PORT"`
		Password string `json:",optional,env=REDIS_PASSWORD"`
		DB       int    `json:",default=0,env=REDIS_DB"`
		// 缓存过期时间（秒）
		TTL int `json:",default=300,env=REDIS_CACHE_TTL"`
	}

	Casdoor struct {
		Endpoint         string `json:",env=CASDOOR_ENDPOINT"`
		PublicEndpoint   string `json:",optional,env=CASDOOR_PUBLIC_ENDPOINT"`
		ClientId         string `json:",env=CASDOOR_CLIENT_ID"`
		ClientSecret     string `json:",env=CASDOOR_CLIENT_SECRET"`
		Certificate      string `json:",optional,env=CASDOOR_CERTIFICATE"`
		OrganizationName string `json:",default=flyiam,env=CASDOOR_ORGANIZATION"`
		ApplicationName  string `json:",default=flyiam,env=CASDOOR_APPLICATION"`
		// DefaultPassword 新增用户时的默认密码。
		// 不设代码默认值，必须显式注入（环境变量 CASDOOR_DEFAULT_PASSWORD / Secret），
		// 避免弱口令被静默沿用。
		DefaultPassword string `json:",optional,env=CASDOOR_DEFAULT_PASSWORD"`
		// UserCacheTTL 用户列表缓存时长（秒）。用户列表可能被外部（Casdoor）
		// 直接修改，故使用较短 TTL，缩短「外部改动不可见」窗口。
		UserCacheTTL int `json:",default=30,env=CASDOOR_USER_CACHE_TTL"`
	}

	Permission struct {
		EnableServiceGroupAuth bool     `json:",default=true,env=PERMISSION_ENABLE_SERVICE_GROUP_AUTH"`
		DefaultPermissions     []string `json:",default=[read],env=PERMISSION_DEFAULT"`
	}

	// FlyIAM 服务对接（用于同步用户字段定义等服务间调用）
	FlyIAM struct {
		// Endpoint FlyIAM 业务 API 地址（如 https://flyiam.example.com）
		Endpoint string `json:",optional,env=CONSUL_MGR_FLYIAM_API_ENDPOINT"`
		// ServiceToken 服务间凭证，需与 FlyIAM 的 SERVICE_TOKEN 一致
		ServiceToken string `json:",optional,env=CONSUL_MGR_FLYIAM_SERVICE_TOKEN"`
	}

	// Security 登录凭证与跨域相关配置
	Security struct {
		// CookieSameSite 登录 Cookie 的 SameSite 策略：lax / strict / none
		CookieSameSite string `json:",default=lax,env=AUTH_COOKIE_SAMESITE"`
		// CookieSecure 是否仅通过 HTTPS 发送：auto / true / false
		CookieSecure string `json:",default=auto,env=AUTH_COOKIE_SECURE"`
		// CookieDomain Cookie 作用域（跨子域共享时设为 .example.com）
		CookieDomain string `json:",optional,env=AUTH_COOKIE_DOMAIN"`
		// CORSAllowedOrigins 允许的跨域来源（逗号分隔，精确匹配）。
		//   为空则不启用跨域；跨域时必须显式列出来源（不能用 *）。
		CORSAllowedOrigins []string `json:",optional"`
	}
}

// CookieConfig 从安全配置构造 Cookie 配置
func (c *Config) CookieConfig() CookieConfig {
	return CookieConfig{
		SameSite: c.Security.CookieSameSite,
		Secure:   c.Security.CookieSecure,
		Domain:   c.Security.CookieDomain,
	}
}

// DSN 返回数据库连接串（优先 DATABASE_URL，否则由分项拼装）
func (c *Config) DSN() string {
	if c.Database.DSN != "" {
		return c.Database.DSN
	}
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host, c.Database.Port, c.Database.User,
		c.Database.Password, c.Database.DBName, c.Database.SSLMode)
}

// MaintenanceDSN 返回连接 postgres 维护库的连接串（用于首次自动建库）
func (c *Config) MaintenanceDSN() string {
	if c.Database.DSN != "" {
		return replaceDBNameInDSN(c.Database.DSN, "postgres")
	}
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=postgres sslmode=%s",
		c.Database.Host, c.Database.Port, c.Database.User, c.Database.Password, c.Database.SSLMode)
}

// replaceDBNameInDSN 将 DSN 中的库名替换为指定值，兼容 URL 与 key=value 两种格式。
func replaceDBNameInDSN(dsn, dbName string) string {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		if u, err := url.Parse(dsn); err == nil {
			u.Path = "/" + dbName
			return u.String()
		}
		return dsn
	}
	re := regexp.MustCompile(`(^|\s)dbname=[^\s]+`)
	if re.MatchString(dsn) {
		return re.ReplaceAllString(dsn, "${1}dbname="+dbName)
	}
	return dsn + " dbname=" + dbName
}

// Validate 校验必填配置（失败即退出，避免以错误配置运行）
func (c *Config) Validate() error {
	if c.JWT.Secret != "" && len(c.JWT.Secret) < 32 {
		return fmt.Errorf("JWT 密钥长度不足: 至少需要 32 个字符，当前 %d 个", len(c.JWT.Secret))
	}
	if c.Casdoor.Endpoint == "" {
		return fmt.Errorf("Casdoor 端点未设置: 请设置 CASDOOR_ENDPOINT 环境变量")
	}
	if c.Casdoor.ClientId == "" || c.Casdoor.ClientSecret == "" {
		return fmt.Errorf("Casdoor 应用凭据未设置: 请设置 CASDOOR_CLIENT_ID / CASDOOR_CLIENT_SECRET")
	}
	if c.Casdoor.DefaultPassword == "" {
		return fmt.Errorf("Casdoor 默认密码未设置: 请设置 CASDOOR_DEFAULT_PASSWORD 环境变量或 Secret（不再提供弱口令默认值）")
	}
	return nil
}

// ApplyEnvOverrides 用环境变量覆盖服务端口、监听地址、运行模式等基础配置
//
// 说明:
//   go-zero 的 rest.RestConf（Host/Port/Mode/Timeout）不携带 env 标签，
//   且 conf.Load 默认不做 ${VAR} 展开，因此这里显式读取环境变量覆盖，
//   保证端口/地址/模式可通过容器编排自定义，代码中不硬编码。
//
// 支持的环境变量:
//   SERVER_HOST    监听地址（默认取配置文件值）
//   SERVER_PORT    监听端口（默认取配置文件值）
//   SERVER_MODE    运行模式 dev|test|rt|pre|pro
//   SERVER_TIMEOUT 服务端超时（毫秒）
func (c *Config) ApplyEnvOverrides() {
	if v := os.Getenv("SERVER_HOST"); v != "" {
		c.Host = v
	}
	if v := os.Getenv("SERVER_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			c.Port = p
		}
	}
	if v := os.Getenv("SERVER_MODE"); v != "" {
		c.Mode = v
	}
	if v := os.Getenv("SERVER_TIMEOUT"); v != "" {
		if t, err := strconv.ParseInt(v, 10, 64); err == nil {
			c.Timeout = t
		}
	}
	// 兜底默认值：纯环境变量部署（无配置文件）且未设置 SERVER_PORT 时避免监听 0 端口
	if c.Host == "" {
		c.Host = "0.0.0.0"
	}
	if c.Port == 0 {
		c.Port = 8080
	}

	// 跨域来源白名单支持环境变量（逗号分隔）
	if v := os.Getenv("CORS_ALLOWED_ORIGINS"); v != "" {
		c.Security.CORSAllowedOrigins = splitAndTrim(v)
	}
}

// splitAndTrim 按逗号切分并去除空白
func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
