package svc

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"github.com/iflyelf/consul_mgr/internal/config"
	"github.com/iflyelf/consul_mgr/internal/pkg/cache"
	"github.com/iflyelf/consul_mgr/internal/pkg/casdoor"
	"github.com/iflyelf/consul_mgr/internal/pkg/consul"
)

// ServiceContext 服务上下文
type ServiceContext struct {
	Config        config.Config
	DB            sqlx.SqlConn
	RawDB         *sql.DB // 原生连接：PostgreSQL 数组(TEXT[])需 pq.Array 扫描
	ConsulManager *consul.Manager
	CasdoorClient *casdoor.Client
	Cache         *cache.Cache
}

// CacheKeyInstances 实例列表缓存键
//
// 说明：service/status 作为维度区分不同查询条件
func (s *ServiceContext) CacheKeyInstances(groupID int64, serviceName, status string) string {
	return fmt.Sprintf("consul_mgr:instances:%d:%s:%s", groupID, serviceName, status)
}

// CacheKeyInstancesByService 单个服务的实例缓存键
//
// 说明：按服务维度缓存原始实例（不含状态过滤），
// 全量列表由各服务缓存合并而来，避免切换筛选条件时全量重拉 Consul。
func (s *ServiceContext) CacheKeyInstancesByService(groupID int64, serviceName string) string {
	return fmt.Sprintf("consul_mgr:instances:%d:svc:%s", groupID, serviceName)
}

// ConsulConcurrency 返回 Consul 批量操作并发度
func (s *ServiceContext) ConsulConcurrency() int {
	if s.Config.Consul.MaxConcurrency > 0 {
		return s.Config.Consul.MaxConcurrency
	}
	return 16
}

// CacheKeyServices 服务列表缓存键
func (s *ServiceContext) CacheKeyServices(groupID int64) string {
	return fmt.Sprintf("consul_mgr:services:%d", groupID)
}

// CacheKeyServiceDetail 服务详情缓存键
func (s *ServiceContext) CacheKeyServiceDetail(groupID int64, serviceName string) string {
	return fmt.Sprintf("consul_mgr:service_detail:%d:%s", groupID, serviceName)
}

// InvalidateInstances 失效指定服务组的实例缓存
func (s *ServiceContext) InvalidateInstances(ctx context.Context, groupID int64) {
	s.Cache.DelPrefix(ctx, fmt.Sprintf("consul_mgr:instances:%d:", groupID))
}

// InvalidateServices 失效指定服务组的服务列表缓存
func (s *ServiceContext) InvalidateServices(ctx context.Context, groupID int64) {
	s.Cache.DelPrefix(ctx, fmt.Sprintf("consul_mgr:services:%d", groupID))
}

// InvalidateServiceDetails 失效指定服务组的服务详情缓存
func (s *ServiceContext) InvalidateServiceDetails(ctx context.Context, groupID int64) {
	s.Cache.DelPrefix(ctx, fmt.Sprintf("consul_mgr:service_detail:%d:", groupID))
}

// InvalidateGroupCaches 失效指定服务组的全部缓存
func (s *ServiceContext) InvalidateGroupCaches(ctx context.Context, groupID int64) {
	s.InvalidateInstances(ctx, groupID)
	s.InvalidateServices(ctx, groupID)
	s.InvalidateServiceDetails(ctx, groupID)
}

// GetConsulClient 获取指定服务组的 Consul 客户端
//
// 功能:
//  1. 从数据库读取服务组配置
//  2. 自动检测数据中心，若与配置不符则自动更新数据库并重建客户端
//
// 参数:
//   ctx     - 上下文
//   groupID - 服务组 ID
//
// 返回:
//   *consul.Client - Consul 客户端
//   error - 错误信息
func (s *ServiceContext) GetConsulClient(ctx context.Context, groupID int64) (*consul.Client, error) {
	query := `SELECT consul_address, consul_token, consul_datacenter FROM service_groups WHERE id = $1`

	var group struct {
		ConsulAddress    string `db:"consul_address"`
		ConsulToken      string `db:"consul_token"`
		ConsulDatacenter string `db:"consul_datacenter"`
	}

	if err := s.DB.QueryRowCtx(ctx, &group, query, groupID); err != nil {
		return nil, fmt.Errorf("查询服务组失败: %w", err)
	}

	cfg := &consul.Config{
		Address:    group.ConsulAddress,
		Token:      group.ConsulToken,
		Datacenter: group.ConsulDatacenter,
	}

	// 直接使用数据库中已保存的数据中心。
	//
	// 说明:
	//   数据中心在「创建/更新服务组」时由 GroupLogic.DetectDatacenter 探测并落库，
	//   也可通过 POST /api/groups/detect-datacenter 手动触发。
	//   此处绝不能再探测：Agent().Self() 属于 Agent API，在只读网关/负载均衡后
	//   可能不可达并卡满超时，且每次列表请求都多一次 Consul 往返，
	//   会导致查询极慢（实测单请求被拖到 10s）。
	client, err := s.ConsulManager.GetClient(groupID, cfg)
	if err != nil {
		return nil, fmt.Errorf("创建 Consul 客户端失败: %w", err)
	}

	return client, nil
}

