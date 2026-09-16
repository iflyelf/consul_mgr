// Package user 提供「用户管理」逻辑
//
// 说明:
//   用户体系由 Casdoor 维护。为避免「人员一多就卡住」：
//   1. 全量用户缓存到 Redis（TTL 内复用，避免每次请求都打 Casdoor）；
//   2. 关键字过滤与分页在内存完成，结果稳定且可控；
//   3. Casdoor 自带分页行为不稳定（页间重叠），故不直接使用。
package user

import (
	"context"
	"sort"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/iflyelf/consul_mgr/internal/pkg/cache"
	"github.com/iflyelf/consul_mgr/internal/pkg/casdoor"
)

// 缓存键：全量 Casdoor 用户
const usersCacheKey = "consul_mgr:casdoor_users:all"

// UserLogic 用户管理逻辑
type UserLogic struct {
	ctx    context.Context
	client *casdoor.Client
	cache  *cache.Cache
	logger logx.Logger
}

// NewUserLogic 创建用户管理逻辑实例
func NewUserLogic(ctx context.Context, client *casdoor.Client, c *cache.Cache) *UserLogic {
	return &UserLogic{
		ctx:    ctx,
		client: client,
		cache:  c,
		logger: logx.WithContext(ctx),
	}
}

// ListUsers 查询用户列表（缓存 + 关键字过滤 + 分页）
//
// 参数:
//   keyword  - 用户名 / 显示名 / 邮箱 模糊匹配（大小写不敏感）
//   page     - 页码（从 1 开始）
//   pageSize - 每页条数
//
// 返回:
//   []casdoor.CasdoorUser - 当前页数据
//   int                   - 过滤后的总数
//   error                 - 错误
func (l *UserLogic) ListUsers(keyword string, page, pageSize int) ([]casdoor.CasdoorUser, int, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	all, err := l.allUsers()
	if err != nil {
		return nil, 0, err
	}

	// 关键字过滤（内存）
	if keyword != "" {
		kw := strings.ToLower(keyword)
		filtered := make([]casdoor.CasdoorUser, 0, len(all))
		for _, u := range all {
			if strings.Contains(strings.ToLower(u.Name), kw) ||
				strings.Contains(strings.ToLower(u.DisplayName), kw) ||
				strings.Contains(strings.ToLower(u.Email), kw) {
				filtered = append(filtered, u)
			}
		}
		all = filtered
	}

	total := len(all)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return all[start:end], total, nil
}

// allUsers 获取全量用户（优先命中 Redis 缓存）
func (l *UserLogic) allUsers() ([]casdoor.CasdoorUser, error) {
	var cached []casdoor.CasdoorUser
	if l.cache.Get(l.ctx, usersCacheKey, &cached) {
		return cached, nil
	}

	users, err := l.client.ListUsers()
	if err != nil {
		return nil, err
	}

	// 稳定排序（按用户名），保证分页结果一致
	sort.SliceStable(users, func(i, j int) bool {
		if users[i].Name == users[j].Name {
			return users[i].Id < users[j].Id
		}
		return users[i].Name < users[j].Name
	})

	l.cache.Set(l.ctx, usersCacheKey, users)
	l.logger.Infof("Casdoor 用户列表已缓存: %d 个", len(users))
	return users, nil
}
