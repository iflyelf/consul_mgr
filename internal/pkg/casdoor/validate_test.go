package casdoor

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// mockCasdoor 模拟 Casdoor：仅接受 token == validToken，其余返回令牌不存在
func mockCasdoor(validToken string, user map[string]interface{}) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/get-account" {
			_, _ = w.Write([]byte(`{"status":"error","msg":"not found"}`))
			return
		}
		if extractBearer(r) != validToken {
			_, _ = w.Write([]byte(`{"status":"error","msg":"Access token doesn't exist in database"}`))
			return
		}
		body, _ := json.Marshal(map[string]interface{}{"status": "ok", "data": user})
		_, _ = w.Write(body)
	}))
}

func extractBearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	return strings.TrimPrefix(h, "Bearer ")
}

func newTestClient(t *testing.T, endpoint string) *Client {
	t.Helper()
	c, err := NewClient(&Config{
		Endpoint:         endpoint,
		ClientId:         "biz-id",
		ClientSecret:     "biz-secret",
		OrganizationName: "flyiam",
		ApplicationName:  "flyiam",
	})
	if err != nil {
		t.Fatalf("NewClient 失败: %v", err)
	}
	return c
}

// 伪造令牌（不在 Casdoor 数据库中）必须被拒绝，绝不能采信 isAdmin=true
func TestValidateToken_RejectsForgedToken(t *testing.T) {
	srv := mockCasdoor("the-real-token", map[string]interface{}{"name": "admin", "isAdmin": true})
	defer srv.Close()

	c := newTestClient(t, srv.URL)

	forged := makeJWT(map[string]interface{}{
		"name": "hacker", "isAdmin": true, "exp": float64(time.Now().Add(time.Hour).Unix()),
	})
	user, err := c.ValidateToken(forged)
	if err == nil {
		t.Fatalf("伪造令牌必须被拒绝，实际通过：user=%+v", user)
	}
	t.Logf("伪造令牌被拒绝: %v", err)
}

// 有效令牌通过校验，并返回权威用户信息
func TestValidateToken_AcceptsValidToken(t *testing.T) {
	valid := "valid-access-token"
	srv := mockCasdoor(valid, map[string]interface{}{
		"id": "u1", "name": "admin", "displayName": "管理员", "isAdmin": true,
	})
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	user, err := c.ValidateToken(valid)
	if err != nil {
		t.Fatalf("有效令牌应通过，实际: %v", err)
	}
	if user.Name != "admin" || !user.IsAdmin {
		t.Fatalf("用户信息不符: %+v", user)
	}
}

// 被禁用（或已删除）用户在权威校验阶段被拒绝
func TestValidateToken_RejectsForbiddenUser(t *testing.T) {
	valid := "valid-access-token"
	srv := mockCasdoor(valid, map[string]interface{}{
		"name": "banned", "isForbidden": true,
	})
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	if _, err := c.ValidateToken(valid); err == nil {
		t.Fatal("被禁用用户必须被拒绝")
	}
}

// 已过期令牌本地快速失败（无需回源）
func TestValidateToken_RejectsExpiredLocally(t *testing.T) {
	srv := mockCasdoor("never", nil)
	defer srv.Close()
	c := newTestClient(t, srv.URL)

	expired := makeJWT(map[string]interface{}{
		"name": "x", "exp": float64(time.Now().Add(-time.Hour).Unix()),
	})
	if _, err := c.ValidateToken(expired); err == nil {
		t.Fatal("过期令牌必须被拒绝")
	}
}

// 校验结果命中缓存，避免每请求回源
func TestValidateToken_CachesResult(t *testing.T) {
	valid := "valid-access-token"
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = w.Write([]byte(`{"status":"ok","data":{"name":"admin","isAdmin":true}}`))
	}))
	defer srv.Close()
	_ = valid

	c := newTestClient(t, srv.URL)
	for i := 0; i < 3; i++ {
		if _, err := c.ValidateToken("tok"); err != nil {
			t.Fatalf("校验失败: %v", err)
		}
	}
	if calls != 1 {
		t.Fatalf("期望回源 1 次（其余命中缓存），实际 %d 次", calls)
	}
}

// makeJWT 生成仅结构合法的 JWT（签名无效；载荷未被篡改，便于本地 exp 判断）
func makeJWT(claims map[string]interface{}) string {
	enc := func(v interface{}) string {
		b, _ := json.Marshal(v)
		return base64.RawURLEncoding.EncodeToString(b)
	}
	header := enc(map[string]string{"alg": "HS256", "typ": "JWT"})
	payload := enc(claims)
	return header + "." + payload + "." + base64.RawURLEncoding.EncodeToString([]byte("bad-signature"))
}
