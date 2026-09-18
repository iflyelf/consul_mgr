package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"

	"github.com/iflyelf/consul_mgr/internal/config"
	auditHandler "github.com/iflyelf/consul_mgr/internal/handler/audit"
	authHandler "github.com/iflyelf/consul_mgr/internal/handler/auth"
	"github.com/iflyelf/consul_mgr/internal/handler/group"
	"github.com/iflyelf/consul_mgr/internal/handler/instance"
	permissionHandler "github.com/iflyelf/consul_mgr/internal/handler/permission"
	roleHandler "github.com/iflyelf/consul_mgr/internal/handler/role"
	serviceHandler "github.com/iflyelf/consul_mgr/internal/handler/service"
	settingHandler "github.com/iflyelf/consul_mgr/internal/handler/setting"
	teamHandler "github.com/iflyelf/consul_mgr/internal/handler/team"
	userHandler "github.com/iflyelf/consul_mgr/internal/handler/user"
	userFieldHandler "github.com/iflyelf/consul_mgr/internal/handler/userfield"
	userfieldLogic "github.com/iflyelf/consul_mgr/internal/logic/userfield"
	"github.com/iflyelf/consul_mgr/internal/middleware"
	"github.com/iflyelf/consul_mgr/internal/svc"
)

var (
	configFile  = flag.String("c", "etc/config.yaml", "配置文件路径")
	showVersion = flag.Bool("version", false, "显示版本号并退出")
	version     = "1.0.0"
)

