package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"

	"github.com/iflyelf/consul_mgr/internal/config"
	"github.com/iflyelf/consul_mgr/internal/handler/auth"
	"github.com/iflyelf/consul_mgr/internal/handler/group"
	"github.com/iflyelf/consul_mgr/internal/handler/instance"
	"github.com/iflyelf/consul_mgr/internal/handler/service"
	"github.com/iflyelf/consul_mgr/internal/handler/user"
	"github.com/iflyelf/consul_mgr/internal/middleware"
	"github.com/iflyelf/consul_mgr/internal/pkg/response"
	"github.com/iflyelf/consul_mgr/internal/svc"
)

var (
	configFile = flag.String("c", "etc/config.yaml", "配置文件路径")
	version    = "dev"
)

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	ctx := svc.NewServiceContext(c)
	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	// 注册路由
	registerHandlers(server, ctx)

	fmt.Printf("Starting consul_mgr server at %s:%d...\n", c.Host, c.Port)
	fmt.Printf("Version: %s\n", version)
	fmt.Printf("Admin: %s\n", c.Admin.Username)
	fmt.Println("======================================")
	fmt.Println("API Documentation: http://localhost:8080/")
	fmt.Println("Health Check: http://localhost:8080/health")
	fmt.Println("======================================")
	
	server.Start()
}

func registerHandlers(server *rest.Server, ctx *svc.ServiceContext) {
	// 创建认证中间件（暂时未使用）
	_ = middleware.NewAuthMiddleware(ctx.JWTManager)
	
	// 健康检查（无需认证）
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/health",
		Handler: healthHandler(),
	})
	
	// 认证路由（无需认证）
	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodPost,
				Path:    "/api/auth/login",
				Handler: auth.LoginHandler(ctx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/api/auth/logout",
				Handler: auth.LogoutHandler(ctx),
			},
		},
	)
	
	// 需要认证的路由
	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodGet,
				Path:    "/api/auth/info",
				Handler: auth.GetUserInfoHandler(ctx),
			},
			// 用户管理
			{
				Method:  http.MethodGet,
				Path:    "/api/users",
				Handler: user.ListUsersHandler(ctx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/api/users/:id",
				Handler: user.GetUserHandler(ctx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/api/users",
				Handler: user.CreateUserHandler(ctx),
			},
			{
				Method:  http.MethodPut,
				Path:    "/api/users/:id",
				Handler: user.UpdateUserHandler(ctx),
			},
			{
				Method:  http.MethodDelete,
				Path:    "/api/users/:id",
				Handler: user.DeleteUserHandler(ctx),
			},
			{
				Method:  http.MethodPut,
				Path:    "/api/users/:id/password",
				Handler: user.ChangePasswordHandler(ctx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/api/users/:id/roles",
				Handler: user.AssignRolesHandler(ctx),
			},
			// 服务组管理
			{
				Method:  http.MethodGet,
				Path:    "/api/groups",
				Handler: group.ListGroupsHandler(ctx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/api/groups",
				Handler: group.CreateGroupHandler(ctx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/api/groups/:id",
				Handler: group.GetGroupHandler(ctx),
			},
			{
				Method:  http.MethodPut,
				Path:    "/api/groups/:id",
				Handler: group.UpdateGroupHandler(ctx),
			},
			{
				Method:  http.MethodDelete,
				Path:    "/api/groups/:id",
				Handler: group.DeleteGroupHandler(ctx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/api/groups/:id/test",
				Handler: group.TestConnectionHandler(ctx),
			},
			// Consul 服务管理
			{
				Method:  http.MethodGet,
				Path:    "/api/services",
				Handler: service.ListServicesHandler(ctx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/api/services/detail",
				Handler: service.GetServiceDetailHandler(ctx),
			},
			{
				Method:  http.MethodDelete,
				Path:    "/api/services",
				Handler: service.DeleteServiceHandler(ctx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/api/services/batch-delete",
				Handler: service.BatchDeleteServicesHandler(ctx),
			},
			// Consul 实例管理
			{
				Method:  http.MethodGet,
				Path:    "/api/instances",
				Handler: instance.ListInstancesHandler(ctx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/api/instances",
				Handler: instance.RegisterInstanceHandler(ctx),
			},
			{
				Method:  http.MethodPut,
				Path:    "/api/instances",
				Handler: instance.UpdateInstanceHandler(ctx),
			},
			{
				Method:  http.MethodDelete,
				Path:    "/api/instances",
				Handler: instance.DeleteInstanceHandler(ctx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/api/instances/batch-delete",
				Handler: instance.BatchDeleteInstancesHandler(ctx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/api/instances/export",
				Handler: instance.ExportInstancesHandler(ctx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/api/instances/import",
				Handler: instance.ImportInstancesHandler(ctx),
			},
		},
		rest.WithJwt(ctx.Config.JWT.Secret),
	)
	
	// 静态文件服务（嵌入的前端）- 必须在最后注册
	// 根路径
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/",
		Handler: getSPAHandler().ServeHTTP,
	})
	
	// assets 静态资源
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/assets/:file",
		Handler: getSPAHandler().ServeHTTP,
	})
	
	// 其他所有路径（SPA 路由）
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/:path",
		Handler: getSPAHandler().ServeHTTP,
	})
	
	log.Println("路由注册完成")
}

// healthHandler 健康检查
func healthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response.Success(w, map[string]interface{}{
			"status":  "ok",
			"version": version,
			"message": "Consul Manager is running",
		})
	}
}

// indexHandler 首页
func indexHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		html := `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Consul Manager</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
            max-width: 800px;
            margin: 50px auto;
            padding: 20px;
            background: #f5f5f5;
        }
        .container {
            background: white;
            padding: 40px;
            border-radius: 10px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        h1 { color: #333; margin-bottom: 10px; }
        .subtitle { color: #666; margin-bottom: 30px; }
        .status { color: #22c55e; font-weight: bold; }
        .info { background: #f9fafb; padding: 20px; border-radius: 5px; margin: 20px 0; }
        .info-item { margin: 10px 0; }
        .label { color: #666; display: inline-block; width: 120px; }
        .value { color: #333; font-weight: 500; }
        .links { margin-top: 30px; }
        .links a { color: #3b82f6; text-decoration: none; margin-right: 20px; }
        .links a:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🛡️ Consul Manager</h1>
        <p class="subtitle">Consul 服务管理平台</p>
        
        <div class="status">● 服务运行中</div>
        
        <div class="info">
            <div class="info-item">
                <span class="label">版本:</span>
                <span class="value">` + version + `</span>
            </div>
            <div class="info-item">
                <span class="label">API 地址:</span>
                <span class="value">/api</span>
            </div>
            <div class="info-item">
                <span class="label">健康检查:</span>
                <span class="value">/health</span>
            </div>
        </div>
        
        <div class="links">
            <a href="/health">健康检查</a>
            <a href="https://github.com/iflyelf/consul_mgr">GitHub</a>
            <a href="https://github.com/iflyelf/consul_mgr/blob/main/README.md">文档</a>
        </div>
    </div>
</body>
</html>
`
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
	}
}
