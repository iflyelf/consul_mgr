// Package casdoor 提供 Casdoor 客户端封装和认证功能
//
// 功能包括：
//   - Casdoor 客户端初始化
//   - Token 验证和解析
//   - 用户信息获取
//   - 权限检查
//
// 作者: iflyelf
// 创建时间: 2024-01-15
package casdoor

// Config Casdoor 配置结构
type Config struct {
	Endpoint         string `json:"endpoint"`          // Casdoor 服务地址
	ClientId         string `json:"client_id"`         // 客户端 ID
	ClientSecret     string `json:"client_secret"`     // 客户端密钥
	Certificate      string `json:"certificate"`       // 证书（可选）
	OrganizationName string `json:"organization_name"` // 组织名称
	ApplicationName  string `json:"application_name"`  // 应用名称
}

// UserInfo Casdoor 用户信息
type UserInfo struct {
	Owner             string   `json:"owner"`              // 所属组织
	Name              string   `json:"name"`               // 用户名
	CreatedTime       string   `json:"createdTime"`        // 创建时间
	UpdatedTime       string   `json:"updatedTime"`        // 更新时间
	Id                string   `json:"id"`                 // 用户 ID
	Type              string   `json:"type"`               // 用户类型
	Password          string   `json:"password"`           // 密码（不应返回）
	PasswordSalt      string   `json:"passwordSalt"`       // 密码盐
	DisplayName       string   `json:"displayName"`        // 显示名称
	FirstName         string   `json:"firstName"`          // 名
	LastName          string   `json:"lastName"`           // 姓
	Avatar            string   `json:"avatar"`             // 头像
	PermanentAvatar   string   `json:"permanentAvatar"`    // 永久头像
	Email             string   `json:"email"`              // 邮箱
	EmailVerified     bool     `json:"emailVerified"`      // 邮箱已验证
	Phone             string   `json:"phone"`              // 电话
	Location          string   `json:"location"`           // 位置
	Address           []string `json:"address"`            // 地址
	Affiliation       string   `json:"affiliation"`        // 所属单位
	Title             string   `json:"title"`              // 职位
	IdCardType        string   `json:"idCardType"`         // 证件类型
	IdCard            string   `json:"idCard"`             // 证件号
	Homepage          string   `json:"homepage"`           // 主页
	Bio               string   `json:"bio"`                // 简介
	Tag               string   `json:"tag"`                // 标签
	Region            string   `json:"region"`             // 地区
	Language          string   `json:"language"`           // 语言
	Gender            string   `json:"gender"`             // 性别
	Birthday          string   `json:"birthday"`           // 生日
	Education         string   `json:"education"`          // 教育程度
	Score             int      `json:"score"`              // 积分
	Karma             int      `json:"karma"`              // 声望
	Ranking           int      `json:"ranking"`            // 排名
	IsDefaultAvatar   bool     `json:"isDefaultAvatar"`    // 是否默认头像
	IsOnline          bool     `json:"isOnline"`           // 是否在线
	IsAdmin           bool     `json:"isAdmin"`            // 是否管理员
	IsGlobalAdmin     bool     `json:"isGlobalAdmin"`      // 是否全局管理员
	IsForbidden       bool     `json:"isForbidden"`        // 是否禁用
	IsDeleted         bool     `json:"isDeleted"`          // 是否删除
	SignupApplication string   `json:"signupApplication"`  // 注册应用
	Hash              string   `json:"hash"`               // 哈希
	PreHash           string   `json:"preHash"`            // 预哈希
	CreatedIp         string   `json:"createdIp"`          // 创建 IP
	LastSigninTime    string   `json:"lastSigninTime"`     // 最后登录时间
	LastSigninIp      string   `json:"lastSigninIp"`       // 最后登录 IP
	GitHub            string   `json:"github"`             // GitHub
	Google            string   `json:"google"`             // Google
	QQ                string   `json:"qq"`                 // QQ
	WeChat            string   `json:"wechat"`             // 微信
	Facebook          string   `json:"facebook"`           // Facebook
	DingTalk          string   `json:"dingtalk"`           // 钉钉
	Weibo             string   `json:"weibo"`              // 微博
	Gitee             string   `json:"gitee"`              // Gitee
	LinkedIn          string   `json:"linkedin"`           // LinkedIn
	Wecom             string   `json:"wecom"`              // 企业微信
	Lark              string   `json:"lark"`               // 飞书
	Gitlab            string   `json:"gitlab"`             // GitLab
	Adfs              string   `json:"adfs"`               // ADFS
	Baidu             string   `json:"baidu"`              // 百度
	Alipay            string   `json:"alipay"`             // 支付宝
	Casdoor           string   `json:"casdoor"`            // Casdoor
	Infoflow          string   `json:"infoflow"`           // Infoflow
	Apple             string   `json:"apple"`              // Apple
	AzureAD           string   `json:"azuread"`            // Azure AD
	Slack             string   `json:"slack"`              // Slack
	Steam             string   `json:"steam"`              // Steam
	Bilibili          string   `json:"bilibili"`           // Bilibili
	Okta              string   `json:"okta"`               // Okta
	Douyin            string   `json:"douyin"`             // 抖音
	Line              string   `json:"line"`               // Line
	Amazon            string   `json:"amazon"`             // Amazon
	Auth0             string   `json:"auth0"`              // Auth0
	BattleNet         string   `json:"battlenet"`          // Battle.net
	Bitbucket         string   `json:"bitbucket"`          // Bitbucket
	Box               string   `json:"box"`                // Box
	CloudFoundry      string   `json:"cloudfoundry"`       // Cloud Foundry
	Dailymotion       string   `json:"dailymotion"`        // Dailymotion
	Deezer            string   `json:"deezer"`             // Deezer
	DigitalOcean      string   `json:"digitalocean"`       // DigitalOcean
	Discord           string   `json:"discord"`            // Discord
	Dropbox           string   `json:"dropbox"`            // Dropbox
	EveOnline         string   `json:"eveonline"`          // EVE Online
	Fitbit            string   `json:"fitbit"`             // Fitbit
	Gitea             string   `json:"gitea"`              // Gitea
	Heroku            string   `json:"heroku"`             // Heroku
	InfluxCloud       string   `json:"influxcloud"`        // InfluxCloud
	Instagram         string   `json:"instagram"`          // Instagram
	Intercom          string   `json:"intercom"`           // Intercom
	Kakao             string   `json:"kakao"`              // Kakao
	Lastfm            string   `json:"lastfm"`             // Last.fm
	Mailru            string   `json:"mailru"`             // Mail.ru
	Meetup            string   `json:"meetup"`             // Meetup
	MicrosoftOnline   string   `json:"microsoftonline"`    // Microsoft Online
	Naver             string   `json:"naver"`              // Naver
	Nextcloud         string   `json:"nextcloud"`          // Nextcloud
	OneDrive          string   `json:"onedrive"`           // OneDrive
	Oura              string   `json:"oura"`               // Oura
	Patreon           string   `json:"patreon"`            // Patreon
	Paypal            string   `json:"paypal"`             // PayPal
	SalesForce        string   `json:"salesforce"`         // Salesforce
	Shopify           string   `json:"shopify"`            // Shopify
	Soundcloud        string   `json:"soundcloud"`         // SoundCloud
	Spotify           string   `json:"spotify"`            // Spotify
	Strava            string   `json:"strava"`             // Strava
	Stripe            string   `json:"stripe"`             // Stripe
	TikTok            string   `json:"tiktok"`             // TikTok
	Tumblr            string   `json:"tumblr"`             // Tumblr
	Twitch            string   `json:"twitch"`             // Twitch
	Twitter           string   `json:"twitter"`            // Twitter
	Typetalk          string   `json:"typetalk"`           // Typetalk
	Uber              string   `json:"uber"`               // Uber
	VK                string   `json:"vk"`                 // VK
	Wepay             string   `json:"wepay"`              // WePay
	Xero              string   `json:"xero"`               // Xero
	Yahoo             string   `json:"yahoo"`              // Yahoo
	Yammer            string   `json:"yammer"`             // Yammer
	Yandex            string   `json:"yandex"`             // Yandex
	Zoom              string   `json:"zoom"`               // Zoom
	Properties        map[string]string `json:"properties"` // 自定义属性
	Roles             []*Role           `json:"roles"`      // 角色列表
	Permissions       []*Permission     `json:"permissions"` // 权限列表
}

