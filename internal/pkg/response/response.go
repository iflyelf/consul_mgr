package response

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Msg     string      `json:"msg,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// Success 成功响应
func Success(w http.ResponseWriter, data interface{}) {
	httpx.WriteJson(w, http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    data,
	})
}

// Error 错误响应
func Error(w http.ResponseWriter, code int, msg string) {
	httpx.WriteJson(w, http.StatusOK, Response{
		Code:    code,
		Message: msg,
		Data:    nil,
	})
}

// ErrorWithHttpStatus 带 HTTP 状态码的错误响应
func ErrorWithHttpStatus(w http.ResponseWriter, httpStatus int, code int, msg string) {
	httpx.WriteJson(w, httpStatus, Response{
		Code:    code,
		Message: msg,
		Data:    nil,
	})
}

// Unauthorized 未授权响应
func Unauthorized(w http.ResponseWriter, msg string) {
	ErrorWithHttpStatus(w, http.StatusUnauthorized, 401, msg)
}

// Forbidden 禁止访问响应
func Forbidden(w http.ResponseWriter, msg string) {
	ErrorWithHttpStatus(w, http.StatusForbidden, 403, msg)
}

// NotFound 未找到响应
func NotFound(w http.ResponseWriter, msg string) {
	ErrorWithHttpStatus(w, http.StatusNotFound, 404, msg)
}

// BadRequest 错误请求响应
func BadRequest(w http.ResponseWriter, msg string) {
	Error(w, 400, msg)
}

// InternalError 内部错误响应
func InternalError(w http.ResponseWriter, msg string) {
	Error(w, 500, msg)
}
