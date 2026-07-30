//go:build !windows

package helper

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

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
