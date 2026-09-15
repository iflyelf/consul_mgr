package svc

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"github.com/iflyelf/consul_mgr/internal/config"
	"github.com/iflyelf/consul_mgr/internal/pkg/casdoor"
	"github.com/iflyelf/consul_mgr/internal/pkg/consul"
	"github.com/iflyelf/consul_mgr/internal/pkg/jwt"
)

// ServiceContext 服务上下文
type ServiceContext struct {
	Config         config.Config
	DB             sqlx.SqlConn
	JWTManager     *jwt.JWTManager
	ConsulManager  *consul.Manager
	CasdoorClient  *casdoor.Client
}

// NewServiceContext 创建服务上下文
func NewServiceContext(c config.Config) *ServiceContext {
	// 初始化数据库连接
	db := initDB(c)
	
	// 初始化数据库表（不再创建用户表，由 Casdoor 管理）
	if err := initSchema(db, c); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	
	// 初始化 JWT 管理器
	jwtManager := jwt.NewJWTManager(
		c.JWT.Secret,
		c.JWT.AccessExpire,
		c.JWT.RefreshExpire,
		c.JWT.Issuer,
	)
	
	// 初始化 Consul 管理器
	consulManager := consul.NewManager()
	
	// 初始化 Casdoor 客户端
	casdoorClient, err := initCasdoorClient(c)
	if err != nil {
		log.Fatalf("初始化 Casdoor 客户端失败: %v", err)
	}
	
	return &ServiceContext{
		Config:         c,
		DB:             sqlx.NewSqlConnFromDB(db),
		JWTManager:     jwtManager,
		ConsulManager:  consulManager,
		CasdoorClient:  casdoorClient,
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

// initSchema 初始化数据库表结构和数据
func initSchema(db *sql.DB, c config.Config) error {
	// 检查 service_groups 表是否存在
	schemaSQL := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'service_groups'
		);
	`
	
	var exists bool
	if err := db.QueryRow(schemaSQL).Scan(&exists); err != nil {
		return fmt.Errorf("检查表失败: %w", err)
	}
	
	if !exists {
		log.Println("首次启动，初始化数据库表结构...")
		// 创建服务组相关表
		if err := createServiceGroupTables(db); err != nil {
			return err
		}
		log.Println("数据库初始化完成")
	} else {
		log.Println("数据库表已存在，跳过初始化")
	}
	
	return nil
}

// executeSchemaSQL 执行表结构创建
func executeSchemaSQL(db *sql.DB) error {
	// 读取并执行 schema.sql
	schemas := []string{
		// 1. users 表
		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username VARCHAR(50) UNIQUE NOT NULL,
			password VARCHAR(255) NOT NULL,
			email VARCHAR(100),
			real_name VARCHAR(50),
			status SMALLINT DEFAULT 1,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_login_at TIMESTAMP,
			last_login_ip VARCHAR(50)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)`,
		`CREATE INDEX IF NOT EXISTS idx_users_status ON users(status)`,
		
		// 2. roles 表
		`CREATE TABLE IF NOT EXISTS roles (
			id SERIAL PRIMARY KEY,
			name VARCHAR(50) UNIQUE NOT NULL,
			code VARCHAR(50) UNIQUE NOT NULL,
			description TEXT,
			status SMALLINT DEFAULT 1,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_roles_code ON roles(code)`,
		
		// 3. permissions 表
		`CREATE TABLE IF NOT EXISTS permissions (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			code VARCHAR(100) UNIQUE NOT NULL,
			resource VARCHAR(255),
			action VARCHAR(20),
			description TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_permissions_code ON permissions(code)`,
		`CREATE INDEX IF NOT EXISTS idx_permissions_resource ON permissions(resource)`,
		
		// 4. user_roles 表
		`CREATE TABLE IF NOT EXISTS user_roles (
			id SERIAL PRIMARY KEY,
			user_id INT NOT NULL,
			role_id INT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, role_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_user_roles_user ON user_roles(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_roles_role ON user_roles(role_id)`,
		
		// 5. role_permissions 表
		`CREATE TABLE IF NOT EXISTS role_permissions (
			id SERIAL PRIMARY KEY,
			role_id INT NOT NULL,
			permission_id INT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(role_id, permission_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_role_permissions_role ON role_permissions(role_id)`,
		`CREATE INDEX IF NOT EXISTS idx_role_permissions_perm ON role_permissions(permission_id)`,
		
		// 6. service_groups 表
		`CREATE TABLE IF NOT EXISTS service_groups (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) UNIQUE NOT NULL,
			code VARCHAR(100) UNIQUE NOT NULL,
			description TEXT,
			consul_address VARCHAR(255) NOT NULL,
			consul_token VARCHAR(255),
			consul_datacenter VARCHAR(50) DEFAULT 'dc1',
			status SMALLINT DEFAULT 1,
			created_by INT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_service_groups_code ON service_groups(code)`,
		`CREATE INDEX IF NOT EXISTS idx_service_groups_status ON service_groups(status)`,
		
		// 7. role_group_permissions 表
		`CREATE TABLE IF NOT EXISTS role_group_permissions (
			id SERIAL PRIMARY KEY,
			role_id INT NOT NULL,
			group_id INT NOT NULL,
			permissions JSONB NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(role_id, group_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_role_group_role ON role_group_permissions(role_id)`,
		`CREATE INDEX IF NOT EXISTS idx_role_group_group ON role_group_permissions(group_id)`,
		
		// 8. audit_logs 表
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id BIGSERIAL PRIMARY KEY,
			user_id INT,
			username VARCHAR(50),
			action VARCHAR(100) NOT NULL,
			resource_type VARCHAR(50),
			resource_id VARCHAR(255),
			resource_name VARCHAR(255),
			group_id INT,
			details JSONB,
			ip_address VARCHAR(50),
			user_agent TEXT,
			status VARCHAR(20),
			error_message TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_user ON audit_logs(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs(created_at DESC)`,
	}
	
	for i, schema := range schemas {
		if _, err := db.Exec(schema); err != nil {
			return fmt.Errorf("执行建表语句 %d 失败: %w", i+1, err)
		}
	}
	
	log.Println("数据库表结构创建成功")
	return nil
}

// executeInitDataSQL 执行初始数据插入
func executeInitDataSQL(db *sql.DB) error {
	// 插入默认角色
	roles := []struct {
		name        string
		code        string
		description string
	}{
		{"超级管理员", "admin", "拥有所有权限的超级管理员"},
		{"运维人员", "operator", "可以管理Consul服务和实例"},
		{"只读用户", "viewer", "只能查看信息"},
	}
	
	for _, role := range roles {
		_, err := db.Exec(`
			INSERT INTO roles (name, code, description, status, created_at, updated_at)
			VALUES ($1, $2, $3, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
			ON CONFLICT (code) DO NOTHING
		`, role.name, role.code, role.description)
		if err != nil {
			return fmt.Errorf("插入角色失败: %w", err)
		}
	}
	
	log.Println("初始角色数据创建成功")
	return nil
}

// initCasdoorClient 初始化 Casdoor 客户端
func initCasdoorClient(c config.Config) (*casdoor.Client, error) {
	// 构造 Casdoor 配置
	casdoorConfig := &casdoor.Config{
		Endpoint:         c.Casdoor.Endpoint,
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
		consul_address VARCHAR(255) NOT NULL,
		consul_token VARCHAR(255),
		datacenter VARCHAR(50) DEFAULT 'dc1',
		description TEXT,
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);

	-- 服务组用户权限表
	CREATE TABLE IF NOT EXISTS service_group_users (
		id BIGSERIAL PRIMARY KEY,
		group_id BIGINT NOT NULL,
		user_id VARCHAR(100) NOT NULL,
		permissions TEXT[] DEFAULT '{}',
		created_at TIMESTAMP DEFAULT NOW(),
		FOREIGN KEY (group_id) REFERENCES service_groups(id) ON DELETE CASCADE,
		UNIQUE(group_id, user_id)
	);

	-- 服务组角色权限表
	CREATE TABLE IF NOT EXISTS service_group_roles (
		id BIGSERIAL PRIMARY KEY,
		group_id BIGINT NOT NULL,
		role_name VARCHAR(100) NOT NULL,
		permissions TEXT[] DEFAULT '{}',
		created_at TIMESTAMP DEFAULT NOW(),
		FOREIGN KEY (group_id) REFERENCES service_groups(id) ON DELETE CASCADE,
		UNIQUE(group_id, role_name)
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
	CREATE INDEX IF NOT EXISTS idx_service_group_users_group ON service_group_users(group_id);
	CREATE INDEX IF NOT EXISTS idx_service_group_users_user ON service_group_users(user_id);
	CREATE INDEX IF NOT EXISTS idx_service_group_roles_group ON service_group_roles(group_id);
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
