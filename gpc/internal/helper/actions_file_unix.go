//go:build !windows

package helper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type fileInfo struct {
	Path      string     `json:"path"`
	Name      string     `json:"name"`
	User      string     `json:"user"`
	Group     string     `json:"group"`
	Uid       string     `json:"uid"`
	Gid       string     `json:"gid"`
	Extension string     `json:"extension"`
	Content   string     `json:"content,omitempty"`
	Size      int64      `json:"size"`
	IsDir     bool       `json:"isDir"`
	IsSymlink bool       `json:"isSymlink"`
	IsHidden  bool       `json:"isHidden"`
	LinkPath  string     `json:"linkPath"`
	Type      string     `json:"type"`
	Mode      string     `json:"mode"`
	MimeType  string     `json:"mimeType"`
	ModTime   time.Time  `json:"modTime"`
	Items     []fileInfo `json:"items,omitempty"`
	ItemTotal int        `json:"itemTotal,omitempty"`
	IsDetail  bool       `json:"isDetail,omitempty"`
}

func (s *Server) actionFileStat(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	p := getString(params, "path")
	if p == "" {
		return "", errors.New("invalid params: path is empty")
	}
	abs, err := s.cleanAndCheckPath(p, true)
	if err != nil {
		return "", err
	}
	fi, err := os.Lstat(abs)
	if err != nil {
		return "", err
	}
	out := s.buildFileInfo(abs, fi, false)
	b, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (s *Server) actionFileList(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	p := getString(params, "path")
	if p == "" {
		return "", errors.New("invalid params: path is empty")
	}
	page, _ := getInt(params, "page")
	limit, _ := getInt(params, "limit")
	sortBy := strings.TrimSpace(getString(params, "sortBy"))
	sortOrder := strings.TrimSpace(getString(params, "sortOrder"))
	showHidden := strings.TrimSpace(getString(params, "showHidden"))

	abs, err := s.cleanAndCheckPath(p, true)
	if err != nil {
		return "", err
	}
	fi, err := os.Lstat(abs)
	if err != nil {
		return "", err
	}
	if !fi.IsDir() {
		return "", errors.New("path is not a directory")
	}

	entries, err := os.ReadDir(abs)
	if err != nil {
		return "", err
	}

	var items []fileInfo
	for _, ent := range entries {
		name := ent.Name()
		if name == "" {
			continue
		}
		if showHidden != "true" && strings.HasPrefix(name, ".") {
			continue
		}
		full := filepath.Join(abs, name)
		st, err := os.Lstat(full)
		if err != nil {
			continue
		}
		items = append(items, s.buildFileInfo(full, st, false))
	}

	sortItems(items, sortBy, sortOrder)

	total := len(items)
	if limit <= 0 {
		limit = total
	}
	if page <= 0 {
		page = 1
	}
	start := (page - 1) * limit
	if start < 0 {
		start = 0
	}
	end := start + limit
	if end > total {
		end = total
	}
	if start > total {
		start = total
	}
	paged := items[start:end]

	out := s.buildFileInfo(abs, fi, false)
	out.Items = paged
	out.ItemTotal = total
	out.IsDetail = true

	b, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (s *Server) actionFileRead(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	p := getString(params, "path")
	if p == "" {
		return "", errors.New("invalid params: path is empty")
	}
	abs, err := s.cleanAndCheckPath(p, true)
	if err != nil {
		return "", err
	}
	fi, err := os.Lstat(abs)
	if err != nil {
		return "", err
	}
	if fi.IsDir() {
		return "", errors.New("path is a directory")
	}

	maxN := s.cfg.MaxFileReadBytes
	if maxN <= 0 {
		maxN = 1048576
	}
	f, err := os.Open(abs)
	if err != nil {
		return "", err
	}
	defer f.Close()
	r := io.LimitReader(f, maxN+1)
	b, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	if int64(len(b)) > maxN {
		b = b[:maxN]
	}
	out := s.buildFileInfo(abs, fi, true)
	out.Content = string(b)

	j, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(j), nil
}

func (s *Server) actionFileWrite(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	p := getString(params, "path")
	if p == "" {
		return "", errors.New("invalid params: path is empty")
	}
	content := getString(params, "content")

	abs, err := s.cleanAndCheckPath(p, false)
	if err != nil {
		return "", err
	}

	maxN := s.cfg.MaxFileWriteBytes
	if maxN <= 0 {
		maxN = 1048576
	}
	if int64(len(content)) > maxN {
		return "", errors.New("invalid params: content too large")
	}

	mode := os.FileMode(0644)
	if st, err := os.Stat(abs); err == nil {
		mode = st.Mode().Perm()
	} else {
		if m, ok := getInt(params, "mode"); ok && m > 0 {
			mode = os.FileMode(m)
		}
	}

	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(abs, []byte(content), mode); err != nil {
		return "", err
	}
	return "ok", nil
}

func (s *Server) actionFileMkdir(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	p := getString(params, "path")
	if p == "" {
		return "", errors.New("invalid params: path is empty")
	}
	abs, err := s.cleanAndCheckPath(p, false)
	if err != nil {
		return "", err
	}
	mode := os.FileMode(0755)
	if m, ok := getInt(params, "mode"); ok && m > 0 {
		mode = os.FileMode(m)
	}
	if err := os.MkdirAll(abs, mode); err != nil {
		return "", err
	}
	return "ok", nil
}

func (s *Server) actionFileCreate(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	p := getString(params, "path")
	if p == "" {
		return "", errors.New("invalid params: path is empty")
	}
	abs, err := s.cleanAndCheckPath(p, false)
	if err != nil {
		return "", err
	}
	mode := os.FileMode(0644)
	if m, ok := getInt(params, "mode"); ok && m > 0 {
		mode = os.FileMode(m)
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		return "", err
	}
	if _, err := os.Stat(abs); err == nil {
		return "", errors.New("file already exists")
	}
	if err := os.WriteFile(abs, []byte(getString(params, "content")), mode); err != nil {
		return "", err
	}
	return "ok", nil
}

func (s *Server) actionFileRemove(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	p := getString(params, "path")
	if p == "" {
		return "", errors.New("invalid params: path is empty")
	}
	abs, err := s.cleanAndCheckPath(p, false)
	if err != nil {
		return "", err
	}
	if err := os.RemoveAll(abs); err != nil {
		return "", err
	}
	return "ok", nil
}

func (s *Server) actionFileChmod(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	p := getString(params, "path")
	if p == "" {
		return "", errors.New("invalid params: path is empty")
	}
	modeInt, ok := getInt(params, "mode")
	if !ok || modeInt <= 0 {
		return "", errors.New("invalid params: mode")
	}
	abs, err := s.cleanAndCheckPath(p, true)
	if err != nil {
		return "", err
	}
	if err := os.Chmod(abs, os.FileMode(modeInt)); err != nil {
		return "", err
	}
	return "ok", nil
}

func (s *Server) actionFileChown(ctx context.Context, params map[string]interface{}) (string, error) {
	_ = ctx
	p := getString(params, "path")
	if p == "" {
		return "", errors.New("invalid params: path is empty")
	}
	userName := strings.TrimSpace(getString(params, "user"))
	groupName := strings.TrimSpace(getString(params, "group"))
	if userName == "" || groupName == "" {
		return "", errors.New("invalid params: user/group")
	}
	u, err := user.Lookup(userName)
	if err != nil {
		return "", err
	}
	g, err := user.LookupGroup(groupName)
	if err != nil {
		return "", err
	}
	uid, _ := strconv.Atoi(u.Uid)
	gid, _ := strconv.Atoi(g.Gid)
	abs, err := s.cleanAndCheckPath(p, true)
	if err != nil {
		return "", err
	}
	if err := os.Chown(abs, uid, gid); err != nil {
		return "", err
	}
	return "ok", nil
}

func (s *Server) cleanAndCheckPath(p string, mustExist bool) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", errors.New("invalid params: path is empty")
	}
	clean := filepath.Clean(p)
	if !filepath.IsAbs(clean) {
		return "", errors.New("invalid params: path must be absolute")
	}

	if !s.cfg.AllowRootFS {
		blocked := []string{"/proc", "/sys", "/dev"}
		for _, b := range blocked {
			if clean == b || strings.HasPrefix(clean, b+string(os.PathSeparator)) {
				return "", errors.New("path is not allowed")
			}
		}
	}

	roots := s.cfg.FileRoots
	if len(roots) == 0 && strings.TrimSpace(s.cfg.BaseDir) != "" {
		roots = []string{s.cfg.BaseDir}
	}
	if len(roots) == 0 && !s.cfg.AllowRootFS {
		return "", errors.New("file_roots is empty")
	}
	if s.cfg.AllowRootFS {
		if mustExist {
			if _, err := os.Lstat(clean); err != nil {
				return "", err
			}
		}
		return clean, nil
	}

	checkPath := clean
	if !mustExist {
		checkPath = filepath.Dir(clean)
	}
	resolved, err := filepath.EvalSymlinks(checkPath)
	if err != nil {
		if mustExist {
			return "", err
		}
		resolved = checkPath
	}
	within := func(target string) bool {
		for _, r := range roots {
			r = strings.TrimSpace(r)
			if r == "" {
				continue
			}
			r = filepath.Clean(r)
			if target == r || strings.HasPrefix(target, r+string(os.PathSeparator)) {
				return true
			}
		}
		return false
	}
	if within(resolved) {
		if mustExist {
			if _, err := os.Lstat(clean); err != nil {
				return "", err
			}
		} else if full, err := filepath.EvalSymlinks(clean); err == nil && !within(full) {
			// 写入场景下，最终路径本身若是指向 roots 外的符号链接，
			// os.WriteFile 会跟随它以 root 身份写到允许根之外——这里显式拦截。
			return "", errors.New("path is outside allowed roots")
		}
		return clean, nil
	}
	return "", errors.New("path is outside allowed roots")
}