func main() {
	flag.Parse()

	// 显示版本号
	if *showVersion {
		fmt.Printf("consul_mgr version %s\n", version)
		return
	}

	// 加载配置：
	//  1. 配置文件存在：读取文件（字段上的 env 标签会自动应用环境变量覆盖）；
	//  2. 文件不存在：回退为代码内置默认值 + 环境变量，支持纯环境变量部署（如 Kubernetes）；
	//  3. 其它错误（权限等）：直接报错退出。
	var c config.Config
	if _, statErr := os.Stat(*configFile); statErr == nil {
		conf.MustLoad(*configFile, &c)
	} else if os.IsNotExist(statErr) {
		fmt.Fprintf(os.Stderr, "⚠️  配置文件 %s 不存在，改用内置默认值 + 环境变量\n", *configFile)
		if err := conf.FillDefault(&c); err != nil {
			log.Fatalf("加载内置默认配置失败: %v", err)
		}
	} else {
		log.Fatalf("读取配置文件 %s 失败: %v", *configFile, statErr)
	}

	// 环境变量覆盖监听地址/端口/模式/超时（支持容器编排自定义端口）
	c.ApplyEnvOverrides()

	// 必填项校验（失败即退出，避免以错误配置运行）
	if err := c.Validate(); err != nil {
		log.Fatalf("配置校验失败: %v", err)
	}

	// 创建服务上下文
	ctx := svc.NewServiceContext(&c)

	// 创建 REST 服务器
	opts := []rest.RunOption{
		rest.WithNotFoundHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 如果启用了嵌入式 Web，未找到的路由使用 SPA 处理器
			if c.Web.Embedded {
				getSPAHandler().ServeHTTP(w, r)
			} else {
				http.NotFound(w, r)
			}
		})),
	}
	// 跨域支持：仅当前端与后端不同源时配置 CORS_ALLOWED_ORIGINS。
	// go-zero 对具体 origin 会同时下发 Access-Control-Allow-Credentials: true，
	// 从而允许跨域携带登录 Cookie（需配合 AUTH_COOKIE_SAMESITE=none + HTTPS）。
	if len(c.Security.CORSAllowedOrigins) > 0 {
		logx.Infof("已启用跨域访问，允许来源: %v", c.Security.CORSAllowedOrigins)
		opts = append(opts, rest.WithCors(c.Security.CORSAllowedOrigins...))
	}
	server := rest.MustNewServer(c.RestConf, opts...)
	defer server.Stop()

	// 注册路由
	registerHandlers(server, ctx)

	// 启动用户字段自动同步调度器（启动时同步一次 + 定时检查，配置以页面/DB 为准）
	schedCtx, cancelScheduler := context.WithCancel(context.Background())
	defer cancelScheduler()
	// 首次启动写入配置种子（已存在则不覆盖）
	if err := userfieldLogic.NewLogic(ctx.DB).SeedSyncConfig(schedCtx,
		c.FlyIAM.SyncEnabled, c.FlyIAM.SyncOnStartup, c.FlyIAM.SyncInterval); err != nil {
		logx.Errorf("写入同步配置种子失败: %v", err)
	}
	userfieldLogic.StartScheduler(schedCtx, ctx.DB, &c)

	// 启动信息
	fmt.Printf("🚀 Starting Consul Manager Server\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("Version:       %s\n", version)
	fmt.Printf("Listen:        %s:%d\n", c.Host, c.Port)
	fmt.Printf("Auth Service:  FlyIAM (Casdoor)\n")
	fmt.Printf("  Endpoint:    %s\n", c.Casdoor.Endpoint)
	fmt.Printf("  Organization: %s\n", c.Casdoor.OrganizationName)
	fmt.Printf("  Application:  %s\n", c.Casdoor.ApplicationName)
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
	casdoorAuth := middleware.NewCasdoorAuthMiddleware(ctx.Casdoor)
	permissionMw := middleware.NewPermissionMiddleware(ctx.Casdoor, ctx.RawDB)
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
			Path:    "/api/auth/config",
			Handler: authHandler.ConfigHandler(ctx.Config),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/auth/login",
			Handler: authHandler.LoginHandler(ctx.Casdoor, ctx.Config),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/auth/callback",
			Handler: authHandler.CallbackHandler(ctx.Casdoor, ctx.Config),
		},
	})

	// ============================================================
	// 需要认证的路由
	// ============================================================

	// 认证相关
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/api/auth/userinfo",
		Handler: casdoorAuth.Handle(authHandler.GetUserInfoHandler(ctx.Casdoor)),
	})

	server.AddRoute(rest.Route{
		Method:  http.MethodPost,
		Path:    "/api/auth/refresh",
		Handler: casdoorAuth.Handle(authHandler.RefreshTokenHandler(ctx.Casdoor)),
	})

	server.AddRoute(rest.Route{
		Method:  http.MethodPost,
		Path:    "/api/auth/logout",
		Handler: casdoorAuth.Handle(authHandler.LogoutHandler(ctx.Config)),
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

	server.AddRoute(rest.Route{
		Method: http.MethodPost,
		Path:   "/api/groups/:id/test",
		Handler: casdoorAuth.Handle(
			group.TestConnectionHandler(ctx),
		),
	})

	server.AddRoute(rest.Route{
		Method: http.MethodPost,
		Path:   "/api/groups/detect-datacenter",
		Handler: casdoorAuth.Handle(
			group.DetectDatacenterHandler(ctx),
		),
	})

	// ============================================================
	// 服务管理（需要权限：consul_service）
	// ============================================================

	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/api/services",
		Handler: casdoorAuth.Handle(serviceHandler.ListServicesHandler(ctx)),
	})

	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/api/services/detail",
		Handler: casdoorAuth.Handle(serviceHandler.GetServiceDetailHandler(ctx)),
	})

	server.AddRoute(rest.Route{
		Method: http.MethodDelete,
		Path:   "/api/services",
		Handler: casdoorAuth.Handle(
			auditMw.Handle(
				permissionMw.RequireServiceGroupAccess("delete")(
					serviceHandler.DeleteServiceHandler(ctx),
				),
			),
		),
	})

	server.AddRoute(rest.Route{
		Method: http.MethodPost,
		Path:   "/api/services/batch-delete",
		Handler: casdoorAuth.Handle(
			auditMw.Handle(
				permissionMw.RequireServiceGroupAccess("delete")(
					serviceHandler.BatchDeleteServicesHandler(ctx),
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
			permissionMw.RequireServiceGroupAccess("read")(
				instance.GetInstanceHandler(ctx),
			),
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

	// 兼容基于查询参数的更新/删除（前端使用 instance_id 查询参数）
	server.AddRoute(rest.Route{
		Method: http.MethodPut,
		Path:   "/api/instances",
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
		Path:   "/api/instances",
		Handler: casdoorAuth.Handle(
			auditMw.Handle(
				permissionMw.RequireServiceGroupAccess("delete")(
					instance.DeregisterInstanceHandler(ctx),
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
		Method: http.MethodGet,
		Path:   "/api/instances/export",
		Handler: casdoorAuth.Handle(
			permissionMw.RequireServiceGroupAccess("read")(
				instance.ExportInstancesHandler(ctx),
			),
		),
	})

	server.AddRoute(rest.Route{
		Method: http.MethodPost,
		Path:   "/api/instances/import",
		Handler: casdoorAuth.Handle(
			auditMw.Handle(
				permissionMw.RequireServiceGroupAccess("write")(
					instance.ImportInstancesHandler(ctx),
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
		Method: http.MethodPost,
		Path:   "/api/instances/batch-register",
		Handler: casdoorAuth.Handle(
			auditMw.Handle(
				permissionMw.RequireServiceGroupAccess("write")(
					instance.BatchRegisterHandler(ctx),
				),
			),
		),
	})

	server.AddRoute(rest.Route{
		Method:  http.MethodPost,
		Path:    "/api/instances/batch-register/preview",
		Handler: casdoorAuth.Handle(instance.PreviewBatchRegisterHandler(ctx)),
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
	// 人员组织：用户 / 团队 / 角色（需要管理员权限）
	// ============================================================

	// 用户管理（来自 Casdoor，与 FlyIAM 对齐：列表 + 增删改 + 重置密码 + 管理员标记）
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/api/users",
		Handler: casdoorAuth.Handle(permissionMw.RequireAdmin()(userHandler.ListUsersHandler(ctx))),
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodPost,
		Path:    "/api/users",
		Handler: casdoorAuth.Handle(permissionMw.RequireAdmin()(userHandler.CreateUserHandler(ctx))),
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodPut,
		Path:    "/api/users/:name",
		Handler: casdoorAuth.Handle(permissionMw.RequireAdmin()(userHandler.UpdateUserHandler(ctx))),
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodDelete,
		Path:    "/api/users/:name",
		Handler: casdoorAuth.Handle(permissionMw.RequireAdmin()(userHandler.DeleteUserHandler(ctx))),
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodPost,
		Path:    "/api/users/:name/reset-password",
		Handler: casdoorAuth.Handle(permissionMw.RequireAdmin()(userHandler.ResetPasswordHandler(ctx))),
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodPost,
		Path:    "/api/users/:name/admin",
		Handler: casdoorAuth.Handle(permissionMw.RequireAdmin()(userHandler.SetUserAdminHandler(ctx))),
	})

	// 用户字段定义（页面可配置，与 FlyIAM 对齐）
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/api/user-fields",
		Handler: casdoorAuth.Handle(permissionMw.RequireAdmin()(userFieldHandler.ListUserFieldsHandler(ctx))),
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodPost,
		Path:    "/api/user-fields",
		Handler: casdoorAuth.Handle(permissionMw.RequireAdmin()(userFieldHandler.CreateUserFieldHandler(ctx))),
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodPut,
		Path:    "/api/user-fields/:id",
		Handler: casdoorAuth.Handle(permissionMw.RequireAdmin()(userFieldHandler.UpdateUserFieldHandler(ctx))),
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodDelete,
		Path:    "/api/user-fields/:id",
		Handler: casdoorAuth.Handle(permissionMw.RequireAdmin()(userFieldHandler.DeleteUserFieldHandler(ctx))),
	})
	// 从 FlyIAM 同步字段定义（数据源字段变化时无需改代码）
	server.AddRoute(rest.Route{
		Method:  http.MethodPost,
		Path:    "/api/user-fields/sync",
		Handler: casdoorAuth.Handle(permissionMw.RequireAdmin()(userFieldHandler.SyncUserFieldsHandler(ctx))),
	})
	// 自动同步配置 / 进度 / 日志
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/api/user-fields/sync/config",
		Handler: casdoorAuth.Handle(permissionMw.RequireAdmin()(userFieldHandler.GetSyncConfigHandler(ctx))),
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodPut,
		Path:    "/api/user-fields/sync/config",
		Handler: casdoorAuth.Handle(permissionMw.RequireAdmin()(userFieldHandler.UpdateSyncConfigHandler(ctx))),
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/api/user-fields/sync/progress",
		Handler: casdoorAuth.Handle(permissionMw.RequireAdmin()(userFieldHandler.GetSyncProgressHandler(ctx))),
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/api/user-fields/sync/logs",
		Handler: casdoorAuth.Handle(permissionMw.RequireAdmin()(userFieldHandler.ListSyncLogsHandler(ctx))),
	})

	// 应用设置（页面可配置，DB 优先 / env 兜底）
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/api/settings",
		Handler: casdoorAuth.Handle(permissionMw.RequireAdmin()(settingHandler.ListSettingsHandler(ctx))),
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodPut,
		Path:    "/api/settings",
		Handler: casdoorAuth.Handle(permissionMw.RequireAdmin()(settingHandler.UpdateSettingsHandler(ctx))),
	})

	// 角色管理
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/api/roles",
		Handler: casdoorAuth.Handle(permissionMw.RequireAdmin()(roleHandler.ListRolesHandler(ctx))),
	})
	server.AddRoute(rest.Route{
		Method: http.MethodPost,
		Path:   "/api/roles",
		Handler: casdoorAuth.Handle(
			auditMw.Handle(permissionMw.RequireAdmin()(roleHandler.CreateRoleHandler(ctx)))),
	})
	server.AddRoute(rest.Route{
		Method: http.MethodPut,
		Path:   "/api/roles/:id",
		Handler: casdoorAuth.Handle(
			auditMw.Handle(permissionMw.RequireAdmin()(roleHandler.UpdateRoleHandler(ctx)))),
	})
	server.AddRoute(rest.Route{
		Method: http.MethodDelete,
		Path:   "/api/roles/:id",
		Handler: casdoorAuth.Handle(
			auditMw.Handle(permissionMw.RequireAdmin()(roleHandler.DeleteRoleHandler(ctx)))),
	})

	// 团队管理
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/api/teams",
		Handler: casdoorAuth.Handle(permissionMw.RequireAdmin()(teamHandler.ListTeamsHandler(ctx))),
	})
	server.AddRoute(rest.Route{
		Method: http.MethodPost,
		Path:   "/api/teams",
		Handler: casdoorAuth.Handle(
			auditMw.Handle(permissionMw.RequireAdmin()(teamHandler.CreateTeamHandler(ctx)))),
	})
	server.AddRoute(rest.Route{
		Method: http.MethodPut,
		Path:   "/api/teams/:id",
		Handler: casdoorAuth.Handle(
			auditMw.Handle(permissionMw.RequireAdmin()(teamHandler.UpdateTeamHandler(ctx)))),
	})
	server.AddRoute(rest.Route{
		Method: http.MethodDelete,
		Path:   "/api/teams/:id",
		Handler: casdoorAuth.Handle(
			auditMw.Handle(permissionMw.RequireAdmin()(teamHandler.DeleteTeamHandler(ctx)))),
	})

	// 团队成员
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/api/teams/:id/members",
		Handler: casdoorAuth.Handle(permissionMw.RequireAdmin()(teamHandler.ListMembersHandler(ctx))),
	})
	server.AddRoute(rest.Route{
		Method: http.MethodPost,
		Path:   "/api/teams/:id/members",
		Handler: casdoorAuth.Handle(
			auditMw.Handle(permissionMw.RequireAdmin()(teamHandler.AddMemberHandler(ctx)))),
	})
	server.AddRoute(rest.Route{
		Method: http.MethodDelete,
		Path:   "/api/teams/:id/members",
		Handler: casdoorAuth.Handle(
			auditMw.Handle(permissionMw.RequireAdmin()(teamHandler.RemoveMemberHandler(ctx)))),
	})

	// 团队 → 服务组授权
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/api/teams/:id/permissions",
		Handler: casdoorAuth.Handle(permissionMw.RequireAdmin()(teamHandler.ListGroupPermissionsHandler(ctx))),
	})
	server.AddRoute(rest.Route{
		Method: http.MethodPost,
		Path:   "/api/teams/:id/permissions",
		Handler: casdoorAuth.Handle(
			auditMw.Handle(permissionMw.RequireAdmin()(teamHandler.GrantGroupPermissionHandler(ctx)))),
	})
	server.AddRoute(rest.Route{
		Method: http.MethodDelete,
		Path:   "/api/teams/:id/permissions",
		Handler: casdoorAuth.Handle(
			auditMw.Handle(permissionMw.RequireAdmin()(teamHandler.RevokeGroupPermissionHandler(ctx)))),
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
