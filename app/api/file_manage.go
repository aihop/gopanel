package api

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/common"
	"github.com/aihop/gopanel/utils/files"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

func ListFiles(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.FileOption](c.Body())
	if err != nil {
		return c.JSON(e.Result(err))
	}
	if claims, ok := c.Locals(constant.AppAuthName).(*token.CustomClaims); ok && (claims.Role == constant.UserRoleSubAdmin || claims.Role == constant.UserRoleDemo) {
		baseDir := filepath.Clean(claims.FileBaseDir)
		if req.Path == "" || req.Path == "/" || strings.HasSuffix(req.Path, "pipelines") {
			req.Path = baseDir
		} else {
			if !strings.HasPrefix(filepath.Clean(req.Path), baseDir) {
				req.Path = baseDir
			}
		}
	}
	fileList, err := fileService.GetFileList(*req)
	if err != nil {
		return c.JSON(e.Result(err))
	}
	return c.JSON(e.Succ(fileList))
}
func CreateFile(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.FileCreate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if claims, ok := c.Locals(constant.AppAuthName).(*token.CustomClaims); ok && claims.Role == constant.UserRoleSubAdmin {
		baseDir := filepath.Clean(claims.FileBaseDir)
		if !strings.HasPrefix(filepath.Clean(req.Path), baseDir) {
			return c.JSON(e.Fail(errors.New("permission denied: you can only access your designated workspace")))
		}
	}
	err = fileService.Create(*req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(nil))
}
func DeleteFile(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.FileDelete](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if claims, ok := c.Locals(constant.AppAuthName).(*token.CustomClaims); ok && claims.Role == constant.UserRoleSubAdmin {
		baseDir := filepath.Clean(claims.FileBaseDir)
		if !strings.HasPrefix(filepath.Clean(req.Path), baseDir) {
			return c.JSON(e.Fail(errors.New("permission denied: you can only access your designated workspace")))
		}
	}
	err = fileService.Delete(*req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(nil))
}
func BatchDeleteFile(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.FileBatchDelete](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if claims, ok := c.Locals(constant.AppAuthName).(*token.CustomClaims); ok && claims.Role == constant.UserRoleSubAdmin {
		baseDir := filepath.Clean(claims.FileBaseDir)
		for _, p := range req.Paths {
			if !strings.HasPrefix(filepath.Clean(p), baseDir) {
				return c.JSON(e.Fail(errors.New("permission denied: you can only access your designated workspace")))
			}
		}
	}
	err = fileService.BatchDelete(*req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(nil))
}
func ChangeFileMode(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.FileCreate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	err = fileService.ChangeMode(*req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(nil))
}
func ChangeFileOwner(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.FileRoleUpdate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := fileService.ChangeOwner(*req); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(nil))
}

// CompressFile 发起压缩任务，返回任务 key，客户端通过 /file/compress/logs?key=xxx 订阅实时日志
func CompressFile(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.FileCompress](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if claims, ok := c.Locals(constant.AppAuthName).(*token.CustomClaims); ok && claims.Role == constant.UserRoleSubAdmin {
		baseDir := filepath.Clean(claims.FileBaseDir)
		if !strings.HasPrefix(filepath.Clean(req.Dst), baseDir) {
			return c.JSON(e.Fail(errors.New("permission denied: you can only access your designated workspace")))
		}
		for _, f := range req.Files {
			if !strings.HasPrefix(filepath.Clean(f), baseDir) {
				return c.JSON(e.Fail(errors.New("permission denied: you can only access your designated workspace")))
			}
		}
	}

	key := "compress_" + common.RandStrAndNum(20)
	logger := service.GetFileCompressLogger(key)
	logger.Appendf("已提交压缩任务：类型=%s，目标=%s/%s，文件数=%d", req.Type, req.Dst, req.Name, len(req.Files))

	go func() {
		defer func() {
			service.RemoveFileCompressLogger(key)
		}()

		_ = fileService.CompressStream(*req, logger)
	}()

	return c.JSON(e.Succ(map[string]interface{}{"key": key}))
}

