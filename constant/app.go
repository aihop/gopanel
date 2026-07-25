package constant

// 版本管理设置为变量，通过go build -ldflags 注入
var (
	AppVersion       = "1.0.0"
	AppName          = "gopanel"
	AppBrand         = "GoPanel"
	BuildTime        = "2026-04-14T08:42:20+0000"
	BuildVersionCode = "100000"
)

const (
	AppSite      = "https://gopanel.cn/"
	AppXAuth     = "x-auth"
	AppAuth      = "auth"
	AppToken     = "token"
	AppAPIKey    = "apiKey"
	AppTimestamp = "timestamp"

	AppAuthName    = "claims"
	HashDefaultLen = 6 // HASH默认长度
	StrLen32       = 32

	StatusFail    = 10   // 未发布
	StatusSucc    = 20   // 已发布
	StatusSuccStr = "20" // 已发布

	StatusCodeSuccess = 0  // 成功
	StatusCodeDialog  = 10 // 对话逻辑

	StatusCodeAccessUnauth   = 30 // 未授权访问
	StatusCodeAccessPassword = 31 // 需要密码访问
	StatusCodeAccessRedirect = 32 // 跳转访问

	StatusCodeFail        = 40  // 失败-不拦截
	StatusCodeFullFail    = 41  // 失败-全局拦截
	StatusCodeAuthInvalid = 50  // 未登录
	StatusCodeAuthOut     = 51  // 设备已在其他设备登录
	StatusCodeError       = 500 // 内部服务器错误

	OsWindows = "windows"
	OsLinux   = "linux"
)

const (
	Running     = "Running"
	UnHealthy   = "UnHealthy"
	Error       = "Error"
	Stopped     = "Stopped"
	Installing  = "Installing"  // 安装中
	DownloadErr = "DownloadErr" // 下载错误
	Upgrading   = "Upgrading"
	UpgradeErr  = "UpgradeErr"
	Rebuilding  = "Rebuilding"
	Syncing     = "Syncing"
	SyncSuccess = "SyncSuccess"
	Paused      = "Paused"
	UpErr       = "UpErr"      // 更新错误
	SyncFailed  = "SyncFailed" // 同步失败

	AppNormal   = "Normal"
	AppTakeDown = "TakeDown"

	AppOpenresty  = "openresty"
	AppMysql      = "mysql"
	AppMariaDB    = "mariadb"
	AppPostgresql = "postgresql"
	AppRedis      = "redis"
	AppPostgres   = "postgres"
	AppMongodb    = "mongodb"
	AppMemcached  = "memcached"

	AppResourceLocal  = "local"
	AppResourceRemote = "remote"

	CPUS          = "CPUS"
	MemoryLimit   = "MEMORY_LIMIT"
	HostIP        = "HOST_IP"
	ContainerName = "CONTAINER_NAME"
	Entrance      = "Entrance"

	// NodeAccessTokenKey 被控侧只读令牌在 setting 表中的存储键
	NodeAccessTokenKey = "NodeAccessToken"

	// NodeControlTokenKey 被控侧控制令牌。与只读令牌严格分开：
	// 只读令牌只能取摘要，控制令牌等价于该机管理员，能通过代理执行写操作。
	NodeControlTokenKey = "NodeControlToken"

	OperateUp = "up"
)

type AppOperate string

var (
	Start   AppOperate = "start"
	Stop    AppOperate = "stop"
	Restart AppOperate = "restart"
	Delete  AppOperate = "delete"
	Sync    AppOperate = "sync"
	Backup  AppOperate = "backup"
	Update  AppOperate = "update"
	Rebuild AppOperate = "rebuild"
	Upgrade AppOperate = "upgrade"
	Reload  AppOperate = "reload"
)
