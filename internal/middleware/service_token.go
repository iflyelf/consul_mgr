package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/iflyelf/consul_mgr/internal/pkg/response"
)

// ServiceTokenHeader 服务间调用的令牌请求头（与 FlyIAM 协议一致）。
const ServiceTokenHeader = "X-Service-Token"

// ServiceTokenMiddleware 校验服务间调用令牌。
//
// 用途：upgrade_mgr 等平台以「服务身份」只读拉取主机清单，没有浏览器会话，
// 无法走 Casdoor 认证。这里用独立的共享令牌，与用户会话完全隔离：
//   - 只保护显式挂载的只读路由，不影响任何用户态 API；
//   - 令牌为空时调用方直接拿到 503，提示管理员先配置；
//   - 使用常量时间比较，避免按字节比较带来的时序侧信道。
//
// tokenFn 每次请求求值，页面改完设置即时生效，无需重启。
type ServiceTokenMiddleware struct {
	tokenFn func() string
}

// NewServiceTokenMiddleware 创建中间件；tokenFn 返回当前配置的令牌。
func NewServiceTokenMiddleware(tokenFn func() string) *ServiceTokenMiddleware {
	return &ServiceTokenMiddleware{tokenFn: tokenFn}
}

// Handle 包装处理器，校验 X-Service-Token。
func (m *ServiceTokenMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		want := ""
		if m.tokenFn != nil {
			want = strings.TrimSpace(m.tokenFn())
		}
		if want == "" {
			response.ErrorWithHttpStatus(w, http.StatusServiceUnavailable, 503,
				"未配置服务间令牌，请在 consul_mgr 的「设置 → 服务间 API」填写后重试")
			return
		}
		got := strings.TrimSpace(r.Header.Get(ServiceTokenHeader))
		// 先比长度再比内容：ConstantTimeCompare 要求等长切片。
		if len(got) != len(want) ||
			subtle.ConstantTimeCompare([]byte(got), []byte(want)) != 1 {
			response.Unauthorized(w, "服务令牌无效")
			return
		}
		next(w, r)
	}
}