// NewServiceContext 创建服务上下文
func NewServiceContext(c config.Config) *ServiceContext {
	// 初始化数据库连接
	db := initDB(c)
	
	// 初始化数据库表（不创建用户表，认证由 Casdoor 负责）
	if err := initSchema(db, c); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	// 初始化 Consul 管理器
	consulManager := consul.NewManager()
	
	// 初始化 Casdoor 客户端
	casdoorClient, err := initCasdoorClient(c)
	if err != nil {
		log.Fatalf("初始化 Casdoor 客户端失败: %v", err)
	}

	// 初始化 Redis 缓存（连接失败自动降级，不影响启动）
	cacheClient := cache.New(cache.Config{
		Enabled:  c.Redis.Enabled,
		Host:     c.Redis.Host,
		Port:     c.Redis.Port,
		Password: c.Redis.Password,
		DB:       c.Redis.DB,
		TTL:      c.Redis.TTL,
	})

	return &ServiceContext{
		Config:        c,
		DB:            sqlx.NewSqlConnFromDB(db),
		RawDB:         db,
		ConsulManager: consulManager,
		CasdoorClient: casdoorClient,
		Cache:         cacheClient,
	}
}

// initDB 初始化数据库连接
func initDB(c config.Config) *sql.DB {
	var dsn string
	if c.Database.DSN != "" {
		dsn = c.Database.DSN
	} else {
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			c.Database.Host,
			c.Database.Port,
			c.Database.User,
			c.Database.Password,
			c.Database.DBName,
			c.Database.SSLMode,
		)
	}
	
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	
	db.SetMaxOpenConns(c.Database.MaxOpenConns)
	db.SetMaxIdleConns(c.Database.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(c.Database.ConnMaxLifetime) * time.Second)
	
	if err := db.Ping(); err != nil {
		log.Fatalf("数据库连接测试失败: %v", err)
	}
	
	log.Println("数据库连接成功")
	return db
}

// initSchema 初始化数据库表结构
//
// 说明:
//   - 使用 CREATE TABLE IF NOT EXISTS，可重复执行且幂等
//   - 每次启动都执行，确保新增的表/索引能自动创建
func initSchema(db *sql.DB, c config.Config) error {
	log.Println("初始化数据库表结构...")
	if err := createServiceGroupTables(db); err != nil {
		return err
	}
	log.Println("数据库初始化完成")
	return nil
}

// initCasdoorClient 初始化 Casdoor 客户端
func initCasdoorClient(c config.Config) (*casdoor.Client, error) {
	// 构造 Casdoor 配置
	casdoorConfig := &casdoor.Config{
		Endpoint:         c.Casdoor.Endpoint,
		PublicEndpoint:   c.Casdoor.PublicEndpoint,
		ClientId:         c.Casdoor.ClientId,
		ClientSecret:     c.Casdoor.ClientSecret,
		Certificate:      c.Casdoor.Certificate,
		OrganizationName: c.Casdoor.OrganizationName,
		ApplicationName:  c.Casdoor.ApplicationName,
	}
	
	// 创建 Casdoor 客户端
	client, err := casdoor.NewClient(casdoorConfig)
	if err != nil {
		return nil, fmt.Errorf("创建 Casdoor 客户端失败: %w", err)
	}
	
	log.Printf("Casdoor 客户端初始化成功: %s", c.Casdoor.Endpoint)
	return client, nil
}