// FileCompressLogs SSE 实时推送压缩任务的日志和状态
func FileCompressLogs(c fiber.Ctx) error {
	key := strings.TrimSpace(c.Query("key"))
	if key == "" {
		return c.JSON(e.Fail(errors.New("key is required")))
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	c.RequestCtx().SetBodyStreamWriter(func(w *bufio.Writer) {
		writeData := func(data string) {
			_, _ = fmt.Fprintf(w, "data: %s\n\n", strings.ReplaceAll(data, "\n", " "))
			_ = w.Flush()
		}
		writeEvent := func(event, data string) {
			_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, strings.ReplaceAll(data, "\n", " "))
			_ = w.Flush()
		}

		if !service.IsFileCompressLoggerActive(key) {
			lines, err := service.ReadFileCompressLogFromFile(key)
			if err == nil {
				for _, line := range lines {
					writeData(line)
				}
			}
			writeEvent("eof", "EOF")
			return
		}

		logger := service.GetFileCompressLogger(key)
		for _, line := range logger.GetLogs() {
			writeData(line)
		}
		writeEvent("status", logger.GetStatus())

		ch := logger.Subscribe()
		defer logger.Unsubscribe(ch)

		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-c.Context().Done():
				return
			case <-ticker.C:
				_, _ = fmt.Fprintf(w, "event: ping\ndata: ping\n\n")
				_ = w.Flush()
			case evt, ok := <-ch:
				if !ok {
					return
				}
				switch evt.Type {
				case "log":
					writeData(evt.Message)
				case "status":
					writeEvent("status", evt.Status)
				case "eof":
					writeEvent("eof", "EOF")
					return
				default:
					if evt.Message != "" {
						writeData(evt.Message)
					}
				}
			}
		}
	})

	return nil
}

func DeCompressFile(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.FileDeCompress](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	err = fileService.DeCompress(*req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(nil))
}
func DirExist(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.DirExistReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if _, err := os.Stat(req.Dir); os.IsNotExist(err) {
		return c.JSON(e.Succ(map[string]bool{"exist": false}))
	} else if err != nil {
		return c.JSON(e.RetError(constant.CodeErrInternalServer, err.Error()))
	}
	return c.JSON(e.Succ(map[string]bool{"exist": true}))
}
func CheckFile(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.FilePathCheck](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	fileOp := files.NewFileOp()
	if fileOp.Stat(req.Path) {
		return c.JSON(e.Succ(true))
	}
	if req.WithInit {
		if err := fileOp.CreateDir(req.Path, 0644); err != nil {
			return c.JSON(e.Succ(false))
		}
		return c.JSON(e.Succ(true))
	}
	return c.JSON(e.Succ(false))
}
func BatchCheckFiles(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.FilePathsCheck](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	fileList := fileService.BatchCheckFiles(*req)
	return c.JSON(e.Succ(fileList))
}
func ChangeFileName(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.FileRename](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if claims, ok := c.Locals(constant.AppAuthName).(*token.CustomClaims); ok && claims.Role == constant.UserRoleSubAdmin {
		baseDir := filepath.Clean(claims.FileBaseDir)
		if !strings.HasPrefix(filepath.Clean(req.OldName), baseDir) || !strings.HasPrefix(filepath.Clean(req.NewName), baseDir) {
			return c.JSON(e.Fail(errors.New("permission denied: you can only access your designated workspace")))
		}
	}
	if err := fileService.ChangeName(*req); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(nil))
}
func MoveFile(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.FileMove](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := fileService.MvFile(*req); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(nil))
}
func Size(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.DirSizeReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	res, err := fileService.DirSize(*req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(res))
}
func BatchChangeModeAndOwner(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.FileRoleReq](c.Body())
	if err != nil {
		return c.JSON(e.Result(buserr.Err(err)))
	}
	if err := fileService.BatchChangeModeAndOwner(*req); err != nil {
		return c.JSON(e.Error(err))
	}
	return c.JSON(e.Succ(nil))
}
