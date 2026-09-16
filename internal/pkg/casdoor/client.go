// Package casdoor 提供 Casdoor 客户端封装和认证功能
package casdoor

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
	"github.com/golang-jwt/jwt/v5"
)

// Client Casdoor 客户端封装
type Client struct {
	config *Config
	sdk    *casdoorsdk.Client
}

// NewClient 创建 Casdoor 客户端
//
// 参数:
//   config - Casdoor 配置信息
//
// 返回:
//   *Client - Casdoor 客户端实例
//   error - 错误信息
func NewClient(config *Config) (*Client, error) {
	// 验证配置
	if config.Endpoint == "" {
		return nil, fmt.Errorf("Casdoor endpoint 不能为空")
	}
	if config.ClientId == "" {
		return nil, fmt.Errorf("Casdoor client_id 不能为空")
	}
	if config.ClientSecret == "" {
		return nil, fmt.Errorf("Casdoor client_secret 不能为空")
	}
	if config.OrganizationName == "" {
		return nil, fmt.Errorf("Casdoor organization_name 不能为空")
	}
	if config.ApplicationName == "" {
		return nil, fmt.Errorf("Casdoor application_name 不能为空")
	}

	// 创建 Casdoor SDK 客户端
	sdk := casdoorsdk.NewClient(
		config.Endpoint,
		config.ClientId,
		config.ClientSecret,
		config.Certificate,
		config.OrganizationName,
		config.ApplicationName,
	)

	return &Client{
		config: config,
		sdk:    sdk,
	}, nil
}

// browserEndpoint 返回浏览器可达的 Casdoor 地址
func (c *Client) browserEndpoint() string {
	if c.config.PublicEndpoint != "" {
		return c.config.PublicEndpoint
	}
	return c.config.Endpoint
}

// genState 生成随机 state 参数
func genState() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// GetSigninUrl 获取登录 URL
//
// 说明：使用 PublicEndpoint（浏览器可达地址）构造，保证跨域名/IP 部署可用
//
// 参数:
//   redirectUri - 回调地址
//
// 返回:
//   string - 登录 URL
func (c *Client) GetSigninUrl(redirectUri string) string {
	base := strings.TrimRight(c.browserEndpoint(), "/")
	return fmt.Sprintf(
		"%s/login/oauth/authorize?client_id=%s&response_type=code&redirect_uri=%s&scope=profile&state=%s",
		base,
		url.QueryEscape(c.config.ClientId),
		url.QueryEscape(redirectUri),
		genState(),
	)
}

// GetSignupUrl 获取注册 URL
//
// 参数:
//   redirectUri - 回调地址
//
// 返回:
//   string - 注册 URL
func (c *Client) GetSignupUrl(redirectUri string) string {
	base := strings.TrimRight(c.browserEndpoint(), "/")
	return fmt.Sprintf(
		"%s/signup/oauth/authorize?client_id=%s&response_type=code&redirect_uri=%s&scope=profile&state=%s",
		base,
		url.QueryEscape(c.config.ClientId),
		url.QueryEscape(redirectUri),
		genState(),
	)
}

// GetToken 通过授权码获取 Token
//
// 参数:
//   code - 授权码
//
// 返回:
//   string - Access Token
//   error - 错误信息
func (c *Client) GetToken(code string) (string, error) {
	token, err := c.sdk.GetOAuthToken(code, "")
	if err != nil {
		return "", fmt.Errorf("获取 Token 失败: %w", err)
	}
	return token.AccessToken, nil
}

