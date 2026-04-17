package dto

// 最外层
type CaddyConfig struct {
	Apps Apps `json:"apps"`
}

// ── apps --------------------------------------------------
type Apps struct {
	HTTP HTTP `json:"http"`
}

// ── http --------------------------------------------------
type HTTP struct {
	Servers map[string]Server `json:"servers"`
}

// ── server ------------------------------------------------
type Server struct {
	Listen []string `json:"listen"`
	Routes []Route  `json:"routes"`
}

// ── route -------------------------------------------------
type Route struct {
	Match    []Match  `json:"match,omitempty"`
	Handle   []Handle `json:"handle"`
	Terminal bool     `json:"terminal,omitempty"`
}

// ── match -------------------------------------------------
type Match struct {
	Host []string `json:"host,omitempty"`
}

// ── handle ------------------------------------------------
type Handle struct {
	Handler   string     `json:"handler"`
	Body      string     `json:"body,omitempty"`      // static_response 时
	Routes    []SubRoute `json:"routes,omitempty"`    // subroute 时
	Upstreams []Upstream `json:"upstreams,omitempty"` // reverse_proxy 时
}

// ── sub-route ---------------------------------------------
type SubRoute struct {
	Handle []Handle `json:"handle"`
}

// ── upstream ----------------------------------------------
type Upstream struct {
	Dial string `json:"dial"`
}
