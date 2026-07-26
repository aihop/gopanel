package helper

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type goPanelUserInfo struct {
	AdminUser        string   `json:"admin_user"`
	AdminPassword    string   `json:"admin_password"`
	CredentialSource string   `json:"credential_source"`
	Entrance         string   `json:"entrance"`
	Listen           string   `json:"listen"`
	LoginURL         string   `json:"login_url"`
	BaseDir          string   `json:"base_dir"`
	ConfigPath       string   `json:"config_path"`
	InitPath         string   `json:"init_path"`
	DatabasePath     string   `json:"database_path"`
	DatabaseExists   bool     `json:"database_exists"`
	AtUnixMs         int64    `json:"at_unix_ms"`
	ResetPasswordTip string   `json:"reset_password_tip"`
	Notes            []string `json:"notes"`
}

func (s *Server) actionGoPanelUserInfo(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = params
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}
	if strings.TrimSpace(s.cfg.GoPanelConfigPath) == "" {
		return "", errors.New("gopanel config path is empty")
	}

	cfg, err := readGoPanelConfigSimple(s.cfg.GoPanelConfigPath)
	if err != nil {
		return "", err
	}

	entrance := strings.TrimSpace(cfg.System.Entrance)
	// 面板实际监听的是 system.port，http.listen 只是历史遗留写法
	listen := strings.TrimSpace(cfg.System.Port)
	if listen == "" {
		listen = strings.TrimSpace(cfg.HTTP.Listen)
	}
	if listen != "" && !strings.Contains(listen, ":") {
		listen = ":" + listen
	}

	// 数据库位置由 base_dir 推导，和面板运行时一致（init/db 用的就是 <base_dir>/db/gopanel.db）。
	// 之前读 conf 里的 db.database / db.path —— 这两个键面板从来不写，所以这个字段永远是空的。
	baseDir := strings.TrimSpace(s.cfg.BaseDir)
	dbPath := filepath.Join(baseDir, "db", "gopanel.db")
	dbExists := false
	if st, err := os.Stat(dbPath); err == nil && !st.IsDir() {
		dbExists = true
	}

	initPath := filepath.Join(baseDir, "init.yaml")
	adminUser := ""
	adminPassword := ""
	credentialSource := ""
	if init, err := readInitInstallSimple(initPath); err == nil {
		adminUser = strings.TrimSpace(init.User)
		adminPassword = strings.TrimSpace(init.Password)
		if adminUser != "" || adminPassword != "" {
			credentialSource = initPath
		}
		if entrance == "" {
			entrance = strings.TrimSpace(init.SafeEnter)
		}
		if listen == "" && init.Port > 0 {
			listen = ":" + intToString(init.Port)
		}
	}

	notes := []string{}
	if credentialSource == "" {
		// 面板首次启动就会删掉 init.yaml，所以绝大多数机器上这两个字段必然是空的，
		// 明确说清楚，避免让人以为"这命令能查出密码"
		notes = append(notes, "admin_user/admin_password 只在面板首次启动前（init.yaml 尚未被删除）有值；面板启动后凭据无法找回，只能重置")
	}
	if !dbExists {
		notes = append(notes, "database_path 指向的文件不存在，请确认 base_dir 是否正确（当前: "+baseDir+"）")
	}
	if strings.HasPrefix(strings.TrimSpace(listen), ":") {
		notes = append(notes, "login_url 是本机回环地址；从外部访问请换成服务器 IP 或域名，并确保防火墙放行 "+strings.TrimPrefix(strings.TrimSpace(listen), ":"))
	}

	out := goPanelUserInfo{
		AdminUser:        adminUser,
		AdminPassword:    adminPassword,
		CredentialSource: credentialSource,
		Entrance:         entrance,
		Listen:           listen,
		LoginURL:         buildLoginURL(listen, entrance),
		BaseDir:          s.cfg.BaseDir,
		ConfigPath:       s.cfg.GoPanelConfigPath,
		InitPath:         initPath,
		DatabasePath:     dbPath,
		DatabaseExists:   dbExists,
		AtUnixMs:         time.Now().UnixMilli(),
		ResetPasswordTip: "Password cannot be recovered (bcrypt). Reset it with: gpc panel reset-password [email]",
		Notes:            notes,
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

type initInstallSimple struct {
	BaseDir   string
	Port      int
	User      string
	Password  string
	SafeEnter string
}

type goPanelConfSimple struct {
	System struct {
		Entrance string
		Port     string
	}
	HTTP struct {
		Listen string
	}
}

func readGoPanelConfigSimple(p string) (goPanelConfSimple, error) {
	var c goPanelConfSimple
	b, err := os.ReadFile(p)
	if err != nil {
		return c, err
	}
	lines := strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n")
	section := ""
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.Contains(line, ":") {
			continue
		}
		if strings.HasSuffix(line, ":") && !strings.Contains(line, " ") {
			section = strings.TrimSuffix(line, ":")
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		v = strings.Trim(v, `"'`)
		switch section {
		case "system":
			if k == "entrance" {
				c.System.Entrance = v
			}
			if k == "port" {
				c.System.Port = v
			}
		case "http":
			// 历史遗留：面板已不写这个键，仅作为老配置的兜底
			if k == "listen" {
				c.HTTP.Listen = v
			}
		}
	}
	return c, nil
}

func readInitInstallSimple(p string) (initInstallSimple, error) {
	var c initInstallSimple
	b, err := os.ReadFile(p)
	if err != nil {
		return c, err
	}
	lines := strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n")
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.Contains(line, ":") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		v = strings.Trim(v, `"'`)
		switch k {
		case "base_dir":
			c.BaseDir = v
		case "port":
			c.Port = parseInt(v)
		case "user":
			c.User = v
		case "password":
			c.Password = v
		case "safe_enter":
			c.SafeEnter = v
		}
	}
	return c, nil
}

func buildLoginURL(listen string, entrance string) string {
	listen = strings.TrimSpace(listen)
	entrance = strings.TrimSpace(entrance)

	host := ""
	port := ""
	if strings.HasPrefix(listen, ":") {
		host = "127.0.0.1"
		port = strings.TrimPrefix(listen, ":")
	} else {
		h, p, err := net.SplitHostPort(listen)
		if err == nil {
			host = h
			port = p
		}
	}
	if host == "" {
		host = "127.0.0.1"
	}
	if host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	if port == "" {
		return ""
	}
	base := "http://" + host + ":" + port
	if entrance == "" {
		return base
	}
	return base + "/" + strings.TrimPrefix(entrance, "/")
}

func parseInt(s string) int {
	n := 0
	sign := 1
	ss := strings.TrimSpace(s)
	if ss == "" {
		return 0
	}
	if strings.HasPrefix(ss, "-") {
		sign = -1
		ss = strings.TrimPrefix(ss, "-")
	}
	for _, r := range ss {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
	}
	return n * sign
}

func intToString(n int) string {
	if n == 0 {
		return "0"
	}
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	var b [32]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return sign + string(b[i:])
}
