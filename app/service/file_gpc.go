package service

import (
	"context"
	"encoding/json"
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/dto/response"
	"github.com/aihop/gopanel/utils/files"
	"github.com/aihop/gopanel/utils/gpc"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"io"
	"strings"
	"time"
	"unicode/utf8"
)

func (f *FileService) getFileListViaGpc(op request.FileOption) (response.FileInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := gpc.Do(ctx, "FILE_LIST", map[string]interface{}{"path": op.Path, "page": op.Page, "limit": op.Limit, "sortBy": op.SortBy, "sortOrder": op.SortOrder, "showHidden": op.ShowHidden})
	if err != nil {
		return response.FileInfo{}, err
	}
	var out files.FileInfo
	if err := json.Unmarshal([]byte(resp.Output), &out); err != nil {
		return response.FileInfo{}, err
	}
	return response.FileInfo{FileInfo: out}, nil
}
func (f *FileService) getContentViaGpc(op request.FileContentReq) (response.FileInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := gpc.Do(ctx, "FILE_READ", map[string]interface{}{"path": op.Path})
	if err != nil {
		return response.FileInfo{}, err
	}
	var out files.FileInfo
	if err := json.Unmarshal([]byte(resp.Output), &out); err != nil {
		return response.FileInfo{}, err
	}
	content := []byte(out.Content)
	if len(content) > 1024 {
		content = content[:1024]
	}
	if !utf8.Valid(content) {
		_, decodeName, _ := charset.DetermineEncoding(content, "")
		if decodeName == "windows-1252" {
			reader := strings.NewReader(out.Content)
			item := transform.NewReader(reader, simplifiedchinese.GBK.NewDecoder())
			contents, err := io.ReadAll(item)
			if err != nil {
				return response.FileInfo{}, err
			}
			out.Content = string(contents)
		}
	}
	return response.FileInfo{FileInfo: out}, nil
}
func (f *FileService) writeContentViaGpc(path string, content string, mode int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	params := map[string]interface{}{"path": path, "content": content}
	if mode > 0 {
		params["mode"] = mode
	}
	_, err := gpc.Do(ctx, "FILE_WRITE", params)
	return err
}
func (f *FileService) mkdirViaGpc(path string, mode int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	params := map[string]interface{}{"path": path}
	if mode > 0 {
		params["mode"] = mode
	}
	_, err := gpc.Do(ctx, "FILE_MKDIR", params)
	return err
}
func (f *FileService) createFileViaGpc(path string, content string, mode int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	params := map[string]interface{}{"path": path, "content": content}
	if mode > 0 {
		params["mode"] = mode
	}
	_, err := gpc.Do(ctx, "FILE_CREATE", params)
	return err
}
func (f *FileService) removeViaGpc(path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := gpc.Do(ctx, "FILE_REMOVE", map[string]interface{}{"path": path})
	return err
}
func (f *FileService) chmodViaGpc(path string, mode int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := gpc.Do(ctx, "FILE_CHMOD", map[string]interface{}{"path": path, "mode": mode})
	return err
}
func (f *FileService) chownViaGpc(path string, user string, group string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := gpc.Do(ctx, "FILE_CHOWN", map[string]interface{}{"path": path, "user": user, "group": group})
	return err
}
