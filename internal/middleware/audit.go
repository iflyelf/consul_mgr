// Package middleware 提供审计日志中间件
package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/iflyelf/consul_mgr/internal/logic/audit"
	"github.com/iflyelf/consul_mgr/internal/svc"
)

// sensitiveKeys 审计日志中需要脱敏的字段名（小写匹配）
var sensitiveKeys = []string{
	"password", "passwd", "secret", "token", "credential",
	"client_secret", "access_key", "secret_key",
}

// isSensitiveKey 判断字段名是否敏感
func isSensitiveKey(key string) bool {
	k := strings.ToLower(key)
	for _, s := range sensitiveKeys {
		if strings.Contains(k, s) {
			return true
		}
	}
	return false
}

// sanitizeValue 递归脱敏 JSON 值中的敏感字段
func sanitizeValue(v interface{}) interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		for k, val := range t {
			if isSensitiveKey(k) {
				t[k] = "******"
				continue
			}
			t[k] = sanitizeValue(val)
		}
		return t
	case []interface{}:
		for i, item := range t {
			t[i] = sanitizeValue(item)
		}
		return t
	default:
		return v
	}
}

// sanitizeBody 对请求体进行脱敏：JSON 逐字段脱敏；非 JSON 仅保留类型说明。
//
// 目的：审计日志不应明文保存 token / 密码等敏感信息。
func sanitizeBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	var parsed interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		// 非 JSON（如文件上传的表单）不做内容记录，避免误存敏感数据
		return "(非 JSON 请求体，已省略)"
	}
	sanitized := sanitizeValue(parsed)
	out, err := json.Marshal(sanitized)
	if err != nil {
		return "(请求体脱敏失败)"
	}
	const maxLen = 8 << 10
	if len(out) > maxLen {
		return string(out[:maxLen]) + "...(截断)"
	}
	return string(out)
}

// sanitizeQuery 对查询串中的敏感参数脱敏
func sanitizeQuery(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}
	parts := strings.Split(rawQuery, "&")
	for i, p := range parts {
		kv := strings.SplitN(p, "=", 2)
		if len(kv) == 2 && isSensitiveKey(kv[0]) {
			parts[i] = kv[0] + "=******"
		}
	}
	return strings.Join(parts, "&")
}

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
		
		// 读取请求体（限制大小，避免大 body 造成内存膨胀）
		var requestBody []byte
		if r.Body != nil {
			const maxAuditBody = 64 << 10 // 64KB
			requestBody, _ = io.ReadAll(io.LimitReader(r.Body, maxAuditBody))
			r.Body = io.NopCloser(io.MultiReader(bytes.NewReader(requestBody), r.Body))
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
		
		// 构造详情（请求体按敏感字段脱敏，避免 token/密码明文落库）
		details := map[string]interface{}{
			"method":       r.Method,
			"path":         r.URL.Path,
			"query":        sanitizeQuery(r.URL.RawQuery),
			"request_body": sanitizeBody(requestBody),
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
