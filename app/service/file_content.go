package service

import (
	"errors"
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/dto/response"
	"github.com/aihop/gopanel/utils/files"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

func (f *FileService) GetContent(op request.FileContentReq) (response.FileInfo, error) {
	info, err := files.NewFileInfo(files.FileOption{Path: op.Path, Expand: true, IsDetail: op.IsDetail})
	if err != nil {
		if isPermissionErr(err) {
			return f.getContentViaGpc(op)
		}
		return response.FileInfo{}, err
	}
	content := []byte(info.Content)
	if len(content) > 1024 {
		content = content[:1024]
	}
	if !utf8.Valid(content) {
		_, decodeName, _ := charset.DetermineEncoding(content, "")
		if decodeName == "windows-1252" {
			reader := strings.NewReader(info.Content)
			item := transform.NewReader(reader, simplifiedchinese.GBK.NewDecoder())
			contents, err := io.ReadAll(item)
			if err != nil {
				return response.FileInfo{}, err
			}
			info.Content = string(contents)
		}
	}
	return response.FileInfo{FileInfo: *info}, nil
}
func (f *FileService) SaveContent(edit request.FileEdit) error {
	info, err := files.NewFileInfo(files.FileOption{Path: edit.Path, Expand: false})
	if err != nil {
		if isPermissionErr(err) {
			return f.writeContentViaGpc(edit.Path, edit.Content, 0)
		}
		return err
	}
	fo := files.NewFileOp()
	if err := fo.WriteFile(edit.Path, strings.NewReader(edit.Content), info.FileMode); err != nil {
		if isPermissionErr(err) {
			return f.writeContentViaGpc(edit.Path, edit.Content, int(info.FileMode.Perm()))
		}
		return err
	}
	return nil
}
func isPermissionErr(err error) bool {
	return err != nil && (os.IsPermission(err) || errors.Is(err, os.ErrPermission))
}
