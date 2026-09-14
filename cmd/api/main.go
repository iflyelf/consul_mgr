package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"

	"github.com/iflyelf/consul_mgr/internal/config"
	authHandler "github.com/iflyelf/consul_mgr/internal/handler/auth"
	"github.com/iflyelf/consul_mgr/internal/handler/group"
	"github.com/iflyelf/consul_mgr/internal/handler/instance"
	"github.com/iflyelf/consul_mgr/internal/handler/service"
	"github.com/iflyelf/consul_mgr/internal/middleware"
	"github.com/iflyelf/consul_mgr/internal/svc"
)

var (
	configFile = flag.String("c", "etc/config.yaml", "配置文件路径")
	version    = "dev"
)

func main() {
	flag.Parse()

	// 加载配置
	var c config.Config
	conf.MustLoad(*configFile, &c)

	// 创建服务上下文
	ctx := svc.NewServiceContext(c)
	
	// 创建 REST 服务器
	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	// 注册路由
	registerHandlers(server, ctx)

	// 启动信息
	fmt.Printf("🚀 Starting Consul Manager Server\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("Version:       %s\n", version)
	fmt.Printf("Listen:        %s:%d\n", c.Host, c.Port)
	fmt.Printf("Casdoor:       %s\n", c.Casdoor.Endpoint)
	fmt.Printf("Organization:  %s\n", c.Casdoor.OrganizationName)
	fmt.Printf("Application:   %s\n", c.Casdoor.ApplicationName)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("Health Check:  http://%s:%d/health\n", c.Host, c.Port)
	fmt.Printf("Login URL:     http://%s:%d/api/auth/login\n", c.Host, c.Port)
	fmt.Printf("Web UI:        http://%s:%d/\n", c.Host, c.Port)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	
	server.Start()
}

// registerHandlers 注册所有路由
func registerHandlers(server *rest.Server, ctx *svc.ServiceContext) {
	// 创建中间件
	casdoorAuth := middleware.NewCasdoorAuthMiddleware(ctx.CasdoorClient)
	permissionMw := middleware.NewPermissionMiddleware(ctx.CasdoorClient)
	
	// ============================================================
	// 公开路由（无需认证）
	// ============================================================
	
	// 健康检查
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/health",
		Handler: healthHandler(),
	})
	
	// 认证相关
	server.AddRoutes([]rest.Route{
		{
			Method:  http.MethodGet,
			Path:    "/api/auth/login",
			Handler: authHandler.LoginHandler(ctx.CasdoorClient),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/auth/callback",
			Handler: authHandler.CallbackHandler(ctx.CasdoorClient),
		},
	})
	
	// ============================================================
	// 需要认证的路由
	// ============================================================
	
	server.AddRoutes(
		[]rest.Route{
			// 认证相关
			{
				Method:  http.MethodGet,
				Path:    "/api/auth/userinfo",
				Handler: authHandler.GetUserInfoHandler(ctx.CasdoorClient),
			},
			{
				Method:  http.MethodPost,
				Path:    "/api/auth/refresh",
				Handler: authHandler.RefreshTokenHandler(ctx.CasdoorClient),
			},
			{
				Method:  http.MethodPost,
				Path:    "/api/auth/logout",
				Handler: authHandler.LogoutHandler(),
			},
		},
		rest.WithJwt(ctx.Config.JWT.Secret), // 使用 JWT 中间件（可选，主要用 Casdoor Token）
		rest.WithPrefix("/"),
	)
	
	// ============================================================
	// 服务组管理（需要权限：consul_group）
	// ============================================================
	
	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodGet,
				Path:    "/api/groups",
				Handler: group.ListGroupsHandler(ctx),
			},
		},
		rest.WithPrefix("/"),
	)
	
	// 使用自定义中间件链
	server.AddRoute(rest.Route{
		Method: http.MethodPost,
		Path:   "/api/groups",
		Handler: casdoorAuth.Handle(
			permissionMw.RequirePermission("consul_group", "write")(
				group.CreateGroupHandler(ctx),
			),
		),
	})
	
	server.AddRoute(rest.Route{
		Method: http.MethodPut,
		Path:   "/api/groups/:id",
		Handler: casdoorAuth.Handle(
			permissionMw.RequirePermission("consul_group", "write")(
				group.UpdateGroupHandler(ctx),
			),
		),
	})
	
	server.AddRoute(rest.Route{
		Method: http.MethodDelete,
		Path:   "/api/groups/:id",
		Handler: casdoorAuth.Handle(
			permissionMw.RequirePermission("consul_group", "delete")(
				group.DeleteGroupHandler(ctx),
			),
		),
	})
	
	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/api/groups/:id",
		Handler: casdoorAuth.Handle(
			group.GetGroupHandler(ctx),
		),
	})
	
	// ============================================================
	// 服务管理（需要权限：consul_service）
	// ============================================================
	
	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/api/services",
		Handler: casdoorAuth.Handle(
			permissionMw.RequireServiceGroupAccess("read")(
				service.ListServicesHandler(ctx),
			),
		),
	})
	
	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/api/services/:name",
		Handler: casdoorAuth.Handle(
			service.GetServiceHandler(ctx),
		),
	})
	
	// ============================================================
	// 实例管理（需要权限：consul_instance）
	// ============================================================
	
	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/api/instances",
		Handler: casdoorAuth.Handle(
			permissionMw.RequireServiceGroupAccess("read")(
				instance.ListInstancesHandler(ctx),
			),
		),
	})
	
	server.AddRoute(rest.Route{
		Method: http.MethodPost,
		Path:   "/api/instances",
		Handler: casdoorAuth.Handle(
			permissionMw.RequireServiceGroupAccess("write")(
				instance.RegisterInstanceHandler(ctx),
			),
		),
	})
	
	server.AddRoute(rest.Route{
		Method: http.MethodDelete,
		Path:   "/api/instances/:id",
		Handler: casdoorAuth.Handle(
			permissionMw.RequireServiceGroupAccess("delete")(
				instance.DeregisterInstanceHandler(ctx),
			),
		),
	})
	
	// ============================================================
	// 静态文件服务（前端）
	// ============================================================
	
	if ctx.Config.Web.Embedded {
		log.Println("启用嵌入式 Web 界面")
		server.AddRoute(rest.Route{
			Method:  http.MethodGet,
			Path:    "/",
			Handler: serveEmbeddedWeb(),
		})
		
		server.AddRoute(rest.Route{
			Method:  http.MethodGet,
			Path:    "/assets/:file",
			Handler: serveEmbeddedAssets(),
		})
	}
}

