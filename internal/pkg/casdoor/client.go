// Package casdoor 提供 Casdoor 客户端封装和认证功能
package casdoor

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
	"github.com/golang-jwt/jwt/v5"
)

// 令牌校验缓存参数。
// Casdoor 的 get-account 会校验令牌存在于数据库、未过期、用户未被禁用/删除，
// 是权威校验；加短 TTL 缓存避免每个请求都回源。
const (
	tokenCacheTTL      = 60 * time.Second
	tokenValidateLimit = 8 * time.Second
)

// tokenCacheEntry 已校验令牌的缓存项
type tokenCacheEntry struct {
	user      *UserInfo
	expiresAt time.Time
}

// Client Casdoor 客户端封装
type Client struct {
	config *Config
	sdk    *casdoorsdk.Client

	// tokenCache 缓存已通过 Casdoor 权威校验的令牌（键为令牌 SHA256）
	tokenMu    sync.RWMutex
	tokenCache map[string]tokenCacheEntry
}

// NewClient 创建 Casdoor 客户端
//
// 参数:
//
//	config - Casdoor 配置信息
//
// 返回:
//
//	*Client - Casdoor 客户端实例
//	error - 错误信息
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

	// 为 SDK 注入带超时的 HTTP 客户端（SDK 默认使用无超时的 &http.Client{}，
	// Casdoor/反向代理 hang 住会导致请求协程与连接堆积）
	casdoorsdk.SetHttpClient(&http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 20,
			IdleConnTimeout:     90 * time.Second,
		},
	})

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
		config:     config,
		sdk:        sdk,
		tokenCache: make(map[string]tokenCacheEntry),
	}, nil
}

// ValidateToken 权威校验访问令牌并返回当前用户。
//
// 安全说明（重要）：
// 令牌来自浏览器，**不可直接信任其载荷**。此方法先做本地快速失败（结构/过期），
// 再调用 Casdoor 的 /api/get-account 做权威校验：Casdoor 会校验令牌存在于数据库、
// 未过期、且用户未被禁用或删除，并返回最新的 isAdmin / isForbidden。
//
// 校验结果按令牌哈希缓存 tokenCacheTTL，避免每请求回源。
func (c *Client) ValidateToken(token string) (*UserInfo, error) {
	if token == "" {
		return nil, fmt.Errorf("令牌为空")
	}

	key := tokenCacheKey(token)
	if u, ok := c.getCachedToken(key); ok {
		return u, nil
	}

	// 本地快速失败：解析载荷检查 exp（不验签，仅用于提前拒绝明显过期的令牌）
	if exp, ok := tokenExpiry(token); ok && exp > 0 && time.Now().Unix() >= exp {
		return nil, fmt.Errorf("令牌已过期")
	}

	// 权威校验：回源 Casdoor
	ctx, cancel := context.WithTimeout(context.Background(), tokenValidateLimit)
	defer cancel()

	type result struct {
		user *casdoorsdk.User
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		// panic 兜底：goroutine 内 panic 会终止整个进程
		defer func() {
			if r := recover(); r != nil {
				ch <- result{err: fmt.Errorf("校验令牌 panic: %v", r)}
			}
		}()
		u, err := c.sdk.WithAccessToken(token).GetAccount()
		ch <- result{user: u, err: err}
	}()

	var res result
	select {
	case res = <-ch:
	case <-ctx.Done():
		return nil, fmt.Errorf("校验令牌超时")
	}
	if res.err != nil {
		return nil, fmt.Errorf("令牌无效: %w", res.err)
	}
	if res.user == nil {
		return nil, fmt.Errorf("令牌无效：未获取到用户")
	}
	if res.user.IsForbidden || res.user.IsDeleted {
		return nil, fmt.Errorf("账号已被禁用")
	}

	user := &UserInfo{
		Owner:       res.user.Owner,
		Name:        res.user.Name,
		Id:          res.user.Id,
		DisplayName: res.user.DisplayName,
		Email:       res.user.Email,
		Phone:       res.user.Phone,
		Avatar:      res.user.Avatar,
		IsAdmin:     res.user.IsAdmin,
		// 业务约定：FlyIAM 组织（owner=flyiam）的管理员即本系统管理员，
		// 故 IsGlobalAdmin 与 IsAdmin 等价（与历史行为一致）。
		IsGlobalAdmin:     res.user.IsAdmin,
		IsForbidden:       res.user.IsForbidden,
		IsDeleted:         res.user.IsDeleted,
		SignupApplication: res.user.SignupApplication,
		Properties:        res.user.Properties,
	}
	for _, r := range res.user.Roles {
		user.Roles = append(user.Roles, &Role{
			Owner:       r.Owner,
			Name:        r.Name,
			CreatedTime: r.CreatedTime,
			DisplayName: r.DisplayName,
			Description: r.Description,
			IsEnabled:   r.IsEnabled,
		})
	}
	c.setCachedToken(key, user)
	return user, nil
}

