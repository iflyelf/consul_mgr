// Package team 提供团队管理逻辑
//
// 团队是权限分配的主体：团队包含成员，并被授权访问若干服务组
// （授权 = 直接权限 read/write/delete ∪ 引用角色的权限）。
package team

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
	"github.com/zeromicro/go-zero/core/logx"
)

// TeamLogic 团队管理逻辑
type TeamLogic struct {
	ctx    context.Context
	db     *sql.DB
	logger logx.Logger
}

// NewTeamLogic 创建团队管理逻辑实例
func NewTeamLogic(ctx context.Context, db *sql.DB) *TeamLogic {
	return &TeamLogic{ctx: ctx, db: db, logger: logx.WithContext(ctx)}
}

// Team 团队模型
type Team struct {
	ID          int64  `db:"id" json:"id"`
	Name        string `db:"name" json:"name"`
	Code        string `db:"code" json:"code"`
	Description string `db:"description" json:"description"`
	Status      int    `db:"status" json:"status"`
	CreatedBy   string `db:"created_by" json:"created_by"`
	CreatedAt   string `db:"created_at" json:"created_at"`
	UpdatedAt   string `db:"updated_at" json:"updated_at"`
	// 统计
	MemberCount int `db:"member_count" json:"member_count"`
	GroupCount  int `db:"group_count" json:"group_count"`
}

// Member 团队成员
type Member struct {
	ID          int64  `db:"id" json:"id"`
	TeamID      int64  `db:"team_id" json:"team_id"`
	UserID      string `db:"user_id" json:"user_id"`
	Username    string `db:"username" json:"username"`
	DisplayName string `db:"display_name" json:"display_name"`
	CreatedAt   string `db:"created_at" json:"created_at"`
}

// GroupPermission 团队对服务组的授权
type GroupPermission struct {
	ID          int64    `db:"id" json:"id"`
	TeamID      int64    `db:"team_id" json:"team_id"`
	GroupID     int64    `db:"group_id" json:"group_id"`
	GroupName   string   `db:"group_name" json:"group_name"`
	Permissions []string `db:"permissions" json:"permissions"`
	RoleIDs     []int64  `db:"role_ids" json:"role_ids"`
	CreatedAt   string   `db:"created_at" json:"created_at"`
}

const teamSelect = `
	SELECT t.id, t.name, t.code, COALESCE(t.description,'') AS description, t.status,
	       COALESCE(t.created_by,'') AS created_by,
	       TO_CHAR(t.created_at,'YYYY-MM-DD HH24:MI:SS'),
	       TO_CHAR(t.updated_at,'YYYY-MM-DD HH24:MI:SS'),
	       (SELECT COUNT(*) FROM team_members m WHERE m.team_id = t.id) AS member_count,
	       (SELECT COUNT(*) FROM team_group_permissions g WHERE g.team_id = t.id) AS group_count
	FROM teams t
`

// CreateTeam 创建团队
func (l *TeamLogic) CreateTeam(name, code, description, createdBy string) (*Team, error) {
	if code == "" {
		code = name
	}
	query := `
		INSERT INTO teams (name, code, description, status, created_by, created_at, updated_at)
		VALUES ($1,$2,$3,1,$4,NOW(),NOW())
		RETURNING id
	`
	var id int64
	if err := l.db.QueryRowContext(l.ctx, query, name, code, description, createdBy).Scan(&id); err != nil {
		l.logger.Errorf("创建团队失败: %v", err)
		return nil, fmt.Errorf("创建团队失败: %w", err)
	}
	return l.GetTeam(id)
}

// UpdateTeam 更新团队
func (l *TeamLogic) UpdateTeam(id int64, name, description string, status *int) (*Team, error) {
	original, err := l.GetTeam(id)
	if err != nil {
		return nil, err
	}
	if name == "" {
		name = original.Name
	}
	if description == "" {
		description = original.Description
	}
	st := original.Status
	if status != nil {
		st = *status
	}
	if _, err := l.db.ExecContext(l.ctx,
		`UPDATE teams SET name=$1, description=$2, status=$3, updated_at=NOW() WHERE id=$4`,
		name, description, st, id); err != nil {
		return nil, fmt.Errorf("更新团队失败: %w", err)
	}
	return l.GetTeam(id)
}

