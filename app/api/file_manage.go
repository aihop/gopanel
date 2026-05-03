package api

import (
	"errors"
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/files"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
	"os"
	"path/filepath"
	"strings"
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
	err = fileService.Compress(*req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(nil))
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
