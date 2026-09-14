// Package main Casdoor 自动配置工具
//
// 功能：
//   - 自动创建组织（consul_mgr）
//   - 自动创建应用（consul_manager）
//   - 自动配置回调 URL
//   - 自动创建角色（admin、operator、viewer）
//   - 自动创建权限资源
//   - 自动生成并输出配置信息
//
// 作者: iflyelf
// 创建时间: 2024-01-15
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	// Casdoor 默认配置
	defaultEndpoint      = "http://localhost:8000"
	defaultAdminUser     = "admin"
	defaultAdminPassword = "123"
	
	// 组织配置
	organizationName = "consul_mgr"
	organizationDisplayName = "Consul Manager"
	
	// 应用配置
	applicationName = "consul_manager"
	applicationDisplayName = "Consul Manager 应用"
	callbackURL = "http://localhost:8080/api/auth/callback"
	homeURL = "http://localhost:8080"
)

// CasdoorConfig Casdoor 配置信息
type CasdoorConfig struct {
	Endpoint         string `json:"endpoint"`
	ClientID         string `json:"client_id"`
	ClientSecret     string `json:"client_secret"`
	OrganizationName string `json:"organization_name"`
	ApplicationName  string `json:"application_name"`
}

// Organization 组织结构
type Organization struct {
	Owner       string `json:"owner"`
	Name        string `json:"name"`
	CreatedTime string `json:"createdTime"`
	DisplayName string `json:"displayName"`
	WebsiteUrl  string `json:"websiteUrl"`
	Favicon     string `json:"favicon"`
	PasswordType string `json:"passwordType"`
	PasswordSalt string `json:"passwordSalt"`
	PhonePrefix  string `json:"phonePrefix"`
	DefaultAvatar string `json:"defaultAvatar"`
	MasterPassword string `json:"masterPassword"`
	Tags         []string `json:"tags"`
}

// Application 应用结构
type Application struct {
	Owner            string   `json:"owner"`
	Name             string   `json:"name"`
	CreatedTime      string   `json:"createdTime"`
	DisplayName      string   `json:"displayName"`
	Logo             string   `json:"logo"`
	HomepageUrl      string   `json:"homepageUrl"`
	Organization     string   `json:"organization"`
	ClientId         string   `json:"clientId"`
	ClientSecret     string   `json:"clientSecret"`
	RedirectUris     []string `json:"redirectUris"`
	TokenFormat      string   `json:"tokenFormat"`
	ExpireInHours    int      `json:"expireInHours"`
	RefreshExpireInHours int  `json:"refreshExpireInHours"`
	SignupUrl        string   `json:"signupUrl"`
	SigninUrl        string   `json:"signinUrl"`
	ForgetUrl        string   `json:"forgetUrl"`
	AffiliationUrl   string   `json:"affiliationUrl"`
	TermsOfUse       string   `json:"termsOfUse"`
	SignupHtml       string   `json:"signupHtml"`
	SigninHtml       string   `json:"signinHtml"`
	Tags             []string `json:"tags"`
}

// Role 角色结构
type Role struct {
	Owner       string   `json:"owner"`
	Name        string   `json:"name"`
	CreatedTime string   `json:"createdTime"`
	DisplayName string   `json:"displayName"`
	Description string   `json:"description"`
	Users       []string `json:"users"`
	Roles       []string `json:"roles"`
	IsEnabled   bool     `json:"isEnabled"`
}

// Permission 权限结构
type Permission struct {
	Owner       string   `json:"owner"`
	Name        string   `json:"name"`
	CreatedTime string   `json:"createdTime"`
	DisplayName string   `json:"displayName"`
	Description string   `json:"description"`
	Users       []string `json:"users"`
	Roles       []string `json:"roles"`
	ResourceType string  `json:"resourceType"`
	Resources   []string `json:"resources"`
	Actions     []string `json:"actions"`
	Effect      string   `json:"effect"`
	IsEnabled   bool     `json:"isEnabled"`
}

// CasdoorClient Casdoor 客户端
type CasdoorClient struct {
	endpoint string
	token    string
}

