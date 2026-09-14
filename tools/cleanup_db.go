package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	connStr := "postgresql://iflyelf:1q23l@Yc45j@10.0.51.88:6000/consul_mgr?sslmode=disable"
	
	// 连接数据库
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("❌ 连接数据库失败: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("❌ 数据库 Ping 失败: %v", err)
	}

	fmt.Println("✅ 数据库连接成功")
	fmt.Println("")

	// 读取 SQL 文件
	sqlContent, err := ioutil.ReadFile("deploy/sql/cleanup_and_migrate.sql")
	if err != nil {
		log.Fatalf("❌ 读取 SQL 文件失败: %v", err)
	}

	fmt.Println("开始执行数据库迁移...")
	fmt.Println("")

	// 执行 SQL
	_, err = db.Exec(string(sqlContent))
	if err != nil {
		log.Fatalf("❌ 执行 SQL 失败: %v", err)
	}

	fmt.Println("✅ 数据库迁移完成！")
	fmt.Println("")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("当前数据库表：")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 查询表列表
	rows, err := db.Query(`
		SELECT 
			table_name,
			pg_size_pretty(pg_total_relation_size(quote_ident(table_name)::regclass)) as size
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`)
	if err != nil {
		log.Fatalf("❌ 查询表失败: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tableName, size string
		rows.Scan(&tableName, &size)
		fmt.Printf("  ✓ %-30s %s\n", tableName, size)
	}

	fmt.Println("")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("下一步：")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("1. 启动 Casdoor：")
	fmt.Println("   docker-compose -f docker-compose.casdoor-external-db.yml up -d casdoor")
	fmt.Println("")
	fmt.Println("2. 访问 Casdoor 管理界面：")
	fmt.Println("   http://localhost:8000")
	fmt.Println("   默认账号: admin / 123")
	fmt.Println("")
	fmt.Println("3. 配置 Casdoor（参考 QUICK_START_CASDOOR.md）")
	fmt.Println("")
}
