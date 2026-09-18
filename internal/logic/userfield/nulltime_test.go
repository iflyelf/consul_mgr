package userfield

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// TestNullTimeScan 复现并验证可空时间列（last_run_at / completed_at）为 NULL 时可正常读取。
//
// 回归背景：字段曾为 *time.Time，go-zero sqlx 扫描 NULL 会报
// "unsupported Scan, storing driver.Value type <nil> into type *time.Time"，
// 导致同步日志整体查询失败、同步配置被误判为「无配置」。
//
// 需要环境变量 TEST_DATABASE_URL；未设置则跳过。
func TestNullTimeScan(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL 未设置，跳过")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("连接数据库失败: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS userfield_sync_config (
			id SMALLINT PRIMARY KEY,
			enabled BOOLEAN NOT NULL DEFAULT FALSE,
			interval VARCHAR(32) NOT NULL DEFAULT '6h',
			sync_on_startup BOOLEAN NOT NULL DEFAULT FALSE,
			last_run_at TIMESTAMPTZ,
			last_status VARCHAR(32),
			last_message TEXT,
			updated_at TIMESTAMPTZ DEFAULT NOW()
		);
		CREATE TABLE IF NOT EXISTS userfield_sync_logs (
			id BIGSERIAL PRIMARY KEY,
			status VARCHAR(32) NOT NULL,
			trigger_type VARCHAR(32) NOT NULL DEFAULT '',
			total INT NOT NULL DEFAULT 0,
			added INT NOT NULL DEFAULT 0,
			updated INT NOT NULL DEFAULT 0,
			message TEXT,
			started_at TIMESTAMPTZ DEFAULT NOW(),
			completed_at TIMESTAMPTZ
		);`); err != nil {
		t.Fatalf("建表失败: %v", err)
	}

	l := NewLogic(sqlx.NewSqlConnFromDB(db))
	ctx := context.Background()

	// 配置：写入一行，last_run_at 为 NULL，应能读出且 Valid=false
	if _, err := db.Exec(`INSERT INTO userfield_sync_config (id, enabled, interval)
		VALUES (1, TRUE, '6h') ON CONFLICT (id) DO UPDATE SET last_run_at=NULL, enabled=TRUE`); err != nil {
		t.Fatalf("准备配置失败: %v", err)
	}
	cfg, err := l.GetSyncConfig(ctx)
	if err != nil {
		t.Fatalf("GetSyncConfig 失败（NULL last_run_at）: %v", err)
	}
	if !cfg.Enabled || cfg.LastRunAt.Valid {
		t.Fatalf("配置读取异常: enabled=%v lastRunAt=%+v", cfg.Enabled, cfg.LastRunAt)
	}

	// 日志：写入一条 running（completed_at 为 NULL），应能列出
	if _, err := db.Exec(`INSERT INTO userfield_sync_logs (status, trigger_type) VALUES ('running','ut')`); err != nil {
		t.Fatalf("准备日志失败: %v", err)
	}
	logs, err := l.ListSyncLogs(ctx, 5)
	if err != nil {
		t.Fatalf("ListSyncLogs 失败（NULL completed_at）: %v", err)
	}
	if len(logs) == 0 {
		t.Fatal("日志列表为空")
	}
	if logs[0].CompletedAt.Valid {
		t.Fatalf("running 日志 completedAt 应无效: %+v", logs[0].CompletedAt)
	}

	defer db.Exec(`DELETE FROM userfield_sync_logs WHERE trigger_type='ut'`)
}