// NewCasdoorClient 创建 Casdoor 客户端
func NewCasdoorClient(endpoint string) *CasdoorClient {
	return &CasdoorClient{
		endpoint: endpoint,
	}
}

// Login 登录获取 Token
func (c *CasdoorClient) Login(username, password string) error {
	loginURL := fmt.Sprintf("%s/api/login", c.endpoint)
	
	payload := map[string]interface{}{
		"application": "app-built-in",
		"organization": "built-in",
		"username": username,
		"password": password,
		"autoSignin": false,
	}
	
	data, _ := json.Marshal(payload)
	resp, err := http.Post(loginURL, "application/json", strings.NewReader(string(data)))
	if err != nil {
		return fmt.Errorf("登录失败: %w", err)
	}
	defer resp.Body.Close()
	
	body, _ := ioutil.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	
	if result["status"] == "ok" {
		if data, ok := result["data"].(map[string]interface{}); ok {
			if token, ok := data["accessToken"].(string); ok {
				c.token = token
				fmt.Println("✅ Casdoor 登录成功")
				return nil
			}
		}
	}
	
	return fmt.Errorf("登录失败: %s", string(body))
}

// CreateOrganization 创建组织
func (c *CasdoorClient) CreateOrganization() error {
	fmt.Println("\n📋 创建组织...")
	
	org := Organization{
		Owner:       "admin",
		Name:        organizationName,
		CreatedTime: time.Now().Format(time.RFC3339),
		DisplayName: organizationDisplayName,
		WebsiteUrl:  homeURL,
		PasswordType: "plain",
		PhonePrefix:  "86",
		DefaultAvatar: "https://cdn.casbin.org/img/casbin.svg",
		Tags:        []string{},
	}
	
	data, _ := json.Marshal(org)
	url := fmt.Sprintf("%s/api/add-organization", c.endpoint)
	
	req, _ := http.NewRequest("POST", url, strings.NewReader(string(data)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("创建组织失败: %w", err)
	}
	defer resp.Body.Close()
	
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("   组织创建响应: %s\n", string(body))
	fmt.Println("✅ 组织创建成功:", organizationDisplayName)
	
	return nil
}

// CreateApplication 创建应用
func (c *CasdoorClient) CreateApplication() (*Application, error) {
	fmt.Println("\n📱 创建应用...")
	
	// 生成 Client ID 和 Secret
	clientID := fmt.Sprintf("%s_%d", applicationName, time.Now().Unix())
	clientSecret := generateSecret(32)
	
	app := Application{
		Owner:       organizationName,
		Name:        applicationName,
		CreatedTime: time.Now().Format(time.RFC3339),
		DisplayName: applicationDisplayName,
		Logo:        "https://cdn.casbin.org/img/casbin.svg",
		HomepageUrl: homeURL,
		Organization: organizationName,
		ClientId:    clientID,
		ClientSecret: clientSecret,
		RedirectUris: []string{callbackURL},
		TokenFormat: "JWT",
		ExpireInHours: 24,
		RefreshExpireInHours: 168,
		Tags:        []string{},
	}
	
	data, _ := json.Marshal(app)
	url := fmt.Sprintf("%s/api/add-application", c.endpoint)
	
	req, _ := http.NewRequest("POST", url, strings.NewReader(string(data)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("创建应用失败: %w", err)
	}
	defer resp.Body.Close()
	
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("   应用创建响应: %s\n", string(body))
	fmt.Println("✅ 应用创建成功:", applicationDisplayName)
	
	return &app, nil
}

// CreateRoles 创建角色
func (c *CasdoorClient) CreateRoles() error {
	fmt.Println("\n👥 创建角色...")
	
	roles := []struct {
		Name        string
		DisplayName string
		Description string
	}{
		{"admin", "管理员", "系统管理员，拥有所有权限"},
		{"operator", "运维人员", "可以管理服务组、服务和实例"},
		{"viewer", "访客", "只能查看，无法修改"},
	}
	
	for _, r := range roles {
		role := Role{
			Owner:       organizationName,
			Name:        r.Name,
			CreatedTime: time.Now().Format(time.RFC3339),
			DisplayName: r.DisplayName,
			Description: r.Description,
			Users:       []string{},
			Roles:       []string{},
			IsEnabled:   true,
		}
		
		data, _ := json.Marshal(role)
		url := fmt.Sprintf("%s/api/add-role", c.endpoint)
		
		req, _ := http.NewRequest("POST", url, strings.NewReader(string(data)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.token)
		
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("   ⚠️  创建角色 %s 失败: %v\n", r.Name, err)
			continue
		}
		defer resp.Body.Close()
		
		fmt.Printf("   ✅ 角色创建成功: %s (%s)\n", r.DisplayName, r.Name)
	}
	
	return nil
}

// CreatePermissions 创建权限
func (c *CasdoorClient) CreatePermissions() error {
	fmt.Println("\n🔐 创建权限...")
	
	permissions := []struct {
		Name         string
		DisplayName  string
		Description  string
		ResourceType string
		Resources    []string
		Actions      []string
		Roles        []string
	}{
		{
			Name:         "consul_group_all",
			DisplayName:  "服务组管理权限",
			Description:  "服务组的所有操作权限",
			ResourceType: "consul_group",
			Resources:    []string{"*"},
			Actions:      []string{"read", "write", "delete", "admin"},
			Roles:        []string{"admin", "operator"},
		},
		{
			Name:         "consul_service_all",
			DisplayName:  "服务管理权限",
			Description:  "服务的所有操作权限",
			ResourceType: "consul_service",
			Resources:    []string{"*"},
			Actions:      []string{"read", "write", "delete", "admin"},
			Roles:        []string{"admin", "operator"},
		},
		{
			Name:         "consul_instance_all",
			DisplayName:  "实例管理权限",
			Description:  "实例的所有操作权限",
			ResourceType: "consul_instance",
			Resources:    []string{"*"},
			Actions:      []string{"read", "write", "delete", "import", "export", "admin"},
			Roles:        []string{"admin", "operator"},
		},
		{
			Name:         "audit_log_read",
			DisplayName:  "审计日志查看权限",
			Description:  "查看审计日志",
			ResourceType: "audit_log",
			Resources:    []string{"*"},
			Actions:      []string{"read", "export"},
			Roles:        []string{"admin", "operator"},
		},
		{
			Name:         "consul_read_only",
			DisplayName:  "只读权限",
			Description:  "只能查看，不能修改",
			ResourceType: "consul",
			Resources:    []string{"*"},
			Actions:      []string{"read"},
			Roles:        []string{"viewer"},
		},
	}
	
	for _, p := range permissions {
		perm := Permission{
			Owner:        organizationName,
			Name:         p.Name,
			CreatedTime:  time.Now().Format(time.RFC3339),
			DisplayName:  p.DisplayName,
			Description:  p.Description,
			ResourceType: p.ResourceType,
			Resources:    p.Resources,
			Actions:      p.Actions,
			Roles:        p.Roles,
			Effect:       "Allow",
			IsEnabled:    true,
		}
		
		data, _ := json.Marshal(perm)
		url := fmt.Sprintf("%s/api/add-permission", c.endpoint)
		
		req, _ := http.NewRequest("POST", url, strings.NewReader(string(data)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.token)
		
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("   ⚠️  创建权限 %s 失败: %v\n", p.Name, err)
			continue
		}
		defer resp.Body.Close()
		
		fmt.Printf("   ✅ 权限创建成功: %s\n", p.DisplayName)
	}
	
	return nil
}

// generateSecret 生成随机密钥
func generateSecret(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(result)
}

// SaveConfig 保存配置到文件
func SaveConfig(config *CasdoorConfig) error {
	fmt.Println("\n💾 保存配置...")
	
	// 保存到 JSON 文件
	data, _ := json.MarshalIndent(config, "", "  ")
	if err := ioutil.WriteFile("casdoor_config.json", data, 0644); err != nil {
		return err
	}
	
	// 生成 .env 文件
	envContent := fmt.Sprintf(`# Casdoor 配置（自动生成）
CASDOOR_ENDPOINT=%s
CASDOOR_CLIENT_ID=%s
CASDOOR_CLIENT_SECRET=%s
CASDOOR_ORGANIZATION=%s
CASDOOR_APPLICATION=%s
`, config.Endpoint, config.ClientID, config.ClientSecret, config.OrganizationName, config.ApplicationName)
	
	if err := ioutil.WriteFile(".env.casdoor", []byte(envContent), 0644); err != nil {
		return err
	}
	
	fmt.Println("   ✅ 配置已保存到 casdoor_config.json")
	fmt.Println("   ✅ 环境变量已保存到 .env.casdoor")
	
	return nil
}

// PrintSummary 打印配置摘要
func PrintSummary(config *CasdoorConfig) {
	fmt.Println("\n" + strings.Repeat("━", 70))
	fmt.Println("🎉 Casdoor 配置完成！")
	fmt.Println(strings.Repeat("━", 70))
	fmt.Println("\n📋 配置信息：")
	fmt.Printf("   Endpoint:         %s\n", config.Endpoint)
	fmt.Printf("   Client ID:        %s\n", config.ClientID)
	fmt.Printf("   Client Secret:    %s\n", config.ClientSecret)
	fmt.Printf("   Organization:     %s\n", config.OrganizationName)
	fmt.Printf("   Application:      %s\n", config.ApplicationName)
	
	fmt.Println("\n🔗 访问地址：")
	fmt.Printf("   Casdoor 管理后台:  %s\n", config.Endpoint)
	fmt.Printf("   Consul Manager:   %s\n", homeURL)
	
	fmt.Println("\n👥 默认角色：")
	fmt.Println("   - admin (管理员) - 所有权限")
	fmt.Println("   - operator (运维) - 服务管理权限")
	fmt.Println("   - viewer (访客) - 只读权限")
	
	fmt.Println("\n📝 下一步：")
	fmt.Println("   1. 将 .env.casdoor 中的配置添加到系统环境变量")
	fmt.Println("   2. 重启 Consul Manager 服务")
	fmt.Println("   3. 访问 http://localhost:8080 开始使用")
	
	fmt.Println("\n" + strings.Repeat("━", 70))
}

func main() {
	fmt.Println("🚀 Casdoor 自动配置工具")
	fmt.Println(strings.Repeat("━", 70))
	
	// 获取 Casdoor 地址
	endpoint := os.Getenv("CASDOOR_ENDPOINT")
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	
	fmt.Printf("\n📡 连接到 Casdoor: %s\n", endpoint)
	
	// 创建客户端
	client := NewCasdoorClient(endpoint)
	
	// 登录
	if err := client.Login(defaultAdminUser, defaultAdminPassword); err != nil {
		fmt.Printf("❌ 错误: %v\n", err)
		fmt.Println("\n💡 提示：")
		fmt.Println("   1. 确保 Casdoor 服务正在运行")
		fmt.Println("   2. 检查 Casdoor 地址是否正确")
		fmt.Println("   3. 确认默认管理员账号: admin / 123")
		os.Exit(1)
	}
	
	// 创建组织
	if err := client.CreateOrganization(); err != nil {
		fmt.Printf("⚠️  警告: %v (可能已存在)\n", err)
	}
	
	// 创建应用
	app, err := client.CreateApplication()
	if err != nil {
		fmt.Printf("❌ 错误: %v\n", err)
		os.Exit(1)
	}
	
	// 创建角色
	if err := client.CreateRoles(); err != nil {
		fmt.Printf("⚠️  警告: %v\n", err)
	}
	
	// 创建权限
	if err := client.CreatePermissions(); err != nil {
		fmt.Printf("⚠️  警告: %v\n", err)
	}
	
	// 保存配置
	config := &CasdoorConfig{
		Endpoint:         endpoint,
		ClientID:         app.ClientId,
		ClientSecret:     app.ClientSecret,
		OrganizationName: organizationName,
		ApplicationName:  applicationName,
	}
	
	if err := SaveConfig(config); err != nil {
		fmt.Printf("❌ 保存配置失败: %v\n", err)
		os.Exit(1)
	}
	
	// 打印摘要
	PrintSummary(config)
}