// healthHandler 健康检查处理器
func healthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","message":"Consul Manager is running"}`))
	}
}

// serveEmbeddedWeb 提供嵌入式 Web 界面
func serveEmbeddedWeb() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 读取嵌入的 index.html
		content, err := webFS.ReadFile("web/dist/index.html")
		if err != nil {
			http.Error(w, "Web UI not found", http.StatusNotFound)
			return
		}
		
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}
}

// serveEmbeddedAssets 提供嵌入式静态资源
func serveEmbeddedAssets() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 从 URL 路径获取文件名
		file := r.URL.Query().Get(":file")
		if file == "" {
			http.Error(w, "File not specified", http.StatusBadRequest)
			return
		}
		
		// 读取嵌入的文件
		path := fmt.Sprintf("web/dist/assets/%s", file)
		content, err := webFS.ReadFile(path)
		if err != nil {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		
		// 设置 Content-Type
		contentType := getContentType(file)
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}
}

// getContentType 根据文件扩展名获取 Content-Type
func getContentType(filename string) string {
	if len(filename) > 3 && filename[len(filename)-3:] == ".js" {
		return "application/javascript"
	}
	if len(filename) > 4 && filename[len(filename)-4:] == ".css" {
		return "text/css"
	}
	if len(filename) > 4 && filename[len(filename)-4:] == ".png" {
		return "image/png"
	}
	if len(filename) > 4 && filename[len(filename)-4:] == ".jpg" {
		return "image/jpeg"
	}
	if len(filename) > 5 && filename[len(filename)-5:] == ".jpeg" {
		return "image/jpeg"
	}
	if len(filename) > 4 && filename[len(filename)-4:] == ".svg" {
		return "image/svg+xml"
	}
	return "application/octet-stream"
}
