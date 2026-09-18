package config

import "testing"

// validBase 构造一份完整合法配置，供各用例按需破坏。
func validBase() *Config {
	c := &Config{}
	c.JWT.Secret = "0123456789abcdef0123456789abcdef" // 32 位
	c.Casdoor.Endpoint = "http://casdoor:8000"
	c.Casdoor.ClientId = "id"
	c.Casdoor.ClientSecret = "secret"
	c.Casdoor.DefaultPassword = "user-password"
	return c
}

// TestValidate_PreDatabase 验证 Validate 仅校验与页面设置无关的前置项，
// 不再因 Casdoor 配置为空而拦截（已下沉到 ValidateCasdoor）。
func TestValidate_PreDatabase(t *testing.T) {
	// Casdoor 全空也应通过前置校验
	c := validBase()
	c.Casdoor.Endpoint = ""
	c.Casdoor.ClientId = ""
	c.Casdoor.ClientSecret = ""
	c.Casdoor.DefaultPassword = ""
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate 不应校验 Casdoor，但报错: %v", err)
	}

	// JWT 非空但过短应报错
	c = validBase()
	c.JWT.Secret = "short"
	if err := c.Validate(); err == nil {
		t.Fatal("JWT 密钥过短应报错")
	}

	// JWT 为空按现状放行（仅长度校验）
	c = validBase()
	c.JWT.Secret = ""
	if err := c.Validate(); err != nil {
		t.Fatalf("JWT 为空不应报错: %v", err)
	}
}

// TestValidateCasdoor 验证 Casdoor 校验：Endpoint / 凭据 / 默认密码均必填。
func TestValidateCasdoor(t *testing.T) {
	if err := validBase().ValidateCasdoor(); err != nil {
		t.Fatalf("完整配置应通过: %v", err)
	}

	cases := []struct {
		name        string
		mutate      func(*Config)
		errContains string
	}{
		{"端点缺失", func(c *Config) { c.Casdoor.Endpoint = "" }, "Casdoor 端点"},
		{"clientId缺失", func(c *Config) { c.Casdoor.ClientId = "" }, "CASDOOR_CLIENT_ID"},
		{"clientSecret缺失", func(c *Config) { c.Casdoor.ClientSecret = "" }, "CASDOOR_CLIENT_ID"},
		{"默认密码缺失", func(c *Config) { c.Casdoor.DefaultPassword = "" }, "默认密码"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := validBase()
			tc.mutate(c)
			err := c.ValidateCasdoor()
			if err == nil {
				t.Fatalf("期望报错，但校验通过")
			}
			if !contains(err.Error(), tc.errContains) {
				t.Fatalf("错误信息 %q 未包含 %q", err.Error(), tc.errContains)
			}
		})
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