// Role 角色信息
type Role struct {
	Owner       string `json:"owner"`       // 所属组织
	Name        string `json:"name"`        // 角色名称
	CreatedTime string `json:"createdTime"` // 创建时间
	DisplayName string `json:"displayName"` // 显示名称
	Description string `json:"description"` // 描述
	IsEnabled   bool   `json:"isEnabled"`   // 是否启用
}

// Permission 权限信息
type Permission struct {
	Owner        string   `json:"owner"`        // 所属组织
	Name         string   `json:"name"`         // 权限名称
	CreatedTime  string   `json:"createdTime"`  // 创建时间
	DisplayName  string   `json:"displayName"`  // 显示名称
	Description  string   `json:"description"`  // 描述
	ResourceType string   `json:"resourceType"` // 资源类型
	Resources    []string `json:"resources"`    // 资源列表
	Actions      []string `json:"actions"`      // 操作列表
	Effect       string   `json:"effect"`       // 效果（Allow/Deny）
	IsEnabled    bool     `json:"isEnabled"`    // 是否启用
}

// Claims JWT Token 声明
type Claims struct {
	User              *UserInfo `json:"user"`              // 用户信息
	TokenType         string    `json:"tokenType"`         // Token 类型
	Scope             string    `json:"scope"`             // 作用域
	Iss               string    `json:"iss"`               // 签发者
	Sub               string    `json:"sub"`               // 主题
	Aud               []string  `json:"aud"`               // 受众
	Exp               int64     `json:"exp"`               // 过期时间
	Nbf               int64     `json:"nbf"`               // 生效时间
	Iat               int64     `json:"iat"`               // 签发时间
	Jti               string    `json:"jti"`               // JWT ID
	AccessToken       string    `json:"accessToken"`       // 访问令牌
	RefreshToken      string    `json:"refreshToken"`      // 刷新令牌
}
