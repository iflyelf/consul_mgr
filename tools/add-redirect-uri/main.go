// Package main Casdoor 回调白名单工具
//
// 功能：
//   将指定域名/IP 的回调地址加入 Casdoor 应用的白名单，便于跨域名/多机部署。
//
// 用法：
//   export DATABASE_URL="postgresql://user:pass@host:5432/consul_mgr?sslmode=disable"
//   export CASDOOR_APPLICATION="app-built-in"      # 可选，默认 app-built-in
//   go run ./tools/add-redirect-uri http://10.0.88.88:8080 https://consul.example.com
//
// 说明：
//   参数可以是「源」(如 http://host:port)，工具会自动补 /callback；
//   也可以直接传完整回调地址。
//
// 作者: iflyelf
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "错误: 环境变量 DATABASE_URL 未设置")
		os.Exit(1)
	}

	appName := os.Getenv("CASDOOR_APPLICATION")
	if appName == "" {
		appName = "app-built-in"
	}

	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法: go run ./tools/add-redirect-uri <源或回调地址> [...]")
		os.Exit(1)
	}

	// 规范化：源地址补 /callback
	want := make([]string, 0, len(args)*2)
	for _, a := range args {
		a = strings.TrimRight(a, "/")
		if strings.Contains(a, "/callback") {
			want = append(want, a)
		} else {
			want = append(want, a+"/callback")
		}
		// 同时兼容后端 /api/auth/callback 形式
		want = append(want, a+"/api/auth/callback")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "连接数据库失败:", err)
		os.Exit(1)
	}
	defer db.Close()

	var cur string
	if err := db.QueryRow(`SELECT redirect_uris FROM casdoor_application WHERE name=$1`, appName).Scan(&cur); err != nil {
		fmt.Fprintf(os.Stderr, "查询应用 %s 失败: %v\n", appName, err)
		os.Exit(1)
	}

	var list []string
	_ = json.Unmarshal([]byte(cur), &list)

	have := map[string]bool{}
	for _, u := range list {
		have[u] = true
	}
	added := 0
	for _, u := range want {
		if !have[u] {
			list = append(list, u)
			have[u] = true
			added++
		}
	}

	out, _ := json.Marshal(list)
	if _, err := db.Exec(`UPDATE casdoor_application SET redirect_uris=$1 WHERE name=$2`, string(out), appName); err != nil {
		fmt.Fprintln(os.Stderr, "更新失败:", err)
		os.Exit(1)
	}

	fmt.Printf("应用: %s\n", appName)
	fmt.Printf("新增: %d 条\n", added)
	fmt.Printf("当前白名单: %s\n", string(out))
}
