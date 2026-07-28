package api

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/files"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

func UploadFiles(c fiber.Ctx) error {
	form, err := c.MultipartForm()
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	uploadFiles := form.File["file"]
	paths := form.Value["path"]
	if len(paths) == 0 {
		return c.JSON(e.Fail(errors.New("error paths in request")))
	}
	if err := requireFileAccess(c, paths[0]); err != nil {
		return c.JSON(e.Fail(err))
	}
	for _, file := range uploadFiles {
		if file.Filename != filepath.Base(file.Filename) {
			return c.JSON(e.Fail(errors.New("invalid upload filename")))
		}
		if err := requireFileAccess(c, filepath.Join(paths[0], file.Filename)); err != nil {
			return c.JSON(e.Fail(err))
		}
	}

	overwrite := true
	if values, ok := form.Value["overwrite"]; ok && len(values) > 0 {
		overwrite, _ = strconv.ParseBool(values[0])
	}
	dir := filepath.Clean(paths[0])
	if _, err = os.Stat(dir); err != nil && os.IsNotExist(err) {
		mode, modeErr := files.GetParentMode(dir)
		if modeErr != nil {
			return c.JSON(e.Fail(modeErr))
		}
		if err = os.MkdirAll(dir, mode); err != nil {
			return c.JSON(e.Fail(fmt.Errorf("mkdir %s failed, err: %v", dir, err)))
		}
	}
	info, err := os.Stat(dir)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	mode := info.Mode()
	fileOp := files.NewFileOp()
	uid, gid := files.GetUidGid(info)
	success := 0
	failures := make(buserr.MultiErr)
	for _, file := range uploadFiles {
		dstFilename := filepath.Join(dir, file.Filename)
		dstDir := filepath.Dir(dstFilename)
		if !fileOp.Stat(dstDir) {
			if err = fileOp.CreateDir(dstDir, mode); err != nil {
				fileErr := fmt.Errorf("create dir [%s] failed, err: %v", dstDir, err)
				failures[file.Filename] = fileErr
				global.LOG.Error(fileErr)
				continue
			}
			_ = os.Chown(dstDir, uid, gid)
		}
		tmpFile, tmpErr := os.CreateTemp(dstDir, "."+filepath.Base(dstFilename)+".upload-*")
		if tmpErr != nil {
			fileErr := fmt.Errorf("upload [%s] file failed, err: %v", file.Filename, tmpErr)
			failures[file.Filename] = fileErr
			global.LOG.Error(fileErr)
			continue
		}
		tmpFilename := tmpFile.Name()
		_ = tmpFile.Close()
		if err := c.SaveFile(file, tmpFilename); err != nil {
			_ = os.Remove(tmpFilename)
			fileErr := fmt.Errorf("upload [%s] file failed, err: %v", file.Filename, err)
			failures[file.Filename] = fileErr
			global.LOG.Error(fileErr)
			continue
		}
		dstInfo, statErr := os.Stat(dstFilename)
		if !overwrite && statErr == nil {
			_ = os.Remove(tmpFilename)
			fileErr := fmt.Errorf("upload [%s] file failed, destination already exists", file.Filename)
			failures[file.Filename] = fileErr
			global.LOG.Error(fileErr)
			continue
		}
		if overwrite {
			err = os.Rename(tmpFilename, dstFilename)
		} else {
			err = os.Link(tmpFilename, dstFilename)
			if err == nil {
				err = os.Remove(tmpFilename)
			}
		}
		if err != nil {
			_ = os.Remove(tmpFilename)
			fileErr := fmt.Errorf("upload [%s] file failed, err: %v", file.Filename, err)
			failures[file.Filename] = fileErr
			global.LOG.Error(fileErr)
			continue
		}
		if statErr == nil {
			_ = os.Chmod(dstFilename, dstInfo.Mode())
		} else {
			_ = os.Chmod(dstFilename, mode)
		}
		if uid != -1 && gid != -1 {
			_ = os.Chown(dstFilename, uid, gid)
		}
		success++
	}
	if success == 0 {
		return c.JSON(e.Fail(errors.New("all files upload failed")))
	}
	return c.JSON(e.Succ(fmt.Sprintf("%d files upload success", success)))
}

