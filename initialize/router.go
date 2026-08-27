package initialize

import (
	aiRouter "apipig/app/ai/router"
	reviewRouter "apipig/app/apps/code-review/router"
	reviewService "apipig/app/apps/code-review/service"
	remoteAgentRouter "apipig/app/apps/remote-agent/router"
	remoteAgentService "apipig/app/apps/remote-agent/service"
	wechatBotRouter "apipig/app/apps/wechat-bot/router"
	wechatBotService "apipig/app/apps/wechat-bot/service"
	sysRouter "apipig/app/sys/router"
	_ "apipig/docs"
	"apipig/global"
	"apipig/middleware"
	"apipig/web"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"

	swagger "github.com/arsmn/fiber-swagger/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"go.uber.org/zap"
)

func Routers() *fiber.App {
	var app = fiber.New(fiber.Config{
		DisableStartupMessage: true,
		BodyLimit:             8 << 20,

		// https://github.com/goccy/go-json
		JSONEncoder: json.Marshal,
		JSONDecoder: json.Unmarshal,
	})
	wechatBotService.WechatBotService.BotService.SetInboundHandler(remoteAgentService.RemoteAgentService.ConversationService.HandleWechatInbound)
	wechatBotService.WechatBotService.BotService.SetOutboundHandler(remoteAgentService.RemoteAgentService.ConversationService.HandleWechatOutbound)
	remoteAgentService.RemoteAgentService.ConversationService.SetTakeoverSender(wechatBotService.WechatBotService.BotService.SendTakeover)
	wechatBotService.StartWechatBots()
	if err := remoteAgentService.RemoteAgentService.AgentService.StartLivenessTracking(); err != nil {
		global.LOG.Panic("start remote agent liveness tracking failed", zap.Error(err))
	}
	if global.CONFIG.System.EnableSwagger {
		global.LOG.Debug("register swagger handler")
		app.Get("/swagger/*", swagger.HandlerDefault)
	}
	aiRouter.AiRouter.InitHealthRouter(app)

	global.LOG.Debug("register upload file handler")

	// 路由日志
	if global.CONFIG.System.PrintRoute {
		global.LOG.Debug("use middleware logger")
		app.Use(middleware.Logger())
	}

	global.LOG.Debug("use middleware recover")
	app.Use(middleware.Recover())

	// 跨域
	global.LOG.Debug("use middleware cors")
	app.Use(middleware.Cors())

	// 获取context-path
	prefix := global.CONFIG.System.ContextPath
	if prefix == "" {
		fmt.Printf("context-path为默认值,路径为/ \n")
		prefix = "/"
	} else {
		fmt.Printf("context-path为%v \n", prefix)
	}

	prefix = prefix + "v1"

	// 注入免鉴权路由
	publicGroup := app.Group(prefix)
	{
		sysRouter.SysRouter.InitWebRouter(publicGroup)
		aiRouter.AiRouter.InitProtocolRouter(publicGroup)
		reviewRouter.ReviewRouter.InitWebhookRouter(publicGroup)
		wechatBotRouter.WechatBotRoutes.InitWebhookRouter(publicGroup)
	}

	// Agent-facing endpoints use a dedicated bootstrap/per-agent token flow.
	remoteAgentGroup := app.Group(prefix + "/remote-agent/")
	{
		remoteAgentRouter.RemoteAgentRoutes.InitAgentRouter(remoteAgentGroup)
	}

	// 注入系统鉴权路由
	sysGroup := app.Group(prefix + "/sys/")
	sysGroup.Use(middleware.JWTAuth()).Use(middleware.RbacHandler())
	{
		sysRouter.SysRouter.InitUserRouter(sysGroup)
		sysRouter.SysRouter.InitRoleRouter(sysGroup)
		sysRouter.SysRouter.InitResourceRouter(sysGroup)
		sysRouter.SysRouter.InitResourceApiRouter(sysGroup)

	}

	// AI 网关管理面先校验 JWT，再按请求路径映射已有菜单资源并校验角色权限。
	aiGroup := app.Group(prefix + "/ai/")
	aiGroup.Use(middleware.JWTAuth()).Use(middleware.RequireAIResourceAccess())
	{
		aiRouter.AiRouter.InitGatewayAdminRouter(aiGroup)
		aiRouter.AiRouter.InitProviderRouter(aiGroup)
		aiRouter.AiRouter.InitChannelRouter(aiGroup)
		aiRouter.AiRouter.InitChannelAccountRouter(aiGroup)
		aiRouter.AiRouter.InitAccessTokenRouter(aiGroup)
		aiRouter.AiRouter.InitAccessTokenTagRouter(aiGroup)
		aiRouter.AiRouter.InitProxyRouter(aiGroup)
		aiRouter.AiRouter.InitCallLogRouter(aiGroup)
	}

	// AI 网关流量入口使用独立的调用方 Token 鉴权，不能强制走后台 JWT。
	aiGatewayGroup := app.Group(prefix + "/ai-gateway/")
	{
		aiRouter.AiRouter.InitGatewayProxyRouter(aiGatewayGroup)
	}

	reviewGroup := app.Group(prefix + "/ai-applications/code-review/")
	reviewGroup.Use(middleware.JWTAuth()).Use(middleware.RbacHandler())
	{
		reviewRouter.ReviewRouter.InitAdminRouter(reviewGroup)
	}

	remoteAgentAdminGroup := app.Group(prefix + "/ai-applications/remote-agent/")
	remoteAgentAdminGroup.Use(middleware.JWTAuth()).Use(middleware.RbacHandler())
	{
		remoteAgentRouter.RemoteAgentRoutes.InitAdminRouter(remoteAgentAdminGroup)
	}
	wechatBotAdminGroup := app.Group(prefix + "/ai-applications/wechat-bot/")
	wechatBotAdminGroup.Use(middleware.JWTAuth()).Use(middleware.RbacHandler())
	{
		wechatBotRouter.WechatBotRoutes.InitAdminRouter(wechatBotAdminGroup)
	}
	reviewService.StartReview()

	global.LOG.Debug("router register success")

	global.LOG.Debug("register filesystem handler")

	// 将嵌入的文件系统转换为文件系统子树
	staticFiles, err := fs.Sub(web.Dist, "dist")
	if err != nil {
		global.LOG.Error("staticFiles error.", zap.Any("err", err))
	}

	// 设置静态文件服务
	app.Use("/", filesystem.New(filesystem.Config{
		Root:         http.FS(staticFiles),
		Browse:       true,
		Index:        "index.html",
		NotFoundFile: "index.html",
		MaxAge:       3600,
	}))

	// 404 处理，重定向到首页
	app.Use(func(c *fiber.Ctx) error {
		if c.Path() != "/" {
			// 重定向到首页
			return c.Redirect("/", http.StatusFound)
		}
		return c.Next()
	})
	return app
}
