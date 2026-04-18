package app

import (
	"encoding/gob"
	"fmt"
	sysLog "log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gofiber/fiber/v3"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"golang.org/x/text/language"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/app/router"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/init/app"
	"github.com/aihop/gopanel/init/cache"
	"github.com/aihop/gopanel/init/caddy"
	"github.com/aihop/gopanel/init/conf"
	"github.com/aihop/gopanel/init/cron"
	"github.com/aihop/gopanel/init/daemon"
	"github.com/aihop/gopanel/init/db"
	"github.com/aihop/gopanel/init/docker"
	"github.com/aihop/gopanel/init/log"
	"github.com/aihop/gopanel/init/repo"
	"github.com/aihop/gopanel/init/session"
	"github.com/aihop/gopanel/init/session/psession"
	"github.com/aihop/gopanel/pkg/i18n"

	"github.com/gofiber/fiber/v3/middleware/pprof"
	appRecover "github.com/gofiber/fiber/v3/middleware/recover"

	"github.com/aihop/gopanel/cmd"
)

type App struct {
	App    *fiber.App
	IsInit bool
}

var (
	List    = make(map[string]*Config)
	Indexes []string
)

type Config struct {
	Name   string
	Config any
	Init   func(t *App)
	Route  func(t *App)
}

func (t *App) Init() {
	cmd.Init()
	conf.Init()
	db.Init()
	repo.Init()
	app.Init()
	log.Init()
	caddy.Init()
	cache.Init()
	daemon.Init()
	docker.Init()
	cron.Init()
	gob.Register(psession.SessionUser{})
	session.Init()
	t.IsInit = true
}

func (r *App) Route() *fiber.App {
	r.Init()
	r.App = r.newFiber()
	return r.App
}

func (r *App) newFiber() *fiber.App {
	app := fiber.New() // fiber.Config{ BodyLimit: 10 * 1024 * 1024,} 如果需要设置请求体大小限制,包括文件上传
	// app.Use(cors.New())
	// 国际化 中国
	app.Use(i18n.New(&i18n.Config{
		RootPath:        "resource/locale",
		AcceptLanguages: []language.Tag{language.Chinese, language.English},
		DefaultLanguage: language.Chinese,
		Loader:          &i18n.EmbedLoader{FS: global.EmbedFS},
	}))
	// pprof 性能分析
	var pprofPrefix = ""

	app.Use(pprof.New(pprof.Config{Prefix: pprofPrefix}))
	// 捕捉堆栈错误
	app.Use(appRecover.New())
	app.Use(middleware.CatchPanicError)

	app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(dto.Result{Code: 0, Msg: "success", Data: map[string]interface{}{"appBrand": constant.AppBrand, "appVersion": constant.AppVersion, "appSite": constant.AppSite, "appName": constant.AppName}})
	})

	return app
}

func (r *App) reloadFiber(isNew bool) *fiber.App {
	if isNew {
		r.App = r.newFiber()
	} else {
		if r.App == nil {
			r.App = r.newFiber()
		}
	}
	router.AppRegister(r.App.Group("/"))
	return r.App
}

func (r *App) Run() error {
	// 默认初始化
	if !r.IsInit {
		r.Init()
	}

	r.reloadFiber(false)

	// HTTP服务
	go func() {
		if err := r.App.Listen(global.CONF.System.Port, fiber.ListenConfig{DisableStartupMessage: true}); err != nil {
			sysLog.Fatalf("run listen error: %v", err)
		}
	}()
	r.startupMessage()
	// 优雅地处理退出信号
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	// 阻塞主线程，直到收到退出信号
	<-c
	return nil
}

func (app *App) startupMessage() {

	out := colorable.NewColorableStdout()
	if os.Getenv("TERM") == "dumb" || os.Getenv("NO_COLOR") == "1" || (!isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd())) {
		out = colorable.NewNonColorable(os.Stdout)
	}
	fmt.Fprint(out, strings.Repeat("-", 50)+"\n")
	fmt.Fprintf(out,
		"%sSystem%s Info: \t%s%s%s\n",
		"\u001b[92m", "\u001b[0m", "\u001b[94m", fmt.Sprintf("%s %s", constant.AppBrand, constant.AppVersion), "\u001b[0m")

	fmt.Fprintf(out,
		"Listen %sHTTP%s Server started on: \t%s%s%s\n",
		"\u001b[92m", "\u001b[0m", "\u001b[94m", fmt.Sprintf("%s%s", "127.0.0.1", global.CONF.System.Port), "\u001b[0m")

	// add new Line as spacer
	fmt.Fprintf(out, "\n%s", "\u001b[0m")
}

func (r *App) Reload() {
	r.App.Server().Handler = r.reloadFiber(true).Handler()
}

// Register Call this method to register the application with the framework
func Register(opt *Config) {
	List[opt.Name] = opt
	Indexes = append(Indexes, opt.Name)
}
