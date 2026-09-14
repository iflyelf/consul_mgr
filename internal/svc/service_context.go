package svc

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"github.com/iflyelf/consul_mgr/internal/config"
	"github.com/iflyelf/consul_mgr/internal/pkg/consul"
	"github.com/iflyelf/consul_mgr/internal/pkg/jwt"
	"github.com/iflyelf/consul_mgr/internal/pkg/password"
)

// ServiceContext 服务上下文
type ServiceContext struct {
	Config        config.Config
	DB            sqlx.SqlConn
	JWTManager    *jwt.JWTManager
	ConsulManager *consul.Manager
}

// NewServiceContext 创建服务上下文
func NewServiceContext(c config.Config) *ServiceContext {
	// 初始化数据库连接
	db := initDB(c)
	
	// 初始化数据库表和数据
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
	
	return &ServiceContext{
		Config:        c,
		DB:            sqlx.NewSqlConnFromDB(db),
		JWTManager:    jwtManager,
		ConsulManager: consulManager,
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
	// 执行 schema.sql
	schemaSQL := `
		-- 检查表是否存在
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'users'
		);
	`
	
	var exists bool
	if err := db.QueryRow(schemaSQL).Scan(&exists); err != nil {
		return fmt.Errorf("检查表失败: %w", err)
	}
	
	if !exists {
		log.Println("首次启动，初始化数据库表结构...")
		// 这里简化处理，实际应该读取 SQL 文件执行
		// 为了简化，直接在代码中执行必要的建表语句
		if err := executeSchemaSQL(db); err != nil {
			return err
		}
		
		log.Println("初始化权限和角色数据...")
		if err := executeInitDataSQL(db); err != nil {
			return err
		}
		
		log.Println("创建管理员账号...")
		if err := createAdminUser(db, c); err != nil {
			return err
		}
		
		log.Println("数据库初始化完成")
	} else {
		log.Println("数据库表已存在，跳过初始化")
		
		// 确保角色数据存在
		if err := ensureRolesExist(db); err != nil {
			return err
		}
		
		// 检查管理员账号是否存在
		if err := ensureAdminExists(db, c); err != nil {
			return err
		}
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

// createAdminUser 创建管理员账号
func createAdminUser(db *sql.DB, c config.Config) error {
	if c.Admin.Password == "" {
		return fmt.Errorf("管理员密码未设置，请设置环境变量 ADMIN_PASSWORD")
	}
	
	// 生成密码哈希
	hashedPassword, err := password.Hash(c.Admin.Password)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}
	
	// 插入管理员用户
	_, err = db.Exec(`
		INSERT INTO users (username, password, email, real_name, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (username) DO UPDATE 
		SET password = EXCLUDED.password, email = EXCLUDED.email, updated_at = CURRENT_TIMESTAMP
	`, c.Admin.Username, hashedPassword, c.Admin.Email, c.Admin.Username)
	
	if err != nil {
		return fmt.Errorf("创建管理员用户失败: %w", err)
	}
	
	// 分配超级管理员角色
	_, err = db.Exec(`
		INSERT INTO user_roles (user_id, role_id)
		SELECT 
			(SELECT id FROM users WHERE username = $1),
			(SELECT id FROM roles WHERE code = 'admin')
		ON CONFLICT (user_id, role_id) DO NOTHING
	`, c.Admin.Username)
	
	if err != nil {
		return fmt.Errorf("分配管理员角色失败: %w", err)
	}
	
	log.Printf("管理员账号创建成功: %s", c.Admin.Username)
	return nil
}

// ensureAdminExists 确保管理员账号存在
func ensureAdminExists(db *sql.DB, c config.Config) error {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM users WHERE username = $1`, c.Admin.Username).Scan(&count)
	if err != nil {
		return err
	}
	
	if count == 0 {
		log.Println("管理员账号不存在，正在创建...")
		return createAdminUser(db, c)
	}
	
	log.Printf("管理员账号已存在: %s", c.Admin.Username)
	return nil
}

// ensureRolesExist 确保角色数据存在
func ensureRolesExist(db *sql.DB) error {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM roles`).Scan(&count)
	if err != nil {
		return err
	}
	
	if count == 0 {
		log.Println("角色数据不存在，正在创建...")
		return executeInitDataSQL(db)
	}
	
	log.Printf("角色数据已存在，共 %d 个角色", count)
	return nil
}
