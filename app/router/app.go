package router

import (
	"io/fs"

	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/global"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
)

func AppRegister(r fiber.Router) {

	apiRouter := r.Group("api")
	apiRouter.Use(middleware.OperationLog()) // 增加操作日志拦截

	AuthRouter(apiRouter)      // 认证相关路由
	DashboardRouter(apiRouter) // 首页
	HostRouter(apiRouter)      // 主机信息
	UserRouter(apiRouter)      // 用户
	ContainerRouter(apiRouter) // 容器
	FileRouter(apiRouter)      // 文件
	HttpRouter(apiRouter)      // HTTP 服务
	ProcessRouter(apiRouter)   // 进程管理
	AppsRouter(apiRouter)      // app 安装
	SettingRouter(apiRouter)   // 系统设置
	DatabaseRouter(apiRouter)  // 数据库
	BackupRouter(apiRouter)    // 备份
	CloudRouter(apiRouter)     // 云服务
	WebsiteRouter(apiRouter)   // 网站
	SSLRouter(apiRouter)       // SSL 证书
	PipelineRouter(apiRouter)  // 流水线
	AIAgentRouter(apiRouter)   // AI助手终端
	LogsRouter(apiRouter)      // 日志

	staticRouter(r) // 静态资源文件
}

func staticRouter(r fiber.Router) {
	// 在 Go 1.16+ 中，Fiber v3 推荐直接将 fs.FS 传给 static 中间件，并使用 fs.Sub 定位到子目录
	assetsFS, _ := fs.Sub(global.EmbedFS, "public/assets")
	imagesFS, _ := fs.Sub(global.EmbedFS, "public/images")

	// 托管静态目录
	r.Get("/assets/*", static.New("", static.Config{
		FS: assetsFS,
	}))
	r.Get("/images/*", static.New("", static.Config{
		FS: imagesFS,
	}))
	r.Get("/favicon.svg", func(c fiber.Ctx) error {
		file, err := global.EmbedFS.ReadFile("public/favicon.svg")
		if err != nil {
			return c.SendStatus(fiber.StatusNotFound)
		}
		c.Set("Content-Type", "image/svg+xml")
		return c.Send(file)
	})
	r.Get("/favicon.ico", func(c fiber.Ctx) error {
		file, err := global.EmbedFS.ReadFile("public/favicon.ico")
		if err != nil {
			return c.SendStatus(fiber.StatusNotFound)
		}
		c.Set("Content-Type", "image/x-icon")
		return c.Send(file)
	})
	r.Get("/*", func(c fiber.Ctx) error {
		file, err := global.EmbedFS.ReadFile("public/index.html")
		if err != nil {
			return c.SendStatus(fiber.StatusNotFound)
		}
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.Send(file)
	})
}