// DeleteTeam 删除团队
func (l *TeamLogic) DeleteTeam(id int64) error {
	res, err := l.db.ExecContext(l.ctx, `DELETE FROM teams WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("删除团队失败: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("团队不存在: ID=%d", id)
	}
	return nil
}

// GetTeam 获取团队
func (l *TeamLogic) GetTeam(id int64) (*Team, error) {
	var t Team
	if err := l.db.QueryRowContext(l.ctx, teamSelect+` WHERE t.id=$1`, id).Scan(
		&t.ID, &t.Name, &t.Code, &t.Description, &t.Status, &t.CreatedBy,
		&t.CreatedAt, &t.UpdatedAt, &t.MemberCount, &t.GroupCount); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("团队不存在: ID=%d", id)
		}
		return nil, err
	}
	return &t, nil
}

// ListTeams 查询团队列表
func (l *TeamLogic) ListTeams(keyword string) ([]*Team, error) {
	query := teamSelect
	args := []interface{}{}
	if keyword != "" {
		query += ` WHERE t.name LIKE $1 OR t.code LIKE $1`
		args = append(args, "%"+keyword+"%")
	}
	query += ` ORDER BY t.id ASC`

	rows, err := l.db.QueryContext(l.ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询团队列表失败: %w", err)
	}
	defer rows.Close()
	var list []*Team
	for rows.Next() {
		var t Team
		if err := rows.Scan(&t.ID, &t.Name, &t.Code, &t.Description, &t.Status, &t.CreatedBy,
			&t.CreatedAt, &t.UpdatedAt, &t.MemberCount, &t.GroupCount); err != nil {
			return nil, err
		}
		list = append(list, &t)
	}
	return list, rows.Err()
}

// ListMembers 查询团队成员
func (l *TeamLogic) ListMembers(teamID int64) ([]*Member, error) {
	query := `
		SELECT id, team_id, user_id, COALESCE(username,'') AS username,
		       COALESCE(display_name,'') AS display_name,
		       TO_CHAR(created_at,'YYYY-MM-DD HH24:MI:SS')
		FROM team_members WHERE team_id=$1 ORDER BY id ASC
	`
	rows, err := l.db.QueryContext(l.ctx, query, teamID)
	if err != nil {
		return nil, fmt.Errorf("查询团队成员失败: %w", err)
	}
	defer rows.Close()
	var list []*Member
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ID, &m.TeamID, &m.UserID, &m.Username, &m.DisplayName, &m.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, &m)
	}
	return list, rows.Err()
}

// AddMember 添加团队成员
func (l *TeamLogic) AddMember(teamID int64, userID, username, displayName string) error {
	query := `
		INSERT INTO team_members (team_id, user_id, username, display_name)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (team_id, user_id) DO UPDATE
		SET username=EXCLUDED.username, display_name=EXCLUDED.display_name
	`
	if _, err := l.db.ExecContext(l.ctx, query, teamID, userID, username, displayName); err != nil {
		return fmt.Errorf("添加团队成员失败: %w", err)
	}
	return nil
}

// RemoveMember 移除团队成员
func (l *TeamLogic) RemoveMember(teamID int64, userID string) error {
	res, err := l.db.ExecContext(l.ctx, `DELETE FROM team_members WHERE team_id=$1 AND user_id=$2`, teamID, userID)
	if err != nil {
		return fmt.Errorf("移除团队成员失败: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("成员不存在")
	}
	return nil
}

// ListGroupPermissions 查询团队的服务组授权
func (l *TeamLogic) ListGroupPermissions(teamID int64) ([]*GroupPermission, error) {
	query := `
		SELECT gp.id, gp.team_id, gp.group_id,
		       COALESCE(g.name,'') AS group_name,
		       gp.permissions,
		       COALESCE(gp.role_ids, '{}') AS role_ids,
		       TO_CHAR(gp.created_at,'YYYY-MM-DD HH24:MI:SS')
		FROM team_group_permissions gp
		LEFT JOIN service_groups g ON g.id = gp.group_id
		WHERE gp.team_id=$1 ORDER BY gp.group_id ASC
	`
	rows, err := l.db.QueryContext(l.ctx, query, teamID)
	if err != nil {
		return nil, fmt.Errorf("查询团队授权失败: %w", err)
	}
	defer rows.Close()
	var list []*GroupPermission
	for rows.Next() {
		var gp GroupPermission
		var perms []string
		var rids []int64
		if err := rows.Scan(&gp.ID, &gp.TeamID, &gp.GroupID, &gp.GroupName,
			pq.Array(&perms), pq.Array(&rids), &gp.CreatedAt); err != nil {
			return nil, err
		}
		gp.Permissions = perms
		gp.RoleIDs = rids
		list = append(list, &gp)
	}
	return list, rows.Err()
}

// GrantGroupPermission 授予团队对服务组的权限
func (l *TeamLogic) GrantGroupPermission(teamID, groupID int64, permissions []string, roleIDs []int64) error {
	if permissions == nil {
		permissions = []string{}
	}
	// 校验权限取值
	for _, p := range permissions {
		if p != "read" && p != "write" && p != "delete" {
			return fmt.Errorf("无效的权限: %s（仅支持 read/write/delete）", p)
		}
	}
	if roleIDs == nil {
		roleIDs = []int64{}
	}
	query := `
		INSERT INTO team_group_permissions (team_id, group_id, permissions, role_ids, created_at, updated_at)
		VALUES ($1,$2,$3,$4,NOW(),NOW())
		ON CONFLICT (team_id, group_id) DO UPDATE
		SET permissions=EXCLUDED.permissions, role_ids=EXCLUDED.role_ids, updated_at=NOW()
	`
	if _, err := l.db.ExecContext(l.ctx, query, teamID, groupID, pq.Array(permissions), pq.Array(roleIDs)); err != nil {
		return fmt.Errorf("授予团队权限失败: %w", err)
	}
	return nil
}

// RevokeGroupPermission 撤销团队对服务组的授权
func (l *TeamLogic) RevokeGroupPermission(teamID, groupID int64) error {
	res, err := l.db.ExecContext(l.ctx, `DELETE FROM team_group_permissions WHERE team_id=$1 AND group_id=$2`, teamID, groupID)
	if err != nil {
		return fmt.Errorf("撤销团队权限失败: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("授权不存在")
	}
	return nil
}

// CheckUserGroupPermission 检查用户通过「团队授权」对服务组是否具备某权限
//
// 逻辑:
//  1. 找出用户所在的所有团队；
//  2. 取这些团队对目标服务组的授权；
//  3. 直接权限 ∪ 引用角色的权限，判断是否包含 action。
func (l *TeamLogic) CheckUserGroupPermission(userID string, groupID int64, action string) (bool, error) {
	query := `
		SELECT gp.permissions,
		       COALESCE(gp.role_ids, '{}') AS role_ids
		FROM team_group_permissions gp
		JOIN team_members m ON m.team_id = gp.team_id
		WHERE m.user_id = $1 AND gp.group_id = $2
	`
	rows, err := l.db.QueryContext(l.ctx, query, userID, groupID)
	if err != nil {
		return false, fmt.Errorf("查询团队授权失败: %w", err)
	}
	defer rows.Close()

	var allRoleIDs []int64
	directMatch := false
	for rows.Next() {
		var perms []string
		var rids []int64
		if err := rows.Scan(pq.Array(&perms), pq.Array(&rids)); err != nil {
			return false, err
		}
		for _, p := range perms {
			if p == "*" || p == action {
				directMatch = true
			}
		}
		allRoleIDs = append(allRoleIDs, rids...)
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	if directMatch {
		return true, nil
	}
	if len(allRoleIDs) == 0 {
		return false, nil
	}

	// 引用角色的权限
	rrows, err := l.db.QueryContext(l.ctx,
		`SELECT DISTINCT unnest(permissions) FROM roles WHERE id = ANY($1)`,
		pq.Array(allRoleIDs))
	if err != nil {
		l.logger.Errorf("查询角色权限失败: %v", err)
		return false, nil
	}
	defer rrows.Close()
	for rrows.Next() {
		var p string
		if err := rrows.Scan(&p); err != nil {
			continue
		}
		if p == "*" || p == action {
			return true, nil
		}
	}
	return false, nil
}
