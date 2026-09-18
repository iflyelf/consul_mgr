package main

import (
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/iflyelf/consul_mgr/web"
)

// getSPAHandler 返回 SPA 静态文件处理器
func getSPAHandler() http.Handler {
	// 从嵌入的文件系统中获取 dist 子目录
	distFS, err := fs.Sub(web.DistFS, "dist")
	if err != nil {
		panic(err)
	}
	
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		
		// 如果是 API 请求或健康检查，不处理
		if strings.HasPrefix(path, "api/") || path == "health" {
			http.NotFound(w, r)
			return
		}
		
		// 如果路径为空，使用 index.html
		if path == "" {
			path = "index.html"
		}
		
		// 尝试读取文件
		file, err := distFS.Open(path)
		if err != nil {
			// 静态资源（assets/*.js 等带扩展名）不存在时必须 404，不能回退 index.html：
			// 否则浏览器把 HTML 当 JS 执行，动态 import 报
			// "Failed to fetch dynamically imported module"（发版后旧 chunk 已被删除）。
			if isStaticFile(path) {
				http.NotFound(w, r)
				return
			}
			// 前端路由（无扩展名）回退 index.html
			path = "index.html"
			file, err = distFS.Open(path)
			if err != nil {
				http.NotFound(w, r)
				return
			}
		}
		defer file.Close()
		
		// 设置正确的 Content-Type
		if strings.HasSuffix(path, ".js") {
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		} else if strings.HasSuffix(path, ".css") {
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		} else if strings.HasSuffix(path, ".html") {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		} else if strings.HasSuffix(path, ".json") {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
		} else if strings.HasSuffix(path, ".svg") {
			w.Header().Set("Content-Type", "image/svg+xml")
		} else if strings.HasSuffix(path, ".ico") {
			w.Header().Set("Content-Type", "image/x-icon")
		} else if strings.HasSuffix(path, ".png") {
			w.Header().Set("Content-Type", "image/png")
		}
		
		// 缓存头：带 hash 的资源长缓存；index.html 必须每次校验，
		// 避免发版后仍使用旧 HTML（引用已删除的旧 chunk）。
		if strings.HasPrefix(path, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
		}

		// 使用 http.ServeContent 来处理文件服务
		stat, _ := file.Stat()
		http.ServeContent(w, r, path, stat.ModTime(), file.(io.ReadSeeker))
	})
}

// isStaticFile 判断请求路径是否为静态资源文件（带扩展名，如 assets/xxx.js）。
// 这类路径未命中时必须返回 404，不能回退 index.html。
func isStaticFile(p string) bool {
	return strings.HasPrefix(p, "assets/") || path.Ext(p) != ""
}