// tokenCacheKey 令牌哈希（避免在内存中留存原始令牌）
func tokenCacheKey(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (c *Client) getCachedToken(key string) (*UserInfo, bool) {
	c.tokenMu.RLock()
	entry, ok := c.tokenCache[key]
	c.tokenMu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.tokenMu.Lock()
		delete(c.tokenCache, key)
		c.tokenMu.Unlock()
		return nil, false
	}
	return entry.user, true
}

func (c *Client) setCachedToken(key string, user *UserInfo) {
	c.tokenMu.Lock()
	// 简单容量保护：超过 10000 条时清空，避免无界增长
	if len(c.tokenCache) > 10000 {
		c.tokenCache = make(map[string]tokenCacheEntry)
	}
	c.tokenCache[key] = tokenCacheEntry{user: user, expiresAt: time.Now().Add(tokenCacheTTL)}
	c.tokenMu.Unlock()
}

// tokenExpiry 解析载荷中的 exp（不验签，仅用于本地快速失败）
func tokenExpiry(token string) (int64, bool) {
	parsed, _, err := jwt.NewParser().ParseUnverified(token, jwt.MapClaims{})
	if err != nil || parsed == nil {
		return 0, false
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return 0, false
	}
	if v, ok := claims["exp"].(float64); ok {
		return int64(v), true
	}
	return 0, false
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
//
//	redirectUri - 回调地址
//
// 返回:
//
//	string - 登录 URL
func (c *Client) GetSigninUrl(redirectUri, state string) string {
	base := strings.TrimRight(c.browserEndpoint(), "/")
	return fmt.Sprintf(
		"%s/login/oauth/authorize?client_id=%s&response_type=code&redirect_uri=%s&scope=profile&state=%s",
		base,
		url.QueryEscape(c.config.ClientId),
		url.QueryEscape(redirectUri),
		url.QueryEscape(state),
	)
}

// GenState 生成随机 OAuth state（供登录处理器写入 Cookie 并在回调时校验）
func GenState() string { return genState() }

// GetSignupUrl 获取注册 URL
//
// 参数:
//
//	redirectUri - 回调地址
//
// 返回:
//
//	string - 注册 URL
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
//
//	code - 授权码
//
// 返回:
//
//	string - Access Token
//	error - 错误信息
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
//
//	token - JWT Token
//
// 返回:
//
//	*Claims - Token 声明信息
//	error - 错误信息
//
// 说明:
//
//	由于 Token 是后端直接从 Casdoor 换取的（可信渠道），
//	此处只需解码 JWT 载荷获取用户信息，无需再用证书验证签名。
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
//
//	username - 用户名
//
// 返回:
//
//	*UserInfo - 用户信息
//	error - 错误信息
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
//
//	username - 用户名
//
// 返回:
//
//	[]string - 角色名称列表
//	error - 错误信息
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
//
//	username - 用户名
//	resource - 资源类型（如: consul_group, consul_service）
//	action - 操作类型（如: read, write, delete）
//
// 返回:
//
//	bool - 是否有权限
//	error - 错误信息
//
// 权限检查逻辑：
//  1. 获取用户的所有权限
//  2. 遍历权限列表，匹配资源类型和操作
//  3. 支持通配符匹配（* 表示所有）
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
//
//	token - JWT Token
//	resource - 资源类型
//	action - 操作类型
//
// 返回:
//
//	bool - 是否有权限
//	error - 错误信息
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
//
//	refreshToken - 刷新令牌
//
// 返回:
//
//	string - 新的 Access Token
//	error - 错误信息
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
//
//	*Config - 配置信息
func (c *Client) GetConfig() *Config {
	return c.config
}

// CasdoorUser 用于列表展示的 Casdoor 用户信息
type CasdoorUser struct {
	Id          string            `json:"id"`
	Owner       string            `json:"owner"`
	Name        string            `json:"name"`
	DisplayName string            `json:"displayName"`
	Email       string            `json:"email"`
	Phone       string            `json:"phone"`
	Avatar      string            `json:"avatar"`
	IsAdmin     bool              `json:"isAdmin"`
	IsForbidden bool              `json:"isForbidden"`
	SignupApp   string            `json:"signupApplication"`
	CreatedTime string            `json:"createdTime"`
	Properties  map[string]string `json:"properties"`
}

// listUsersPageSize 分页遍历用户时的每页条数
const listUsersPageSize = 200

// ListUsers 获取 Casdoor 用户列表
//
// 说明：用于「人员组织 → 用户管理」展示；用户体系仍由 Casdoor 维护。
// 同时返回 Properties（自定义字段），供页面展示人事信息。
//
// 实现要点：
//   - 按页遍历而非一次性 c.sdk.GetUsers()。SDK 的无分页接口会把整个响应先
//     反序列化成泛型 interface{}（大组织时内存放大数倍）再二次序列化，
//     3 万+ 用户时峰值内存可达数百 MB，存在 OOM 风险；分页后每页内存有界。
//   - 按唯一键 name 排序，避免 Casdoor 默认按 created_time（可能相同）排序
//     导致 offset 分页在页间重叠或漏读；并按 owner/name 去重兜底。
func (c *Client) ListUsers() ([]CasdoorUser, error) {
	list := make([]CasdoorUser, 0, listUsersPageSize)
	seen := make(map[string]struct{}, listUsersPageSize)
	err := c.IterateUsers(func(users []*casdoorsdk.User) error {
		for _, u := range users {
			key := u.Owner + "/" + u.Name
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			props := u.Properties
			if props == nil {
				props = map[string]string{}
			}
			list = append(list, CasdoorUser{
				Id:          u.Id,
				Owner:       u.Owner,
				Name:        u.Name,
				DisplayName: u.DisplayName,
				Email:       u.Email,
				Phone:       u.Phone,
				Avatar:      u.Avatar,
				IsAdmin:     u.IsAdmin,
				IsForbidden: u.IsForbidden,
				SignupApp:   u.SignupApplication,
				CreatedTime: u.CreatedTime,
				Properties:  props,
			})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("获取用户列表失败: %w", err)
	}
	return list, nil
}

// IterateUsers 分页遍历 Casdoor 用户，逐页回调（内存峰值与页大小成正比）。
//
// 排序固定为 name 升序：owner+name 为唯一键，保证 offset 分页确定性，
// 避免默认 created_time 相同导致的页间重叠/漏读。
func (c *Client) IterateUsers(fn func([]*casdoorsdk.User) error) error {
	page := 1
	for {
		users, total, err := c.sdk.GetPaginationUsers(page, listUsersPageSize, map[string]string{
			"sortField": "name",
			"sortOrder": "ascend",
		})
		if err != nil {
			return err
		}
		if len(users) > 0 {
			if err := fn(users); err != nil {
				return err
			}
		}
		if !hasMoreUsersPage(page, listUsersPageSize, total, len(users)) {
			return nil
		}
		page++
	}
}

// hasMoreUsersPage 判断分页遍历是否应继续（本页无数据或已取满 total 则停止）。
//
// 抽出为纯函数便于单测，避免分页边界（total=0、末页不足、total 偏小）处理出错。
func hasMoreUsersPage(page, pageSize, total, got int) bool {
	if got == 0 {
		return false
	}
	return page*pageSize < total
}

// UserUpsert 用户新增/更新入参
type UserUpsert struct {
	Name        string            `json:"name"`        // 域账号（唯一，创建后不可改）
	DisplayName string            `json:"displayName"` // 姓名
	Avatar      string            `json:"avatar"`      // 头像 URL
	Email       string            `json:"email"`
	Phone       string            `json:"phone"`
	Password    string            `json:"password"`   // 仅创建时使用，留空用默认密码
	Properties  map[string]string `json:"properties"` // 自定义/人事字段
}

// CreateUser 在 Casdoor 创建用户
func (c *Client) CreateUser(in UserUpsert, defaultPassword string) error {
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("域账号不能为空")
	}
	password := in.Password
	if password == "" {
		password = defaultPassword
	}
	props := in.Properties
	if props == nil {
		props = map[string]string{}
	}
	user := &casdoorsdk.User{
		Owner:             c.config.OrganizationName,
		Name:              in.Name,
		DisplayName:       in.DisplayName,
		Avatar:            in.Avatar,
		Email:             in.Email,
		Phone:             in.Phone,
		Password:          password,
		Type:              "normal-user",
		Language:          "zh",
		Tag:               "manual",
		Address:           []string{},
		Properties:        props,
		SignupApplication: c.config.ApplicationName,
	}
	if _, err := c.sdk.AddUser(user); err != nil {
		return fmt.Errorf("创建用户失败: %w", err)
	}
	return nil
}

// UpdateUser 更新用户基础信息与属性（域账号不可改）
func (c *Client) UpdateUser(in UserUpsert) error {
	user, err := c.sdk.GetUser(in.Name)
	if err != nil || user == nil {
		return fmt.Errorf("用户不存在: %s", in.Name)
	}
	if in.DisplayName != "" {
		user.DisplayName = in.DisplayName
	}
	user.Avatar = in.Avatar
	user.Email = in.Email
	user.Phone = in.Phone
	if user.Properties == nil {
		user.Properties = map[string]string{}
	}
	for k, v := range in.Properties {
		if k == "" {
			continue
		}
		user.Properties[k] = v
	}
	if _, err := c.sdk.UpdateUser(user); err != nil {
		return fmt.Errorf("更新用户失败: %w", err)
	}
	return nil
}

// DeleteUser 删除用户
func (c *Client) DeleteUser(name string) error {
	user := &casdoorsdk.User{
		Owner: c.config.OrganizationName,
		Name:  name,
	}
	if _, err := c.sdk.DeleteUser(user); err != nil {
		return fmt.Errorf("删除用户失败: %w", err)
	}
	return nil
}

// ResetPassword 重置用户密码（oldPassword 为空表示管理员重置）
func (c *Client) ResetPassword(name, newPassword string) error {
	if _, err := c.sdk.SetPassword(c.config.OrganizationName, name, "", newPassword); err != nil {
		return fmt.Errorf("重置密码失败: %w", err)
	}
	return nil
}

// SetUserAdmin 设置/取消用户管理员标记
func (c *Client) SetUserAdmin(name string, isAdmin bool) error {
	user, err := c.sdk.GetUser(name)
	if err != nil || user == nil {
		return fmt.Errorf("用户不存在: %s", name)
	}
	user.IsAdmin = isAdmin
	if _, err := c.sdk.UpdateUser(user); err != nil {
		return fmt.Errorf("更新管理员标记失败: %w", err)
	}
	return nil
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

// ProbeAPI 校验 endpoint 指向的是否为可用的 Casdoor API。
//
// /api/health 应返回 JSON（{"status":"ok"}）。若返回 HTML（例如端点误指向
// 前端页面、Ingress 首页、或是其他 Web 服务），后续 SDK 调用会报
// "invalid character '<' looking for beginning of value"，此处提前给出明确错误。
func ProbeAPI(endpoint string) error {
	if endpoint == "" {
		return fmt.Errorf("Casdoor 端点未配置（请设置 CASDOOR_ENDPOINT）")
	}
	url := strings.TrimRight(endpoint, "/") + "/api/health"
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("无法访问 Casdoor 端点 %s: %w（请确认地址可达、Service/DNS 正确）", url, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Casdoor 端点 %s 返回 HTTP %d，响应片段: %s", url, resp.StatusCode, bodySnippet(body))
	}

	var probe map[string]any
	if err := json.Unmarshal(body, &probe); err != nil {
		return fmt.Errorf("Casdoor 端点 %s 返回的不是 JSON（疑似误指向前端页面/其他服务，而非 Casdoor API），响应片段: %s",
			url, bodySnippet(body))
	}
	return nil
}

// bodySnippet 截断响应体用于错误提示
func bodySnippet(b []byte) string {
	s := strings.Join(strings.Fields(string(b)), " ")
	if s == "" {
		return "(空)"
	}
	if len(s) > 200 {
		return s[:200] + "..."
	}
	return s
}
