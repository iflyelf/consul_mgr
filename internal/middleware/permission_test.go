package middleware

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"

	"github.com/iflyelf/consul_mgr/internal/pkg/perm"
)

// TestCheckServiceGroupPermission 验证服务组/服务级权限校验逻辑
//
// 覆盖：
//  1. service_group_users 用户直接授权（视为全部服务）
//  2. 团队授权（指定服务）
//  3. 团队授权（全部服务 "*"）
//  4. 团队引用角色的权限
//  5. 未授权服务的拒绝
//
// 需要环境变量 TEST_DATABASE_URL；未设置则跳过。
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
	ctx := context.Background()

	var groupID int64
	if err := db.QueryRow(`
		INSERT INTO service_groups (name, code, consul_address, consul_datacenter)
		VALUES ('UT服务权限组','ut_svc_perm_group','http://127.0.0.1:8500','dc1')
		ON CONFLICT (name) DO UPDATE SET updated_at=NOW() RETURNING id`).Scan(&groupID); err != nil {
		t.Fatalf("创建服务组失败: %v", err)
	}
	defer db.Exec(`DELETE FROM service_groups WHERE id=$1`, groupID)
	defer db.Exec(`DELETE FROM team_members WHERE user_id LIKE 'ut-svc-%'`)
	defer db.Exec(`DELETE FROM service_group_users WHERE user_id LIKE 'ut-svc-%'`)
	defer db.Exec(`DELETE FROM team_group_permissions WHERE group_id=$1`, groupID)
	db.Exec(`DELETE FROM service_group_users WHERE group_id=$1`, groupID)

	var teamID int64
	if err := db.QueryRow(`INSERT INTO teams (name, code) VALUES ('UT服务团队','ut_svc_team') RETURNING id`).Scan(&teamID); err != nil {
		t.Fatalf("创建团队失败: %v", err)
	}
	defer db.Exec(`DELETE FROM teams WHERE id=$1`, teamID)

	c := perm.New(db)

	// 场景1：用户直接授权 → 全部服务
	u1 := "ut-svc-direct"
	if _, err := db.Exec(`INSERT INTO service_group_users (group_id,user_id,permissions) VALUES ($1,$2,$3)`,
		groupID, u1, "{read}"); err != nil {
		t.Fatalf("插入用户授权失败: %v", err)
	}
	if ok, _ := c.Check(ctx, u1, groupID, "anything", "read"); !ok {
		t.Errorf("场景1: 用户直授应可读任意服务")
	}
	if ok, _ := c.Check(ctx, u1, groupID, "anything", "write"); ok {
		t.Errorf("场景1: 仅 read 不应可写")
	}

	// 场景2：团队授权指定服务
	u2 := "ut-svc-team"
	db.Exec(`INSERT INTO team_members (team_id,user_id,username) VALUES ($1,$2,'t2')`, teamID, u2)
	if _, err := db.Exec(`INSERT INTO team_group_permissions (team_id,group_id,permissions,services)
		VALUES ($1,$2,$3,$4)`, teamID, groupID, "{read,write}", "{selfnode_exporter}"); err != nil {
		t.Fatalf("团队授权失败: %v", err)
	}
	if ok, _ := c.Check(ctx, u2, groupID, "selfnode_exporter", "write"); !ok {
		t.Errorf("场景2: 已授权服务应可写")
	}
	if ok, _ := c.Check(ctx, u2, groupID, "blackbox_exporter", "read"); ok {
		t.Errorf("场景2: 未授权服务应被拒绝")
	}
	if ok, _ := c.Check(ctx, u2, groupID, "", "read"); ok {
		t.Errorf("场景2: 组级操作需全部服务权限，应被拒绝")
	}

	// 场景3：改为全部服务
	db.Exec(`UPDATE team_group_permissions SET services='{*}' WHERE team_id=$1 AND group_id=$2`, teamID, groupID)
	if ok, _ := c.Check(ctx, u2, groupID, "blackbox_exporter", "read"); !ok {
		t.Errorf("场景3: 全部服务授权应覆盖任意服务")
	}
	if ok, _ := c.Check(ctx, u2, groupID, "", "read"); !ok {
		t.Errorf("场景3: 全部服务授权应允许组级操作")
	}

	// 场景4：空服务 = 无权限
	db.Exec(`UPDATE team_group_permissions SET services='{}' WHERE team_id=$1 AND group_id=$2`, teamID, groupID)
	if ok, _ := c.Check(ctx, u2, groupID, "selfnode_exporter", "read"); ok {
		t.Errorf("场景4: services 为空应无任何服务权限")
	}

	// 场景5：团队引用角色（无直接权限）
	var roleID int64
	if err := db.QueryRow(`INSERT INTO roles (name, code, permissions) VALUES ('UT只读角色','ut_svc_role','{read}') RETURNING id`).Scan(&roleID); err != nil {
		t.Fatalf("创建角色失败: %v", err)
	}
	defer db.Exec(`DELETE FROM roles WHERE id=$1`, roleID)
	u3 := "ut-svc-role"
	db.Exec(`INSERT INTO team_members (team_id,user_id,username) VALUES ($1,$2,'t3')`, teamID, u3)
	db.Exec(`UPDATE team_group_permissions SET permissions='{}', role_ids=$3, services=$4
		WHERE team_id=$1 AND group_id=$2`, teamID, groupID, "{"+itoa(roleID)+"}", "{selfrds_exporter}")
	if ok, _ := c.Check(ctx, u3, groupID, "selfrds_exporter", "read"); !ok {
		t.Errorf("场景5: 角色权限应生效")
	}
	if ok, _ := c.Check(ctx, u3, groupID, "selfrds_exporter", "write"); ok {
		t.Errorf("场景5: 角色仅 read，不应可写")
	}

	// 场景6：AllowedServices
	all, list, err := c.AllowedServices(ctx, u3, groupID)
	if err != nil {
		t.Fatalf("AllowedServices 失败: %v", err)
	}
	if all || len(list) != 1 || list[0] != "selfrds_exporter" {
		t.Errorf("场景6: AllowedServices 应为 [selfrds_exporter]，实际 all=%v list=%v", all, list)
	}
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