// createServiceGroupTables 创建服务组相关表
func createServiceGroupTables(db *sql.DB) error {
	schema := `
	-- 服务组表
	CREATE TABLE IF NOT EXISTS service_groups (
		id BIGSERIAL PRIMARY KEY,
		name VARCHAR(100) NOT NULL UNIQUE,
		code VARCHAR(100) NOT NULL DEFAULT '',
		consul_address VARCHAR(255) NOT NULL,
		consul_token VARCHAR(255),
		consul_datacenter VARCHAR(50) DEFAULT 'dc1',
		description TEXT,
		status SMALLINT DEFAULT 1,
		created_by VARCHAR(100),
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);

	-- 兼容旧库：补齐可能缺失的列
	ALTER TABLE service_groups ADD COLUMN IF NOT EXISTS code VARCHAR(100) NOT NULL DEFAULT '';
	ALTER TABLE service_groups ADD COLUMN IF NOT EXISTS consul_datacenter VARCHAR(50) DEFAULT 'dc1';
	ALTER TABLE service_groups ADD COLUMN IF NOT EXISTS status SMALLINT DEFAULT 1;
	ALTER TABLE service_groups ADD COLUMN IF NOT EXISTS created_by VARCHAR(100);
	ALTER TABLE service_groups ADD COLUMN IF NOT EXISTS datacenter VARCHAR(50);

	-- 服务组用户权限表
	CREATE TABLE IF NOT EXISTS service_group_users (
		id BIGSERIAL PRIMARY KEY,
		group_id BIGINT NOT NULL,
		user_id VARCHAR(100) NOT NULL,
		username VARCHAR(100) NOT NULL DEFAULT '',
		permissions TEXT[] DEFAULT '{read}',
		created_by VARCHAR(100),
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW(),
		FOREIGN KEY (group_id) REFERENCES service_groups(id) ON DELETE CASCADE,
		UNIQUE(group_id, user_id)
	);

	-- 兼容旧库：补齐可能缺失的列，并为 username 提供默认值
	-- （历史库里 username 为 NOT NULL 且无默认值，会导致插入失败）
	ALTER TABLE service_group_users ADD COLUMN IF NOT EXISTS username VARCHAR(100);
	ALTER TABLE service_group_users ALTER COLUMN username SET DEFAULT '';
	ALTER TABLE service_group_users ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT NOW();
	ALTER TABLE service_group_users ADD COLUMN IF NOT EXISTS created_by VARCHAR(100);
	ALTER TABLE service_group_users ALTER COLUMN permissions SET DEFAULT '{read}';

	-- 服务组角色权限表（历史表：按服务组+角色名授权，保留兼容）
	CREATE TABLE IF NOT EXISTS service_group_roles (
		id BIGSERIAL PRIMARY KEY,
		group_id BIGINT NOT NULL,
		role_name VARCHAR(100) NOT NULL,
		permissions TEXT[] DEFAULT '{read}',
		created_by VARCHAR(100),
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW(),
		FOREIGN KEY (group_id) REFERENCES service_groups(id) ON DELETE CASCADE,
		UNIQUE(group_id, role_name)
	);

	ALTER TABLE service_group_roles ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT NOW();
	ALTER TABLE service_group_roles ADD COLUMN IF NOT EXISTS created_by VARCHAR(100);
	ALTER TABLE service_group_roles ALTER COLUMN permissions SET DEFAULT '{read}';

	-- ============================================================
	-- 人员组织：角色 / 团队 / 团队成员 / 团队-服务组授权
	-- ============================================================

	-- 角色：可复用的权限集合（read/write/delete）
	CREATE TABLE IF NOT EXISTS roles (
		id BIGSERIAL PRIMARY KEY,
		name VARCHAR(100) NOT NULL UNIQUE,
		code VARCHAR(100) NOT NULL DEFAULT '',
		description TEXT,
		permissions TEXT[] DEFAULT '{}',
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);

	-- 团队
	CREATE TABLE IF NOT EXISTS teams (
		id BIGSERIAL PRIMARY KEY,
		name VARCHAR(100) NOT NULL UNIQUE,
		code VARCHAR(100) NOT NULL DEFAULT '',
		description TEXT,
		status SMALLINT DEFAULT 1,
		created_by VARCHAR(100),
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);

	-- 团队成员（user_id 为 Casdoor 用户 ID）
	CREATE TABLE IF NOT EXISTS team_members (
		id BIGSERIAL PRIMARY KEY,
		team_id BIGINT NOT NULL,
		user_id VARCHAR(100) NOT NULL,
		username VARCHAR(100),
		display_name VARCHAR(200),
		created_at TIMESTAMP DEFAULT NOW(),
		FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
		UNIQUE(team_id, user_id)
	);

	-- 团队-服务组授权：permissions 为直接权限，role_ids 为引用角色的权限（取并集）
	CREATE TABLE IF NOT EXISTS team_group_permissions (
		id BIGSERIAL PRIMARY KEY,
		team_id BIGINT NOT NULL,
		group_id BIGINT NOT NULL,
		permissions TEXT[] DEFAULT '{}',
		role_ids BIGINT[] DEFAULT '{}',
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW(),
		FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
		FOREIGN KEY (group_id) REFERENCES service_groups(id) ON DELETE CASCADE,
		UNIQUE(team_id, group_id)
	);

	-- Consul 实例持久化表
	CREATE TABLE IF NOT EXISTS consul_instances (
		id BIGSERIAL PRIMARY KEY,
		instance_id VARCHAR(255) NOT NULL,
		service_name VARCHAR(100) NOT NULL,
		group_id BIGINT NOT NULL,
		address VARCHAR(100) NOT NULL,
		port INT NOT NULL,
		tags TEXT[] DEFAULT '{}',
		meta JSONB,
		health_check JSONB,
		status VARCHAR(20) DEFAULT 'passing',
		datacenter VARCHAR(50) DEFAULT 'dc1',
		node_name VARCHAR(100),
		created_by VARCHAR(100),
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW(),
		FOREIGN KEY (group_id) REFERENCES service_groups(id) ON DELETE CASCADE,
		UNIQUE(group_id, instance_id)
	);

	-- 审计日志表
	CREATE TABLE IF NOT EXISTS audit_logs (
		id BIGSERIAL PRIMARY KEY,
		user_id VARCHAR(100),
		username VARCHAR(100),
		action VARCHAR(50),
		resource_type VARCHAR(50),
		resource_id VARCHAR(100),
		resource_name VARCHAR(255),
		group_id BIGINT,
		details JSONB,
		ip_address VARCHAR(50),
		user_agent TEXT,
		status VARCHAR(20),
		error_message TEXT,
		created_at TIMESTAMP DEFAULT NOW()
	);

	-- 创建索引
	CREATE INDEX IF NOT EXISTS idx_service_groups_code ON service_groups(code);
	CREATE INDEX IF NOT EXISTS idx_service_group_users_group ON service_group_users(group_id);
	CREATE INDEX IF NOT EXISTS idx_service_group_users_user ON service_group_users(user_id);
	CREATE INDEX IF NOT EXISTS idx_service_group_roles_group ON service_group_roles(group_id);
	CREATE INDEX IF NOT EXISTS idx_teams_code ON teams(code);
	CREATE INDEX IF NOT EXISTS idx_team_members_team ON team_members(team_id);
	CREATE INDEX IF NOT EXISTS idx_team_members_user ON team_members(user_id);
	CREATE INDEX IF NOT EXISTS idx_team_group_team ON team_group_permissions(team_id);
	CREATE INDEX IF NOT EXISTS idx_team_group_group ON team_group_permissions(group_id);
	CREATE INDEX IF NOT EXISTS idx_roles_code ON roles(code);
	CREATE INDEX IF NOT EXISTS idx_consul_instances_group ON consul_instances(group_id);
	CREATE INDEX IF NOT EXISTS idx_consul_instances_service ON consul_instances(service_name);
	CREATE INDEX IF NOT EXISTS idx_consul_instances_status ON consul_instances(status);
	CREATE INDEX IF NOT EXISTS idx_consul_instances_created ON consul_instances(created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_consul_instances_datacenter ON consul_instances(datacenter);
	CREATE INDEX IF NOT EXISTS idx_consul_instances_meta ON consul_instances USING GIN (meta);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_user ON audit_logs(user_id);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs(created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_group ON audit_logs(group_id);
	`
	
	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("创建表失败: %w", err)
	}
	
	log.Println("服务组和实例相关表创建成功")
	return nil
}
