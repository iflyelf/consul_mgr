package middleware

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

// TestCheckServiceGroupPermission 验证服务组级权限校验逻辑
//
// 覆盖三种授权来源：
//  1. service_group_users 用户直接授权
//  2. team_members + team_group_permissions 团队授权（直接权限）
//  3. team 授权中引用角色的权限（roles.permissions）
//
// 需要环境变量 TEST_DATABASE_URL 指向可写的测试库；未设置则跳过。
func TestCheckServiceGroupPermission(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL 未设置，跳过")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("连接数据库失败: %v", err)
	}
	defer db.Close()


	// 准备：一个服务组
	var groupID int64
	groupCode := "ut_perm_group"
	if err := db.QueryRow(
		`INSERT INTO service_groups (name, code, consul_address, consul_datacenter)
		 VALUES ($1, $2, 'http://127.0.0.1:8500', 'dc1')
		 ON CONFLICT (name) DO UPDATE SET updated_at=NOW() RETURNING id`,
		"UT权限测试组", groupCode).Scan(&groupID); err != nil {
		t.Fatalf("创建服务组失败: %v", err)
	}
	defer func() {
		db.Exec(`DELETE FROM service_groups WHERE id=$1`, groupID)
	}()

	// 清理可能的历史残留
	db.Exec(`DELETE FROM service_group_users WHERE group_id=$1`, groupID)
	db.Exec(`DELETE FROM team_group_permissions WHERE group_id=$1`, groupID)

	m := &PermissionMiddleware{db: db}
	userDirect := "ut-user-direct"
	userTeam := "ut-user-team"
	userRole := "ut-user-role"
	userNone := "ut-user-none"

	// 场景 1：用户直接授权
	if _, err := db.Exec(
		`INSERT INTO service_group_users (group_id, user_id, permissions) VALUES ($1,$2,$3)`,
		groupID, userDirect, "{read}"); err != nil {
		t.Fatalf("插入用户授权失败: %v", err)
	}
	defer db.Exec(`DELETE FROM service_group_users WHERE user_id LIKE 'ut-user-%'`)

	got, err := m.checkServiceGroupPermission(userDirect, groupID, "read")
	if err != nil || !got {
		t.Errorf("场景1(用户直授 read): got=%v err=%v, 期望 true", got, err)
	}
	got, _ = m.checkServiceGroupPermission(userDirect, groupID, "write")
	if got {
		t.Errorf("场景1(用户直授 仅read): write 应被拒绝")
	}

	// 场景 2：团队授权（直接权限 write）
	var teamID int64
	if err := db.QueryRow(
		`INSERT INTO teams (name, code) VALUES ('UT测试团队','ut_team') RETURNING id`).Scan(&teamID); err != nil {
		t.Fatalf("创建团队失败: %v", err)
	}
	defer db.Exec(`DELETE FROM teams WHERE id=$1`, teamID)

	if _, err := db.Exec(
		`INSERT INTO team_members (team_id, user_id, username) VALUES ($1,$2,'teamuser')`,
		teamID, userTeam); err != nil {
		t.Fatalf("添加成员失败: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO team_group_permissions (team_id, group_id, permissions) VALUES ($1,$2,$3)`,
		teamID, groupID, "{write}"); err != nil {
		t.Fatalf("团队授权失败: %v", err)
	}
	got, err = m.checkServiceGroupPermission(userTeam, groupID, "write")
	if err != nil || !got {
		t.Errorf("场景2(团队授权 write): got=%v err=%v, 期望 true", got, err)
	}
	got, _ = m.checkServiceGroupPermission(userTeam, groupID, "delete")
	if got {
		t.Errorf("场景2(团队仅write): delete 应被拒绝")
	}

	// 场景 3：团队引用角色的权限
	var roleID int64
	if err := db.QueryRow(
		`INSERT INTO roles (name, code, permissions) VALUES ('UT运维角色','ut_role',$1) RETURNING id`,
		"{read,delete}").Scan(&roleID); err != nil {
		t.Fatalf("创建角色失败: %v", err)
	}
	defer db.Exec(`DELETE FROM roles WHERE id=$1`, roleID)

	if _, err := db.Exec(
		`INSERT INTO team_members (team_id, user_id, username) VALUES ($1,$2,'roleuser')`,
		teamID, userRole); err != nil {
		t.Fatalf("添加成员失败: %v", err)
	}
	// 更新团队授权：无直接权限，仅引用角色
	if _, err := db.Exec(
		`UPDATE team_group_permissions SET permissions='{}', role_ids=$2 WHERE team_id=$1 AND group_id=$3`,
		teamID, "{"+itoa(roleID)+"}", groupID); err != nil {
		t.Fatalf("更新团队授权失败: %v", err)
	}
	got, err = m.checkServiceGroupPermission(userRole, groupID, "delete")
	if err != nil || !got {
		t.Errorf("场景3(角色含delete): got=%v err=%v, 期望 true", got, err)
	}
	got, _ = m.checkServiceGroupPermission(userRole, groupID, "write")
	if got {
		t.Errorf("场景3(角色无write): write 应被拒绝")
	}

	// 场景 4：无任何授权的用户
	got, err = m.checkServiceGroupPermission(userNone, groupID, "read")
	if err != nil || got {
		t.Errorf("场景4(无授权): got=%v err=%v, 期望 false", got, err)
	}

	// 清理团队成员/授权
	db.Exec(`DELETE FROM team_members WHERE user_id LIKE 'ut-user-%'`)
	db.Exec(`DELETE FROM team_group_permissions WHERE team_id=$1`, teamID)
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
