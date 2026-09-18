package userfield

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"github.com/iflyelf/consul_mgr/internal/model"
)

// syncConfigID 单例配置行 ID
const syncConfigID = 1

// 内存进度（供前端轮询；进程重启后由日志恢复）
var (
	progressMu   sync.RWMutex
	progressData = model.UserFieldSyncProgress{}
)

// getProgress 读取当前同步进度
func getProgress() model.UserFieldSyncProgress {
	progressMu.RLock()
	defer progressMu.RUnlock()
	return progressData
}

// setProgress 更新同步进度
func setProgress(p model.UserFieldSyncProgress) {
	progressMu.Lock()
	progressData = p
	progressMu.Unlock()
}

// GetSyncConfig 读取自动同步配置（不存在则返回默认值）
func (l *Logic) GetSyncConfig(ctx context.Context) (*model.UserFieldSyncConfig, error) {
	var cfg model.UserFieldSyncConfig
	query := `SELECT id, enabled, interval, sync_on_startup, last_run_at,
		COALESCE(last_status,'') AS last_status, COALESCE(last_message,'') AS last_message, updated_at
		FROM userfield_sync_config WHERE id = $1`
	if err := l.db.QueryRowCtx(ctx, &cfg, query, syncConfigID); err != nil {
		// 无配置行时返回默认（关闭自动同步）
		return &model.UserFieldSyncConfig{
			ID: syncConfigID, Enabled: false, Interval: "6h", SyncOnStartup: false,
		}, nil
	}
	return &cfg, nil
}

// UpdateSyncConfig 更新自动同步配置
func (l *Logic) UpdateSyncConfig(ctx context.Context, cfg *model.UserFieldSyncConfig) error {
	cfg.Interval = strings.TrimSpace(cfg.Interval)
	if cfg.Interval == "" {
		cfg.Interval = "6h"
	}
	if _, err := ParseInterval(cfg.Interval); err != nil {
		return err
	}
	query := `
		INSERT INTO userfield_sync_config (id, enabled, interval, sync_on_startup, updated_at)
		VALUES ($1,$2,$3,$4,NOW())
		ON CONFLICT (id) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			interval = EXCLUDED.interval,
			sync_on_startup = EXCLUDED.sync_on_startup,
			updated_at = NOW()
	`
	if _, err := l.db.ExecCtx(ctx, query, syncConfigID, cfg.Enabled, cfg.Interval, cfg.SyncOnStartup); err != nil {
		return fmt.Errorf("更新同步配置失败: %w", err)
	}
	return nil
}

// updateLastRun 记录上次执行结果
func (l *Logic) updateLastRun(ctx context.Context, status, message string) {
	_, err := l.db.ExecCtx(ctx, `
		UPDATE userfield_sync_config
		SET last_run_at = NOW(), last_status = $1, last_message = $2, updated_at = NOW()
		WHERE id = $3`, status, message, syncConfigID)
	if err != nil {
		log.Printf("⚠️ 更新同步配置状态失败: %v", err)
	}
}

// ParseInterval 解析同步间隔（支持 h/m/s/d）
func ParseInterval(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("同步间隔不能为空")
	}
	// 支持 d（天）
	if strings.HasSuffix(s, "d") {
		n, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil || n <= 0 {
			return 0, fmt.Errorf("无效的同步间隔: %s", s)
		}
		return time.Duration(n) * 24 * time.Hour, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("无效的同步间隔: %s（示例: 30m、6h、1d）", s)
	}
	return d, nil
}

// ListSyncLogs 查询同步日志（最近 N 条）
func (l *Logic) ListSyncLogs(ctx context.Context, limit int) ([]*model.UserFieldSyncLog, error) {
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	var list []*model.UserFieldSyncLog
	query := `SELECT id, status, trigger_type, total, added, updated,
		COALESCE(message,'') AS message, started_at, completed_at
		FROM userfield_sync_logs ORDER BY id DESC LIMIT $1`
	if err := l.db.QueryRowsCtx(ctx, &list, query, limit); err != nil {
		return nil, fmt.Errorf("查询同步日志失败: %w", err)
	}
	return list, nil
}

// startSyncLog 写入一条 running 日志，返回日志 ID
func (l *Logic) startSyncLog(ctx context.Context, trigger string) int64 {
	var id int64
	err := l.db.QueryRowCtx(ctx, &id, `
		INSERT INTO userfield_sync_logs (status, trigger_type, started_at)
		VALUES ('running', $1, NOW()) RETURNING id`, trigger)
	if err != nil {
		log.Printf("⚠️ 写入同步日志失败: %v", err)
		return 0
	}
	return id
}

// finishSyncLog 更新日志为完成态
func (l *Logic) finishSyncLog(ctx context.Context, id int64, status string, total, added, updated int, message string) {
	if id == 0 {
		return
	}
	if _, err := l.db.ExecCtx(ctx, `
		UPDATE userfield_sync_logs
		SET status=$1, total=$2, added=$3, updated=$4, message=$5, completed_at=NOW()
		WHERE id=$6`, status, total, added, updated, message, id); err != nil {
		log.Printf("⚠️ 更新同步日志失败: %v", err)
	}
}