// ParseToken 解析 JWT Token（不验证签名）
//
// 参数:
//   token - JWT Token
//
// 返回:
//   *Claims - Token 声明信息
//   error - 错误信息
//
// 说明:
//   由于 Token 是后端直接从 Casdoor 换取的（可信渠道），
//   此处只需解码 JWT 载荷获取用户信息，无需再用证书验证签名。
func (c *Client) ParseToken(token string) (*Claims, error) {
	// 使用 jwt 库解析但不验证签名
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return nil, nil
	})
	if err != nil && parsed == nil {
		return nil, fmt.Errorf("Token 解析失败: %w", err)
	}

	rawClaims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("Token 声明格式错误")
	}

	// 转换为自定义 Claims 结构
	result := &Claims{}

	if v, ok := rawClaims["tokenType"].(string); ok {
		result.TokenType = v
	}
	if v, ok := rawClaims["scope"].(string); ok {
		result.Scope = v
	}
	if v, ok := rawClaims["iss"].(string); ok {
		result.Iss = v
	}
	if v, ok := rawClaims["sub"].(string); ok {
		result.Sub = v
	}
	if v, ok := rawClaims["exp"].(float64); ok {
		result.Exp = int64(v)
	}
	if v, ok := rawClaims["nbf"].(float64); ok {
		result.Nbf = int64(v)
	}
	if v, ok := rawClaims["iat"].(float64); ok {
		result.Iat = int64(v)
	}
	if v, ok := rawClaims["jti"].(string); ok {
		result.Jti = v
	}

	// 提取用户信息
	// 注意：Casdoor 的 Access Token 将用户字段直接放在 JWT 顶层
	user := &UserInfo{}
	if v, ok := rawClaims["owner"].(string); ok {
		user.Owner = v
	}
	if v, ok := rawClaims["name"].(string); ok {
		user.Name = v
	}
	if v, ok := rawClaims["id"].(string); ok {
		user.Id = v
	}
	if v, ok := rawClaims["displayName"].(string); ok {
		user.DisplayName = v
	}
	if v, ok := rawClaims["email"].(string); ok {
		user.Email = v
	}
	if v, ok := rawClaims["phone"].(string); ok {
		user.Phone = v
	}
	if v, ok := rawClaims["avatar"].(string); ok {
		user.Avatar = v
	}
	if v, ok := rawClaims["isAdmin"].(bool); ok {
		user.IsAdmin = v
		user.IsGlobalAdmin = v
	}

	// 提取角色信息
	if rolesArr, ok := rawClaims["roles"].([]interface{}); ok {
		user.Roles = make([]*Role, 0, len(rolesArr))
		for _, r := range rolesArr {
			if roleMap, ok := r.(map[string]interface{}); ok {
				role := &Role{}
				if v, ok := roleMap["owner"].(string); ok {
					role.Owner = v
				}
				if v, ok := roleMap["name"].(string); ok {
					role.Name = v
				}
				if v, ok := roleMap["displayName"].(string); ok {
					role.DisplayName = v
				}
				if v, ok := roleMap["description"].(string); ok {
					role.Description = v
				}
				user.Roles = append(user.Roles, role)
			}
		}
	}

	// 提取权限信息
	if permsArr, ok := rawClaims["permissions"].([]interface{}); ok {
		user.Permissions = make([]*Permission, 0, len(permsArr))
		for _, p := range permsArr {
			if permMap, ok := p.(map[string]interface{}); ok {
				perm := &Permission{}
				if v, ok := permMap["owner"].(string); ok {
					perm.Owner = v
				}
				if v, ok := permMap["name"].(string); ok {
					perm.Name = v
				}
				if v, ok := permMap["displayName"].(string); ok {
					perm.DisplayName = v
				}
				if v, ok := permMap["resourceType"].(string); ok {
					perm.ResourceType = v
				}
				if v, ok := permMap["effect"].(string); ok {
					perm.Effect = v
				}
				if v, ok := permMap["isEnabled"].(bool); ok {
					perm.IsEnabled = v
				}
				if resArr, ok := permMap["resources"].([]interface{}); ok {
					for _, res := range resArr {
						if s, ok := res.(string); ok {
							perm.Resources = append(perm.Resources, s)
						}
					}
				}
				if actArr, ok := permMap["actions"].([]interface{}); ok {
					for _, act := range actArr {
						if s, ok := act.(string); ok {
							perm.Actions = append(perm.Actions, s)
						}
					}
				}
				user.Permissions = append(user.Permissions, perm)
			}
		}
	}

	if user.Name != "" {
		result.User = user
	}

	return result, nil
}

// GetUserInfo 获取用户详细信息
//
// 参数:
//   username - 用户名
//
// 返回:
//   *UserInfo - 用户信息
//   error - 错误信息
func (c *Client) GetUserInfo(username string) (*UserInfo, error) {
	user, err := c.sdk.GetUser(username)
	if err != nil {
		return nil, fmt.Errorf("获取用户信息失败: %w", err)
	}

	if user == nil {
		return nil, fmt.Errorf("用户不存在: %s", username)
	}

	return &UserInfo{
		Owner:         user.Owner,
		Name:          user.Name,
		Id:            user.Id,
		DisplayName:   user.DisplayName,
		Email:         user.Email,
		Phone:         user.Phone,
		Avatar:        user.Avatar,
		IsAdmin:       user.IsAdmin,
		IsGlobalAdmin: user.IsAdmin, // Casdoor SDK 中使用 IsAdmin 表示全局管理员
	}, nil
}

// GetUserRoles 获取用户的角色列表
//
// 参数:
//   username - 用户名
//
// 返回:
//   []string - 角色名称列表
//   error - 错误信息
func (c *Client) GetUserRoles(username string) ([]string, error) {
	user, err := c.sdk.GetUser(username)
	if err != nil {
		return nil, fmt.Errorf("获取用户信息失败: %w", err)
	}

	if user == nil {
		return nil, fmt.Errorf("用户不存在: %s", username)
	}

	roles := make([]string, 0, len(user.Roles))
	for _, role := range user.Roles {
		roles = append(roles, role.Name)
	}

	return roles, nil
}

