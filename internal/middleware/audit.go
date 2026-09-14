// Package middleware 提供审计日志中间件
package middleware

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/iflyelf/consul_mgr/internal/logic/audit"
	"github.com/iflyelf/consul_mgr/internal/svc"
)

// AuditMiddleware 审计日志中间件
type AuditMiddleware struct {
	ctx *svc.ServiceContext
}

// NewAuditMiddleware 创建审计日志中间件
func NewAuditMiddleware(ctx *svc.ServiceContext) *AuditMiddleware {
	return &AuditMiddleware{
		ctx: ctx,
	}
}

// responseWriter 包装的响应写入器，用于捕获状态码
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	rw.body.Write(b)
	return rw.ResponseWriter.Write(b)
}

// Handle 处理审计日志记录
//
// 功能：
//   - 记录所有 CUD 操作（Create, Update, Delete）
//   - 记录用户信息（Casdoor 用户）
//   - 记录请求和响应详情
//   - 记录操作结果（成功/失败）
func (m *AuditMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 只记录 POST, PUT, DELETE 操作
		if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodDelete {
			next.ServeHTTP(w, r)
			return
		}
		
		startTime := time.Now()
		
		// 获取用户信息
		ctx := r.Context()
		userID, _ := GetUserIdFromContext(ctx)
		username, _ := GetUsernameFromContext(ctx)
		
		// 如果没有用户信息，说明是未认证的请求，不记录
		if userID == "" {
			next.ServeHTTP(w, r)
			return
		}
		
		// 读取请求体
		var requestBody []byte
		if r.Body != nil {
			requestBody, _ = io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}
		
		// 包装响应写入器
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			body:           &bytes.Buffer{},
		}
		
		// 调用下一个处理器
		next.ServeHTTP(rw, r)
		
		// 确定操作类型
		action := getActionFromMethod(r.Method)
		
		// 确定资源类型
		resourceType, resourceID, resourceName := parseResourceFromPath(r.URL.Path)
		
		// 提取 group_id（如果有）
		var groupID *int64
		if gid := r.URL.Query().Get("group_id"); gid != "" {
			var id int64
			fmt.Sscanf(gid, "%d", &id)
			groupID = &id
		}
		
		// 构造详情
		details := map[string]interface{}{
			"method":       r.Method,
			"path":         r.URL.Path,
			"query":        r.URL.RawQuery,
			"request_body": string(requestBody),
			"duration_ms":  time.Since(startTime).Milliseconds(),
		}
		
		// 确定状态
		status := "success"
		errorMessage := ""
		if rw.statusCode >= 400 {
			status = "error"
			errorMessage = rw.body.String()
		}
		
		// 记录审计日志
		auditLogic := audit.NewAuditLogic(ctx, m.ctx.DB)
		err := auditLogic.LogAction(
			userID,
			username,
			action,
			resourceType,
			resourceID,
			resourceName,
			groupID,
			details,
			getClientIP(r),
			r.UserAgent(),
			status,
			errorMessage,
		)
		
		if err != nil {
			logx.Errorf("记录审计日志失败: %v", err)
		}
	}
}

// getActionFromMethod 从 HTTP 方法获取操作类型
func getActionFromMethod(method string) string {
	switch method {
	case http.MethodPost:
		return "create"
	case http.MethodPut:
		return "update"
	case http.MethodDelete:
		return "delete"
	default:
		return "unknown"
	}
}

// parseResourceFromPath 从路径解析资源类型和 ID
func parseResourceFromPath(path string) (resourceType, resourceID, resourceName string) {
	// 简单的路径解析
	// /api/groups -> groups
	// /api/groups/123 -> groups, 123
	// /api/instances -> instances
	
	if len(path) > 5 && path[:5] == "/api/" {
		path = path[5:]
	}
	
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		resourceType = parts[0]
	}
	if len(parts) > 1 {
		resourceID = parts[1]
		resourceName = parts[1]
	}
	
	return
}

// getClientIP 获取客户端 IP
func getClientIP(r *http.Request) string {
	// 尝试从 X-Forwarded-For 获取
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return strings.Split(ip, ",")[0]
	}
	
	// 尝试从 X-Real-IP 获取
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	
	// 从 RemoteAddr 获取
	return strings.Split(r.RemoteAddr, ":")[0]
}