// SeedSyncConfig 首次启动写入同步配置种子（已存在则不覆盖，页面配置优先）
func (l *Logic) SeedSyncConfig(ctx context.Context, enabled bool, onStartup bool, interval string) error {
	if interval == "" {
		interval = "6h"
	}
	var count int64
	if err := l.db.QueryRowCtx(ctx, &count,
		`SELECT COUNT(*) FROM userfield_sync_config WHERE id = $1`, syncConfigID); err != nil {
		return fmt.Errorf("检查同步配置失败: %w", err)
	}
	if count > 0 {
		return nil
	}
	return l.UpdateSyncConfig(ctx, &model.UserFieldSyncConfig{
		ID: syncConfigID, Enabled: enabled, SyncOnStartup: onStartup, Interval: interval,
	})
}

// StartScheduler 启动后台调度：启动时按配置同步一次，之后按间隔检查。
//
// 说明：运行时以数据库配置（页面可改）为准；进度写入内存（供前端轮询），
// 结果落库（供历史查询）。
func StartScheduler(ctx context.Context, db sqlx.SqlConn, endpoint, serviceToken string) {
	l := NewLogic(db)

	// 读取配置（页面维护）
	cfg, err := l.GetSyncConfig(ctx)
	if err != nil {
		log.Printf("⚠️ 读取用户字段同步配置失败: %v", err)
	}

	// 启动时同步（可选）
	if cfg.SyncOnStartup && endpoint != "" && serviceToken != "" {
		go func() {
			time.Sleep(5 * time.Second) // 等待服务就绪
			runAutoSync(ctx, l, endpoint, serviceToken, "startup")
		}()
	}

	baseTick := time.Minute
	log.Printf("⏰ 用户字段自动同步调度器已启动（每 %s 检查一次配置）", baseTick)
	go func() {
		ticker := time.NewTicker(baseTick)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("⏰ 用户字段自动同步调度器已停止")
				return
			case <-ticker.C:
				if endpoint == "" || serviceToken == "" {
					continue
				}
				cfg, err := l.GetSyncConfig(ctx)
				if err != nil || !cfg.Enabled {
					continue
				}
				d, err := ParseInterval(cfg.Interval)
				if err != nil {
					continue
				}
				if cfg.LastRunAt != nil && time.Since(*cfg.LastRunAt) < d {
					continue
				}
				runAutoSync(ctx, l, endpoint, serviceToken, "auto")
			}
		}
	}()
}

// RunSync 执行一次字段同步（带进度与日志），供手动与自动共用。
//
// 并发保护：已有同步在运行时直接返回错误，避免重复执行。
func (l *Logic) RunSync(ctx context.Context, endpoint, serviceToken, trigger string) (int, int, int, error) {
	if getProgress().Running {
		return 0, 0, 0, fmt.Errorf("已有同步任务在执行中，请稍后再试")
	}

	logID := l.startSyncLog(ctx, trigger)
	setProgress(model.UserFieldSyncProgress{
		Running:   true,
		Status:    "running",
		Message:   "正在从 FlyIAM 同步字段定义...",
		StartedAt: time.Now().Format(time.RFC3339),
		Trigger:   trigger,
	})

	added, updated, total, err := l.SyncFromFlyIAM(ctx, endpoint, serviceToken)
	final := model.UserFieldSyncProgress{
		Running:   false,
		Total:     total,
		Added:     added,
		Updated:   updated,
		StartedAt: time.Now().Format(time.RFC3339),
		Trigger:   trigger,
	}
	if err != nil {
		final.Status = "failed"
		final.Message = err.Error()
		l.finishSyncLog(ctx, logID, "failed", total, added, updated, err.Error())
		l.updateLastRun(ctx, "failed", err.Error())
		log.Printf("❌ 用户字段同步失败: %v", err)
		setProgress(final)
		return added, updated, total, err
	}

	final.Status = "success"
	final.Message = fmt.Sprintf("同步完成：新增 %d，更新 %d，共 %d", added, updated, total)
	l.finishSyncLog(ctx, logID, "success", total, added, updated, final.Message)
	l.updateLastRun(ctx, "success", final.Message)
	log.Printf("✅ %s", final.Message)
	setProgress(final)
	return added, updated, total, nil
}

// runAutoSync 执行一次自动同步
func runAutoSync(ctx context.Context, l *Logic, endpoint, serviceToken, trigger string) {
	_, _, _, _ = l.RunSync(ctx, endpoint, serviceToken, trigger)
}

// GetProgress 返回当前同步进度（供前端轮询）
func (l *Logic) GetProgress() model.UserFieldSyncProgress {
	return getProgress()
}
