package helper

import (
	"context"
	"errors"
	"encoding/json"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type goPanelUserInfo struct {
	AdminEmail       string `json:"admin_email"`
	Entrance         string `json:"entrance"`
	Listen           string `json:"listen"`
	LoginURL         string `json:"login_url"`
	BaseDir          string `json:"base_dir"`
	ConfigPath       string `json:"config_path"`
	DatabasePath     string `json:"database_path"`
	AtUnixMs         int64  `json:"at_unix_ms"`
	ResetPasswordTip string `json:"reset_password_tip"`
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
	listen := strings.TrimSpace(cfg.HTTP.Listen)
	dbPath := strings.TrimSpace(cfg.DB.Database)
	if dbPath == "" {
		dbPath = strings.TrimSpace(cfg.DB.Path)
	}
	if dbPath == "" {
		return "", errors.New("db.database is empty in config")
	}
	if !filepath.IsAbs(dbPath) {
		dbPath = filepath.Join(strings.TrimSpace(s.cfg.BaseDir), dbPath)
	}

	adminEmail := ""
	if e, err := queryAdminEmailBySqlite3(ctx, dbPath); err == nil {
		adminEmail = e
	}

	out := goPanelUserInfo{
		AdminEmail:       adminEmail,
		Entrance:         entrance,
		Listen:           listen,
		LoginURL:         buildLoginURL(listen, entrance),
		BaseDir:          s.cfg.BaseDir,
		ConfigPath:       s.cfg.GoPanelConfigPath,
		DatabasePath:     dbPath,
		AtUnixMs:         time.Now().UnixMilli(),
		ResetPasswordTip: "Use gopanel --reset-password to reset SUPER admin password if forgotten.",
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

type goPanelConfSimple struct {
	System struct {
		Entrance string
	}
	HTTP struct {
		Listen string
	}
	DB struct {
		Database string
		Path     string
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
		case "http":
			if k == "listen" {
				c.HTTP.Listen = v
			}
		case "db":
			if k == "database" {
				c.DB.Database = v
			}
			if k == "path" {
				c.DB.Path = v
			}
		}
	}
	return c, nil
}

func queryAdminEmailBySqlite3(ctx context.Context, dbPath string) (string, error) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		return "", err
	}
	q := func(sqlStr string) (string, error) {
		c := exec.CommandContext(ctx, "sqlite3", "-batch", "-noheader", dbPath, sqlStr)
		out, err := c.CombinedOutput()
		s := strings.TrimSpace(string(out))
		if err != nil {
			if s == "" {
				return "", err
			}
			return "", errors.New(s)
		}
		return s, nil
	}
	email, err := q(`SELECT email FROM user WHERE status = 20 AND role = 'SUPER' LIMIT 1;`)
	if err == nil && strings.TrimSpace(email) != "" {
		return email, nil
	}
	email, err = q(`SELECT email FROM user WHERE status = 20 AND role = 'ADMIN' LIMIT 1;`)
	if err == nil && strings.TrimSpace(email) != "" {
		return email, nil
	}
	return "", errors.New("admin user not found")
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
