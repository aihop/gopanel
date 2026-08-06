package constant

const (
	AuthMethodSession = "session"
	SessionName       = "psession"
	AuthMethodName    = "authMethod"
	AuthMethodJWT     = "jwt"
	AuthMethodAPIKey  = "api-key"
	// AuthMethodNodeProxy 请求来自主控面板的代理转发，身份由节点控制令牌签名证明。
	// 单独标记出来是为了让操作日志能区分"人在本机点的"和"主控代理过来的"。
	AuthMethodNodeProxy = "node-proxy"
	JWTHeaderName       = "PanelAuthorization"
	JWTBufferTime       = 3600
	JWTIssuer           = "GoPanel"

	PasswordExpiredName = "expired"
)
