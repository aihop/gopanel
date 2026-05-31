package constant

const (
	WebRunning = "Running"
	WebStopped = "Stopped"

	ProtocolHTTP  = "HTTP"
	ProtocolHTTPS = "HTTPS"

	NewApp       = "new"
	InstalledApp = "installed"

	WebApp    = "web_app"
	Container = "container"
	Static    = "static"
	Proxy     = "proxy"
	Redirect  = "redirect"

	SSLExisted = "existed"
	SSLAuto    = "auto"
	SSLManual  = "manual"

	DNSAccount = "dnsAccount"
	DnsManual  = "dnsManual"
	Http       = "http"
	Manual     = "manual"
	SelfSigned = "selfSigned"

	StartWeb = "start"
	StopWeb  = "stop"

	HTTPSOnly   = "HTTPSOnly"
	HTTPAlso    = "HTTPAlso"
	HTTPToHTTPS = "HTTPToHTTPS"

	GetLog     = "get"
	DisableLog = "disable"
	EnableLog  = "enable"
	DeleteLog  = "delete"

	AccessLog = "access.log"
	ErrorLog  = "error.log"

	ConfigPHP = "php"
	ConfigFPM = "fpm"

	SSLInit       = "init"
	SSLError      = "error"
	SSLReady      = "ready"
	SSLApply      = "applying"
	SSLApplyError = "applyError"
)
