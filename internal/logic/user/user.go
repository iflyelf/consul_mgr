// Package user 提供「用户管理」逻辑
//
// 说明:
//
//	用户体系由 Casdoor 维护。为避免「人员一多就卡住」：
//	1. 全量用户缓存到 Redis（TTL 内复用，避免每次请求都打 Casdoor）；
//	2. 关键字过滤与分页在内存完成，结果稳定且可控；
//	3. 拉取全量时按唯一键 name 稳定排序分页（见 casdoor.IterateUsers），
//	   既避免无分页接口的内存放大（OOM），也避免页间重叠/漏读。
package user

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/syncx"

	"github.com/iflyelf/consul_mgr/internal/pkg/cache"
	"github.com/iflyelf/consul_mgr/internal/pkg/casdoor"
)

// 缓存键：全量 Casdoor 用户
const usersCacheKey = "consul_mgr:casdoor_users:all"

// defaultUserCacheTTL 用户列表缓存默认过期时间（可经 CASDOOR_USER_CACHE_TTL 覆盖）。
//
// 说明：用户列表会被外部（Casdoor）直接修改，本地无法感知；此处使用较短 TTL
// （而非通用缓存的 300s），将「外部改动不可见」的窗口缩短。
// 如需立即刷新，可在 ListUsers 传 refresh=true 强制绕过缓存。
const defaultUserCacheTTL = 30 * time.Second

// 防止缓存过期瞬间并发回源（缓存击穿）：相同 key 的并发请求共享一次加载。
var userFlight = syncx.NewSingleFlight()

// UserLogic 用户管理逻辑
type UserLogic struct {
	ctx      context.Context
	client   *casdoor.Client
	cache    *cache.Cache
	cacheTTL time.Duration
	logger   logx.Logger
}

// NewUserLogic 创建用户管理逻辑实例。
//
// cacheTTLSeconds 为用户列表缓存时长（秒），<=0 时使用默认值。
func NewUserLogic(ctx context.Context, client *casdoor.Client, c *cache.Cache, cacheTTLSeconds int) *UserLogic {
	ttl := defaultUserCacheTTL
	if cacheTTLSeconds > 0 {
		ttl = time.Duration(cacheTTLSeconds) * time.Second
	}
	return &UserLogic{
		ctx:      ctx,
		client:   client,
		cache:    c,
		cacheTTL: ttl,
		logger:   logx.WithContext(ctx),
	}
}

// InvalidateUsersCache 主动失效用户列表缓存（供管理接口在外部变更后调用）
func (l *UserLogic) InvalidateUsersCache() {
	l.cache.Del(l.ctx, usersCacheKey)
}

// ListUsers 查询用户列表（缓存 + 关键字过滤 + 分页）
//
// 参数:
//
//	keyword  - 用户名 / 显示名 / 邮箱 模糊匹配（大小写不敏感）
//	page     - 页码（从 1 开始）
//	pageSize - 每页条数
//
// 返回:
//
//	[]casdoor.CasdoorUser - 当前页数据
//	int                   - 过滤后的总数
//	error                 - 错误
func (l *UserLogic) ListUsers(keyword string, page, pageSize int, refresh ...bool) ([]casdoor.CasdoorUser, int, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	forceRefresh := len(refresh) > 0 && refresh[0]
	if forceRefresh {
		l.cache.Del(l.ctx, usersCacheKey)
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

// 以下为用户管理（与 FlyIAM 对齐：新增/编辑/删除/重置密码/管理员标记）。
// 用户唯一存储于 Casdoor；写操作后清理列表缓存，保证下次读取为新数据。

// CreateUser 新增用户
func (l *UserLogic) CreateUser(in casdoor.UserUpsert, defaultPassword string) error {
	if err := l.client.CreateUser(in, defaultPassword); err != nil {
		return err
	}
	l.cache.Del(l.ctx, usersCacheKey)
	return nil
}

// UpdateUser 更新用户
func (l *UserLogic) UpdateUser(in casdoor.UserUpsert) error {
	if err := l.client.UpdateUser(in); err != nil {
		return err
	}
	l.cache.Del(l.ctx, usersCacheKey)
	return nil
}

// DeleteUser 删除用户
func (l *UserLogic) DeleteUser(name string) error {
	if err := l.client.DeleteUser(name); err != nil {
		return err
	}
	l.cache.Del(l.ctx, usersCacheKey)
	return nil
}

// ResetPassword 重置用户密码
func (l *UserLogic) ResetPassword(name, password string) error {
	if err := l.client.ResetPassword(name, password); err != nil {
		return err
	}
	return nil
}

// SetUserAdmin 设置/取消管理员
func (l *UserLogic) SetUserAdmin(name string, isAdmin bool) error {
	if err := l.client.SetUserAdmin(name, isAdmin); err != nil {
		return err
	}
	l.cache.Del(l.ctx, usersCacheKey)
	return nil
}

// allUsers 获取全量用户（优先命中 Redis 缓存）。
//
// 防击穿：缓存未命中时通过 SingleFlight 合并并发回源，避免缓存过期瞬间
// 大量请求同时打 Casdoor。
func (l *UserLogic) allUsers() ([]casdoor.CasdoorUser, error) {
	var cached []casdoor.CasdoorUser
	if l.cache.Get(l.ctx, usersCacheKey, &cached) {
		return cached, nil
	}

	val, err := userFlight.Do(usersCacheKey, func() (interface{}, error) {
		// 二次确认（可能在等待期间已被其他协程写入）
		var again []casdoor.CasdoorUser
		if l.cache.Get(l.ctx, usersCacheKey, &again) {
			return again, nil
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

		// 用户列表使用较短 TTL，缩短外部改动不可见窗口
		l.cache.SetWithTTL(l.ctx, usersCacheKey, users, l.cacheTTL)
		l.logger.Infof("Casdoor 用户列表已缓存: %d 个（TTL=%s）", len(users), l.cacheTTL)
		return users, nil
	})
	if err != nil {
		return nil, err
	}
	users, _ := val.([]casdoor.CasdoorUser)
	return users, nil
}
