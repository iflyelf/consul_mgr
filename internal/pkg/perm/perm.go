// Package perm 提供服务组 / 服务级权限判定
//
// 权限来源（任一满足即可）:
//  1. service_group_users：用户对服务组的直接授权（视为该组全部服务）
//  2. team_group_permissions：用户所在团队对服务组的授权
//     - permissions：直接权限
//     - role_ids：引用角色的权限（取并集）
//     - services：授权的服务名
//     · 空 []        = 无任何服务权限
//     · ["*"]        = 该服务组下全部服务
//     · ["a","b"]    = 仅限这些服务
//
// 权限取值: read / write / delete，支持 "*" 通配。
package perm

import (
	"context"
	"database/sql"

	"github.com/lib/pq"
)

// AllServices 表示「全部服务」的特殊标记
const AllServices = "*"

// Checker 权限判定器
type Checker struct {
	db *sql.DB
}

// New 创建权限判定器
func New(db *sql.DB) *Checker {
	return &Checker{db: db}
}

// Check 判断用户是否具备某操作权限
//
// 参数:
//
//	userID      - Casdoor 用户 ID
//	groupID     - 服务组 ID
//	serviceName - 服务名；为空表示「组级」判定（仅 all 授权可通过）
//	action      - read / write / delete
func (c *Checker) Check(ctx context.Context, userID string, groupID int64, serviceName, action string) (bool, error) {
	if c == nil || c.db == nil {
		return false, nil
	}

	// 1. 用户对服务组的直接授权（视为全部服务）
	var directPerms []string
	err := c.db.QueryRowContext(ctx,
		`SELECT permissions FROM service_group_users WHERE group_id=$1 AND user_id=$2`,
		groupID, userID).Scan(pq.Array(&directPerms))
	if err == nil {
		if permissionAllows(directPerms, action) {
			return true, nil
		}
	} else if err != sql.ErrNoRows {
		return false, err
	}

	// 2. 团队授权
	rows, err := c.db.QueryContext(ctx, `
		SELECT gp.permissions,
		       COALESCE(gp.role_ids, '{}') AS role_ids,
		       COALESCE(gp.services, '{}') AS services
		FROM team_group_permissions gp
		JOIN team_members tm ON tm.team_id = gp.team_id
		WHERE tm.user_id = $1 AND gp.group_id = $2
	`, userID, groupID)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	type grant struct {
		permissions []string
		roleIDs     []int64
		services    []string
	}
	var grants []grant
	var allRoleIDs []int64
	for rows.Next() {
		var g grant
		if err := rows.Scan(pq.Array(&g.permissions), pq.Array(&g.roleIDs), pq.Array(&g.services)); err != nil {
			return false, err
		}
		grants = append(grants, g)
		allRoleIDs = append(allRoleIDs, g.roleIDs...)
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	if len(grants) == 0 {
		return false, nil
	}

	// 汇总角色权限
	rolePerms := map[string]bool{}
	if len(allRoleIDs) > 0 {
		rrows, rerr := c.db.QueryContext(ctx,
			`SELECT DISTINCT unnest(permissions) FROM roles WHERE id = ANY($1)`,
			pq.Array(allRoleIDs))
		if rerr == nil {
			defer rrows.Close()
			for rrows.Next() {
				var p string
				if rrows.Scan(&p) == nil {
					rolePerms[p] = true
				}
			}
		}
	}

	for _, g := range grants {
		// 权限校验（直接权限 ∪ 角色权限）
		if !permissionAllows(g.permissions, action) && !rolePerms[AllServices] && !rolePerms[action] {
			continue
		}
		// 服务维度校验
		if serviceAllowed(g.services, serviceName) {
			return true, nil
		}
	}
	return false, nil
}

// permissionAllows 判断权限列表是否包含目标操作
func permissionAllows(perms []string, action string) bool {
	for _, p := range perms {
		if p == AllServices || p == action {
			return true
		}
	}
	return false
}

// serviceAllowed 判断授权服务列表是否覆盖目标服务
//
// 规则:
//   - 列表为空            → 无任何服务权限
//   - 包含 "*"            → 全部服务
//   - serviceName 为空    → 组级操作，仅 "*" 可通过
//   - 否则                → 需精确命中
func serviceAllowed(services []string, serviceName string) bool {
	if len(services) == 0 {
		return false
	}
	hasAll := false
	for _, s := range services {
		if s == AllServices {
			hasAll = true
			break
		}
	}
	if hasAll {
		return true
	}
	if serviceName == "" {
		// 组级操作（如创建新服务）要求全部服务权限
		return false
	}
	for _, s := range services {
		if s == serviceName {
			return true
		}
	}
	return false
}

// AllowedServices 返回用户在指定服务组被授权的服务集合
//
// 返回:
//
//	all      - 是否拥有全部服务权限
//	services - 明确授权的服务名（all=false 时有意义）
func (c *Checker) AllowedServices(ctx context.Context, userID string, groupID int64) (bool, []string, error) {
	if c == nil || c.db == nil {
		return false, nil, nil
	}

	// 用户直接授权 → 视为全部服务
	var direct []string
	err := c.db.QueryRowContext(ctx,
		`SELECT permissions FROM service_group_users WHERE group_id=$1 AND user_id=$2`,
		groupID, userID).Scan(pq.Array(&direct))
	if err == nil && len(direct) > 0 {
		return true, nil, nil
	}

	rows, err := c.db.QueryContext(ctx, `
		SELECT COALESCE(gp.services, '{}') AS services
		FROM team_group_permissions gp
		JOIN team_members tm ON tm.team_id = gp.team_id
		WHERE tm.user_id = $1 AND gp.group_id = $2
	`, userID, groupID)
	if err != nil {
		return false, nil, err
	}
	defer rows.Close()

	set := map[string]bool{}
	for rows.Next() {
		var services []string
		if err := rows.Scan(pq.Array(&services)); err != nil {
			return false, nil, err
		}
		for _, s := range services {
			if s == AllServices {
				return true, nil, nil
			}
			set[s] = true
		}
	}
	if err := rows.Err(); err != nil {
		return false, nil, err
	}
	list := make([]string, 0, len(set))
	for s := range set {
		list = append(list, s)
	}
	return false, list, nil
}

// HasAnyGroupAccess 判断用户是否对服务组有任意权限（用于列表过滤）
func (c *Checker) HasAnyGroupAccess(ctx context.Context, userID string, groupID int64) (bool, error) {
	allowed, list, err := c.AllowedServices(ctx, userID, groupID)
	if err != nil {
		return false, err
	}
	return allowed || len(list) > 0, nil
}