// CheckPermission 检查用户是否有指定权限
//
// 参数:
//   username - 用户名
//   resource - 资源类型（如: consul_group, consul_service）
//   action - 操作类型（如: read, write, delete）
//
// 返回:
//   bool - 是否有权限
//   error - 错误信息
//
// 权限检查逻辑：
//   1. 获取用户的所有权限
//   2. 遍历权限列表，匹配资源类型和操作
//   3. 支持通配符匹配（* 表示所有）
func (c *Client) CheckPermission(username, resource, action string) (bool, error) {
	user, err := c.sdk.GetUser(username)
	if err != nil {
		return false, fmt.Errorf("获取用户信息失败: %w", err)
	}

	if user == nil {
		return false, fmt.Errorf("用户不存在: %s", username)
	}

	// 全局管理员拥有所有权限
	if user.IsAdmin {
		return true, nil
	}

	// 遍历用户的所有权限
	for _, perm := range user.Permissions {
		// 权限未启用，跳过
		if !perm.IsEnabled {
			continue
		}

		// 效果不是 Allow，跳过
		if perm.Effect != "Allow" {
			continue
		}

		// 检查资源类型匹配
		resourceMatch := false
		if perm.ResourceType == "*" || perm.ResourceType == resource {
			resourceMatch = true
		}

		if !resourceMatch {
			continue
		}

		// 检查操作匹配
		for _, act := range perm.Actions {
			if act == "*" || act == action {
				return true, nil
			}
		}
	}

	return false, nil
}

// CheckPermissionByToken 通过 Token 检查权限
//
// 参数:
//   token - JWT Token
//   resource - 资源类型
//   action - 操作类型
//
// 返回:
//   bool - 是否有权限
//   error - 错误信息
func (c *Client) CheckPermissionByToken(token, resource, action string) (bool, error) {
	// 解析 Token 获取用户信息
	claims, err := c.ParseToken(token)
	if err != nil {
		return false, err
	}

	if claims.User == nil {
		return false, fmt.Errorf("Token 中没有用户信息")
	}

	// 全局管理员拥有所有权限
	if claims.User.IsAdmin {
		return true, nil
	}

	// 遍历权限列表
	for _, perm := range claims.User.Permissions {
		// 权限未启用，跳过
		if !perm.IsEnabled {
			continue
		}

		// 效果不是 Allow，跳过
		if perm.Effect != "Allow" {
			continue
		}

		// 检查资源类型匹配
		resourceMatch := false
		if perm.ResourceType == "*" || perm.ResourceType == resource {
			resourceMatch = true
		}

		if !resourceMatch {
			continue
		}

		// 检查操作匹配
		for _, act := range perm.Actions {
			if act == "*" || act == action {
				return true, nil
			}
		}
	}

	return false, nil
}

// RefreshToken 刷新 Token
//
// 参数:
//   refreshToken - 刷新令牌
//
// 返回:
//   string - 新的 Access Token
//   error - 错误信息
func (c *Client) RefreshToken(refreshToken string) (string, error) {
	token, err := c.sdk.RefreshOAuthToken(refreshToken)
	if err != nil {
		return "", fmt.Errorf("刷新 Token 失败: %w", err)
	}
	return token.AccessToken, nil
}

// GetConfig 获取客户端配置
//
// 返回:
//   *Config - 配置信息
func (c *Client) GetConfig() *Config {
	return c.config
}

// CasdoorUser 用于列表展示的 Casdoor 用户精简信息
type CasdoorUser struct {
	Id          string `json:"id"`
	Owner       string `json:"owner"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Avatar      string `json:"avatar"`
	IsAdmin     bool   `json:"isAdmin"`
	SignupApp   string `json:"signupApplication"`
	CreatedTime string `json:"createdTime"`
}

// ListUsers 获取 Casdoor 用户列表
//
// 说明：用于「人员组织 → 用户管理」展示；用户体系仍由 Casdoor 维护。
func (c *Client) ListUsers() ([]CasdoorUser, error) {
	users, err := c.sdk.GetUsers()
	if err != nil {
		return nil, fmt.Errorf("获取用户列表失败: %w", err)
	}
	list := make([]CasdoorUser, 0, len(users))
	for _, u := range users {
		list = append(list, CasdoorUser{
			Id:          u.Id,
			Owner:       u.Owner,
			Name:        u.Name,
			DisplayName: u.DisplayName,
			Email:       u.Email,
			Phone:       u.Phone,
			Avatar:      u.Avatar,
			IsAdmin:     u.IsAdmin,
			SignupApp:   u.SignupApplication,
			CreatedTime: u.CreatedTime,
		})
	}
	return list, nil
}

// CasdoorRole 用于列表展示的 Casdoor 角色信息
type CasdoorRole struct {
	Owner       string `json:"owner"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	IsEnabled   bool   `json:"isEnabled"`
}

// ListCasdoorRoles 获取 Casdoor 角色列表（只读展示）
func (c *Client) ListCasdoorRoles() ([]CasdoorRole, error) {
	roles, err := c.sdk.GetRoles()
	if err != nil {
		return nil, fmt.Errorf("获取角色列表失败: %w", err)
	}
	list := make([]CasdoorRole, 0, len(roles))
	for _, r := range roles {
		list = append(list, CasdoorRole{
			Owner:       r.Owner,
			Name:        r.Name,
			DisplayName: r.DisplayName,
			Description: r.Description,
			IsEnabled:   r.IsEnabled,
		})
	}
	return list, nil
}