func (s *Server) buildFileInfo(abs string, fi os.FileInfo, withContent bool) fileInfo {
	out := fileInfo{
		Path:    abs,
		Name:    filepath.Base(abs),
		IsDir:   fi.IsDir(),
		Size:    fi.Size(),
		Mode:    fmt.Sprintf("%04o", fi.Mode().Perm()),
		ModTime: fi.ModTime(),
		Type:    "file",
	}
	if out.IsDir {
		out.Type = "dir"
	}
	if strings.HasPrefix(out.Name, ".") {
		out.IsHidden = true
	}
	out.Extension = strings.TrimPrefix(filepath.Ext(out.Name), ".")

	if fi.Mode()&os.ModeSymlink != 0 {
		out.IsSymlink = true
		if link, err := os.Readlink(abs); err == nil {
			out.LinkPath = link
		}
	}

	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		out.Uid = strconv.FormatUint(uint64(st.Uid), 10)
		out.Gid = strconv.FormatUint(uint64(st.Gid), 10)
		if u, err := user.LookupId(out.Uid); err == nil {
			out.User = u.Username
		}
		if g, err := user.LookupGroupId(out.Gid); err == nil {
			out.Group = g.Name
		}
	}
	if out.User == "" && out.Uid != "" {
		out.User = out.Uid
	}
	if out.Group == "" && out.Gid != "" {
		out.Group = out.Gid
	}
	if !withContent {
		out.Content = ""
	}
	return out
}

func sortItems(items []fileInfo, sortBy, sortOrder string) {
	desc := strings.ToLower(sortOrder) == "desc"
	key := strings.ToLower(sortBy)
	if key == "" {
		key = "name"
	}
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		var less bool
		switch key {
		case "size":
			less = a.Size < b.Size
		case "modtime", "mtime", "time":
			less = a.ModTime.Before(b.ModTime)
		case "type":
			less = a.Type < b.Type
		default:
			less = strings.ToLower(a.Name) < strings.ToLower(b.Name)
		}
		if desc {
			return !less
		}
		return less
	})
}
