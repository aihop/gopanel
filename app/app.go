package app

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"golang.org/x/text/language"

	"github.com/aihop/gopanel/app/api"
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/app/router"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/init/app"
	"github.com/aihop/gopanel/init/cache"
	"github.com/aihop/gopanel/init/conf"
	"github.com/aihop/gopanel/init/cron"
	"github.com/aihop/gopanel/init/db"
	"github.com/aihop/gopanel/init/docker"
	"github.com/aihop/gopanel/init/geo"
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

func (t *App) Init() error {
	cmd.Init()
	if err := conf.Init(); err != nil {
		return fmt.Errorf("initialize config: %w", err)
	}
	db.Init()
	if err := repo.Init(); err != nil {
		return fmt.Errorf("initialize repositories: %w", err)
	}
	tokenRepair, err := api.RepairLegacyCodeTokenUsage(global.DB)
	if err != nil {
		return fmt.Errorf("repair legacy Code token usage: %w", err)
	}
	if tokenRepair.Recorded+tokenRepair.Recovered+tokenRepair.Unavailable > 0 {
		fmt.Printf("Code token usage repair: recorded=%d recovered=%d unavailable=%d\n", tokenRepair.Recorded, tokenRepair.Recovered, tokenRepair.Unavailable)
	}
	// 从数据库读回 API Token 等运行期设置到内存 CONF（DB 为准，修复重启后 token 丢失）
	service.LoadApiSettingsFromDB()
	app.Init()
	log.Init()
	geo.Init()
	cache.Init()
	docker.Init()
	cron.Init()
	gob.Register(psession.SessionUser{})
	session.Init()

	// 注意：gp-agent 不做自动更新 —— 既不在后端启动时触发，也不在进入面板时触发。
	// 只有用户在「主机 - 工具箱 - 守护进程」页面点「更新 gp-agent」才会更新，
	// 走 POST /agent/update（见 api.AgentUpdate → service.UpdateGpAgent）。

	t.IsInit = true
	return nil
}

func (r *App) Route() (*fiber.App, error) {
	if err := r.Init(); err != nil {
		return nil, err
	}
	r.App = r.newFiber()
	return r.App, nil
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
	// 捕捉堆栈错误
	app.Use(appRecover.New())
	app.Use(middleware.CatchPanicError)
	app.Use(middleware.Entrance)
	app.Use("/debug/pprof", middleware.JWT(constant.UserRoleSuper))
	app.Use(pprof.New())

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
		if err := r.Init(); err != nil {
			return err
		}
	}

	listener, err := net.Listen("tcp", global.CONF.System.Port)
	if err != nil {
		return fmt.Errorf("listen %s: %w", global.CONF.System.Port, err)
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- r.Serve(listener)
	}()

	// 优雅地处理退出信号
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(c)

	select {
	case <-c:
	case err := <-serveErr:
		return err
	}

	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelShutdown()
	return r.Shutdown(shutdownContext)
}

func (r *App) Serve(listener net.Listener) error {
	if !r.IsInit {
		if err := r.Init(); err != nil {
			return err
		}
	}
	r.reloadFiber(false)
	api.StartCodeSessionInitialization()
	api.StartCodeInstructionRecovery()
	api.StartCodeDeliveryRecovery()
	r.startupMessage(listener.Addr().String())

	err := r.App.Listener(listener, fiber.ListenConfig{DisableStartupMessage: true})
	if errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}

func (r *App) Shutdown(ctx context.Context) error {
	codeErr := api.ShutdownCodeExecutions(ctx)
	httpErr := r.App.ShutdownWithContext(ctx)
	return errors.Join(codeErr, httpErr)
}

func (app *App) startupMessage(address string) {

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
		"\u001b[92m", "\u001b[0m", "\u001b[94m", address, "\u001b[0m")

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
