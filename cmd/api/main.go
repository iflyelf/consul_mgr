package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"

	"github.com/iflyelf/consul_mgr/internal/config"
	auditHandler "github.com/iflyelf/consul_mgr/internal/handler/audit"
	authHandler "github.com/iflyelf/consul_mgr/internal/handler/auth"
	"github.com/iflyelf/consul_mgr/internal/handler/group"
	"github.com/iflyelf/consul_mgr/internal/handler/instance"
	permissionHandler "github.com/iflyelf/consul_mgr/internal/handler/permission"
	"github.com/iflyelf/consul_mgr/internal/middleware"
	"github.com/iflyelf/consul_mgr/internal/svc"
)

var (
	configFile = flag.String("c", "etc/config.yaml", "配置文件路径")
	version    = "1.0.0"
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
	auditMw := middleware.NewAuditMiddleware(ctx)
	
	// ============================================================
	// 公开路由（无需认证）
	// ============================================================
	
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/health",
		Handler: healthHandler(),
	})
	
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
	
	// 认证相关
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/api/auth/userinfo",
		Handler: casdoorAuth.Handle(authHandler.GetUserInfoHandler(ctx.CasdoorClient)),
	})
	
	server.AddRoute(rest.Route{
		Method:  http.MethodPost,
		Path:    "/api/auth/refresh",
		Handler: casdoorAuth.Handle(authHandler.RefreshTokenHandler(ctx.CasdoorClient)),
	})
	
	server.AddRoute(rest.Route{
		Method:  http.MethodPost,
		Path:    "/api/auth/logout",
		Handler: casdoorAuth.Handle(authHandler.LogoutHandler()),
	})
	
	// ============================================================
	// 服务组管理（需要权限：consul_group）
	// ============================================================
	
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/api/groups",
		Handler: casdoorAuth.Handle(group.ListGroupsHandler(ctx)),
	})
	
	server.AddRoute(rest.Route{
		Method: http.MethodPost,
		Path:   "/api/groups",
		Handler: casdoorAuth.Handle(
			auditMw.Handle(
				permissionMw.RequirePermission("consul_group", "write")(
					group.CreateGroupHandler(ctx),
				),
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
	
	server.AddRoute(rest.Route{
		Method: http.MethodPut,
		Path:   "/api/groups/:id",
		Handler: casdoorAuth.Handle(
			auditMw.Handle(
				permissionMw.RequirePermission("consul_group", "write")(
					group.UpdateGroupHandler(ctx),
				),
			),
		),
	})
	
	server.AddRoute(rest.Route{
		Method: http.MethodDelete,
		Path:   "/api/groups/:id",
		Handler: casdoorAuth.Handle(
			auditMw.Handle(
				permissionMw.RequirePermission("consul_group", "delete")(
					group.DeleteGroupHandler(ctx),
				),
			),
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
		Method: http.MethodGet,
		Path:   "/api/instances/:id",
		Handler: casdoorAuth.Handle(
			instance.GetInstanceHandler(ctx),
		),
	})
	
	server.AddRoute(rest.Route{
		Method: http.MethodPost,
		Path:   "/api/instances",
		Handler: casdoorAuth.Handle(
			auditMw.Handle(
				permissionMw.RequireServiceGroupAccess("write")(
					instance.RegisterInstanceHandler(ctx),
				),
			),
		),
	})
	
	server.AddRoute(rest.Route{
		Method: http.MethodPut,
		Path:   "/api/instances/:id",
		Handler: casdoorAuth.Handle(
			auditMw.Handle(
				permissionMw.RequireServiceGroupAccess("write")(
					instance.UpdateInstanceHandler(ctx),
				),
			),
		),
	})
	
	server.AddRoute(rest.Route{
		Method: http.MethodDelete,
		Path:   "/api/instances/:id",
		Handler: casdoorAuth.Handle(
			auditMw.Handle(
				permissionMw.RequireServiceGroupAccess("delete")(
					instance.DeregisterInstanceHandler(ctx),
				),
			),
		),
	})
	
	server.AddRoute(rest.Route{
		Method: http.MethodPost,
		Path:   "/api/instances/batch-delete",
		Handler: casdoorAuth.Handle(
			auditMw.Handle(
				permissionMw.RequireServiceGroupAccess("delete")(
					instance.BatchDeleteHandler(ctx),
				),
			),
		),
	})
	
	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/api/instances/datacenters",
		Handler: casdoorAuth.Handle(
			instance.GetDatacentersHandler(ctx),
		),
	})
	
	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/api/instances/services",
		Handler: casdoorAuth.Handle(
			instance.GetServicesHandler(ctx),
		),
	})
	
	// ============================================================
	// 权限管理（需要权限：admin）
	// ============================================================
	
	// 用户权限
	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/api/permissions/users",
		Handler: casdoorAuth.Handle(
			permissionMw.RequireAdmin()(
				permissionHandler.ListUserPermissionsHandler(ctx),
			),
		),
	})
	
	server.AddRoute(rest.Route{
		Method: http.MethodPost,
		Path:   "/api/permissions/users",
		Handler: casdoorAuth.Handle(
			auditMw.Handle(
				permissionMw.RequireAdmin()(
					permissionHandler.GrantUserPermissionHandler(ctx),
				),
			),
		),
	})
	
	server.AddRoute(rest.Route{
		Method: http.MethodDelete,
		Path:   "/api/permissions/users",
		Handler: casdoorAuth.Handle(
			auditMw.Handle(
				permissionMw.RequireAdmin()(
					permissionHandler.RevokeUserPermissionHandler(ctx),
				),
			),
		),
	})
	
	// 角色权限
	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/api/permissions/roles",
		Handler: casdoorAuth.Handle(
			permissionMw.RequireAdmin()(
				permissionHandler.ListRolePermissionsHandler(ctx),
			),
		),
	})
	
	server.AddRoute(rest.Route{
		Method: http.MethodPost,
		Path:   "/api/permissions/roles",
		Handler: casdoorAuth.Handle(
			auditMw.Handle(
				permissionMw.RequireAdmin()(
					permissionHandler.GrantRolePermissionHandler(ctx),
				),
			),
		),
	})
	
	server.AddRoute(rest.Route{
		Method: http.MethodDelete,
		Path:   "/api/permissions/roles",
		Handler: casdoorAuth.Handle(
			auditMw.Handle(
				permissionMw.RequireAdmin()(
					permissionHandler.RevokeRolePermissionHandler(ctx),
				),
			),
		),
	})
	
	// ============================================================
	// 审计日志（需要权限：audit_log）
	// ============================================================
	
	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/api/audit-logs",
		Handler: casdoorAuth.Handle(
			permissionMw.RequirePermission("audit_log", "read")(
				auditHandler.ListAuditLogsHandler(ctx),
			),
		),
	})
	
	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/api/audit-logs/export",
		Handler: casdoorAuth.Handle(
			permissionMw.RequirePermission("audit_log", "export")(
				auditHandler.ExportAuditLogsHandler(ctx),
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
		w.Write([]byte(`{"status":"ok","message":"Consul Manager is running","version":"1.0.0"}`))
	}
}

// serveEmbeddedWeb 提供嵌入式 Web 界面
func serveEmbeddedWeb() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		file := r.URL.Query().Get(":file")
		if file == "" {
			http.Error(w, "File not specified", http.StatusBadRequest)
			return
		}
		
		path := fmt.Sprintf("web/dist/assets/%s", file)
		content, err := webFS.ReadFile(path)
		if err != nil {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		
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
