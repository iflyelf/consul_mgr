package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestExtractToken 验证登录凭证提取顺序：
//  1. 优先 HttpOnly Cookie（浏览器登录态依赖此项）
//  2. 其次 Authorization: Bearer（API 调用/旧会话兼容）
//  3. 最后 token 查询参数
//
// 回归背景：flyiam 曾因 handler 内另写一份「只读 Bearer/query」的实现，
// 导致 Cookie 登录态在部分接口失效（401 未登录）。此处固化提取规则，
// 确保全项目只有一份实现且优先读 Cookie。
func TestExtractToken(t *testing.T) {
	cases := []struct {
		name   string
		cookie string
		bearer string
		query  string
		want   string
	}{
		{"仅 Cookie", "cookie-token", "", "", "cookie-token"},
		{"仅 Bearer", "", "bearer-token", "", "bearer-token"},
		{"仅 query", "", "", "query-token", "query-token"},
		{"Cookie 优先于 Bearer", "cookie-token", "bearer-token", "", "cookie-token"},
		{"Cookie 优先于 query", "cookie-token", "", "query-token", "cookie-token"},
		{"Bearer 优先于 query", "", "bearer-token", "query-token", "bearer-token"},
		{"空 Cookie 值回退 Bearer", "", "bearer-token", "", "bearer-token"},
		{"全空", "", "", "", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			url := "/api/auth/userinfo"
			if tc.query != "" {
				url += "?token=" + tc.query
			}
			r := httptest.NewRequest(http.MethodGet, url, nil)
			if tc.cookie != "" {
				r.AddCookie(&http.Cookie{Name: AuthCookieName, Value: tc.cookie})
			}
			if tc.bearer != "" {
				r.Header.Set("Authorization", "Bearer "+tc.bearer)
			}
			if got := ExtractToken(r); got != tc.want {
				t.Fatalf("ExtractToken() = %q, 期望 %q", got, tc.want)
			}
		})
	}
}

// TestExtractToken_BearerCaseInsensitive 验证 scheme 大小写不敏感。
func TestExtractToken_BearerCaseInsensitive(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/auth/userinfo", nil)
	r.Header.Set("Authorization", "bearer lower-case-token")
	if got := ExtractToken(r); got != "lower-case-token" {
		t.Fatalf("ExtractToken() = %q, 期望 lower-case-token", got)
	}
}