func UploadChunkFiles(c fiber.Ctx) error {
	fileForm, err := c.FormFile("chunk")
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	uploadFile, err := fileForm.Open()
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	defer uploadFile.Close()
	type chunkRequest struct {
		ChunkIndex string `form:"chunkIndex"`
		ChunkCount string `form:"chunkCount"`
		Filename   string `form:"filename"`
		Overwrite  string `form:"overwrite"`
		Path       string `form:"path"`
	}
	req := new(chunkRequest)
	if err = c.Bind().Body(req); err != nil {
		return c.JSON(e.Fail(err))
	}
	if req.Filename == "" || req.Filename != filepath.Base(req.Filename) {
		return c.JSON(e.Fail(errors.New("invalid upload filename")))
	}
	if err := requireFileAccess(c, req.Path, filepath.Join(req.Path, req.Filename)); err != nil {
		return c.JSON(e.Fail(err))
	}
	chunkIndex, err := strconv.Atoi(req.ChunkIndex)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	chunkCount, err := strconv.Atoi(req.ChunkCount)
	if err != nil || chunkIndex < 0 || chunkCount <= 0 || chunkIndex >= chunkCount {
		return c.JSON(e.Fail(errors.New("invalid chunk range")))
	}
	fileOp := files.NewFileOp()
	tmpDir := filepath.Join(global.CONF.System.TmpDir, "upload")
	if !fileOp.Stat(tmpDir) {
		if err := fileOp.CreateDir(tmpDir, 0o755); err != nil {
			return c.JSON(e.Fail(err))
		}
	}
	filename := req.Filename
	claims, _ := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	if claims == nil {
		return c.JSON(e.Fail(errors.New("unauthorized")))
	}
	fileDir := chunkUploadDir(tmpDir, claims.UserId, req.Path, filename)
	if chunkIndex == 0 {
		if fileOp.Stat(fileDir) {
			_ = fileOp.DeleteDir(fileDir)
		}
		_ = os.MkdirAll(fileDir, 0o755)
	}
	chunkData, err := io.ReadAll(uploadFile)
	if err != nil {
		return c.JSON(e.Fail(buserr.WithMap(constant.ErrFileUpload, map[string]interface{}{"name": filename, "detail": err.Error()})))
	}
	chunkPath := filepath.Join(fileDir, fmt.Sprintf("%s.%d", filename, chunkIndex))
	if err = os.WriteFile(chunkPath, chunkData, 0o644); err != nil {
		return c.JSON(e.Fail(buserr.WithMap(constant.ErrFileUpload, map[string]interface{}{"name": filename, "detail": err.Error()})))
	}
	if chunkIndex+1 != chunkCount {
		return c.JSON(e.Succ(false))
	}
	overwrite := true
	if req.Overwrite != "" {
		overwrite, _ = strconv.ParseBool(req.Overwrite)
	}
	if err = mergeChunks(filename, fileDir, req.Path, chunkCount, overwrite); err != nil {
		return c.JSON(e.Fail(buserr.WithMap(constant.ErrFileUpload, map[string]interface{}{"name": filename, "detail": err.Error()})))
	}
	return c.JSON(e.Succ(true))
}

func mergeChunks(fileName, fileDir, dstDir string, chunkCount int, overwrite bool) error {
	defer func() { _ = os.RemoveAll(fileDir) }()
	op := files.NewFileOp()
	dstDir = strings.TrimSpace(dstDir)
	mode, _ := files.GetParentMode(dstDir)
	if mode == 0 {
		mode = 0o755
	}
	uid, gid := -1, -1
	if info, err := os.Stat(dstDir); err != nil {
		if os.IsNotExist(err) {
			if err = op.CreateDir(dstDir, mode); err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		uid, gid = files.GetUidGid(info)
	}
	dstFileName := filepath.Join(dstDir, fileName)
	dstInfo, statErr := os.Stat(dstFileName)
	if !overwrite && statErr == nil {
		return errors.New("destination already exists")
	}
	if statErr == nil {
		mode = dstInfo.Mode()
	} else {
		mode = 0o644
	}
	targetFile, err := os.CreateTemp(dstDir, "."+fileName+".upload-*")
	if err != nil {
		return err
	}
	tmpTarget := targetFile.Name()
	defer func() {
		_ = targetFile.Close()
		_ = os.Remove(tmpTarget)
	}()
	if err := targetFile.Chmod(mode); err != nil {
		return err
	}
	for i := 0; i < chunkCount; i++ {
		chunkPath := filepath.Join(fileDir, fmt.Sprintf("%s.%d", fileName, i))
		chunkData, err := os.ReadFile(chunkPath)
		if err != nil {
			return err
		}
		if _, err = targetFile.Write(chunkData); err != nil {
			return err
		}
		_ = os.Remove(chunkPath)
	}
	if err := targetFile.Close(); err != nil {
		return err
	}
	if uid != -1 && gid != -1 {
		_ = os.Chown(tmpTarget, uid, gid)
	}
	if !overwrite {
		if err := os.Link(tmpTarget, dstFileName); err != nil {
			return err
		}
		return os.Remove(tmpTarget)
	}
	return os.Rename(tmpTarget, dstFileName)
}

func chunkUploadDir(tmpDir string, userID uint, dstDir, fileName string) string {
	uploadKey := fmt.Sprintf("%d:%s:%s", userID, filepath.Clean(dstDir), fileName)
	digest := sha256.Sum256([]byte(uploadKey))
	return filepath.Join(tmpDir, fmt.Sprintf("%x", digest))
}

func FileUploadSearch(c fiber.Ctx) error {
	R, err := e.BodyToStruct[request.SearchUploadWithPage](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := requireFileAccess(c, R.Path); err != nil {
		return c.JSON(e.Fail(err))
	}
	total, files, err := fileService.SearchUploadWithPage(R)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(dto.PageResult{
		Items: files,
		Total: total,
	}))
}
