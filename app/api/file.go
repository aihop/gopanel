package api

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/dto/response"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/pkg/websocket"
	"github.com/aihop/gopanel/utils/files"
	"github.com/aihop/gopanel/utils/token"
	websocket2 "github.com/aihop/gopanel/utils/websocket"
	"github.com/gofiber/fiber/v3"
)

// @Tags File
// @Summary List files
// @Accept json
// @Param request body request.FileOption true "request"
// @Success 200 {object} response.FileInfo
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /files/search [post]
func ListFiles(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.FileOption](c.Body())
	if err != nil {
		return c.JSON(e.Result(err))
	}

	// 处理如果 SubAdmin 访问根目录则重定向到它的限制目录
	if claims, ok := c.Locals(constant.AppAuthName).(*token.CustomClaims); ok && (claims.Role == constant.UserRoleSubAdmin || claims.Role == constant.UserRoleDemo) {
		baseDir := filepath.Clean(claims.FileBaseDir)
		if req.Path == "" || req.Path == "/" || strings.HasSuffix(req.Path, "pipelines") || strings.HasSuffix(req.Path, "pipelines_archive") {
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

// @Tags File
// @Summary Create file
// @Accept json
// @Param request body request.FileCreate true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /files [post]
// @x-panel-log {"bodyKeys":["path"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"创建文件/文件夹 [path]","formatEN":"Create dir or file [path]"}
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

// @Tags File
// @Summary Delete file
// @Accept json
// @Param request body request.FileDelete true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /files/del [post]
// @x-panel-log {"bodyKeys":["path"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"删除文件/文件夹 [path]","formatEN":"Delete dir or file [path]"}
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

// @Tags File
// @Summary Batch delete file
// @Accept json
// @Param request body request.FileBatchDelete true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /files/batch/del [post]
// @x-panel-log {"bodyKeys":["paths"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"批量删除文件/文件夹 [paths]","formatEN":"Batch delete dir or file [paths]"}
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

// @Tags File
// @Summary Change file mode
// @Accept json
// @Param request body request.FileCreate true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /files/mode [post]
// @x-panel-log {"bodyKeys":["path","mode"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"修改权限 [paths] => [mode]","formatEN":"Change mode [paths] => [mode]"}
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

// @Tags File
// @Summary Change file owner
// @Accept json
// @Param request body request.FileRoleUpdate true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /files/owner [post]
// @x-panel-log {"bodyKeys":["path","user","group"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"修改用户/组 [paths] => [user]/[group]","formatEN":"Change owner [paths] => [user]/[group]"}
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

// @Tags File
// @Summary Compress file
// @Accept json
// @Param request body request.FileCompress true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /files/compress [post]
// @x-panel-log {"bodyKeys":["name"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"压缩文件 [name]","formatEN":"Compress file [name]"}
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

// @Tags File
// @Summary Decompress file
// @Accept json
// @Param request body request.FileDeCompress true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /files/decompress [post]
// @x-panel-log {"bodyKeys":["path"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"解压 [path]","formatEN":"Decompress file [path]"}
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

// @Tags File
// @Summary Load file content
// @Accept json
// @Param request body request.FileContentReq true "request"
// @Success 200 {object} response.FileInfo
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /files/content [post]
// @x-panel-log {"bodyKeys":["path"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"获取文件内容 [path]","formatEN":"Load file content [path]"}
func GetContent(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.FileContentReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if claims, ok := c.Locals(constant.AppAuthName).(*token.CustomClaims); ok && claims.Role == constant.UserRoleSubAdmin {
		baseDir := filepath.Clean(claims.FileBaseDir)
		if !strings.HasPrefix(filepath.Clean(req.Path), baseDir) {
			return c.JSON(e.Fail(errors.New("permission denied: you can only access your designated workspace")))
		}
	}

	info, err := fileService.GetContent(*req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(info))
}

// @Tags File
// @Summary Update file content
// @Accept json
// @Param request body request.FileEdit true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /files/save [post]
// @x-panel-log {"bodyKeys":["path"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"更新文件内容 [path]","formatEN":"Update file content [path]"}
func SaveContent(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.FileEdit](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if claims, ok := c.Locals(constant.AppAuthName).(*token.CustomClaims); ok && claims.Role == constant.UserRoleSubAdmin {
		baseDir := filepath.Clean(claims.FileBaseDir)
		if !strings.HasPrefix(filepath.Clean(req.Path), baseDir) {
			return c.JSON(e.Fail(errors.New("permission denied: you can only access your designated workspace")))
		}
	}
	if err := fileService.SaveContent(*req); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(nil))
}

// @Tags File
// @Summary Read file by Line
// @Param request body request.FileReadByLineReq true "request"
// @Success 200 {object} response.FileLineContent
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /files/read [post]
func ReadFileByLine(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.FileReadByLineReq](c.Body())
	if err != nil {
		return c.JSON(e.RetError(constant.CodeErrBadRequest, err.Error()))
	}

	res, err := fileService.ReadLogByLine(*req)
	if err != nil {
		return c.JSON(e.RetError(constant.CodeErrInternalServer, err.Error()))
	}
	return c.JSON(e.Succ(res))
}

func DirExist(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.DirExistReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	// 检查目录是否存在
	if _, err := os.Stat(req.Dir); os.IsNotExist(err) {
		return c.JSON(e.Succ(map[string]bool{"exist": false}))
	} else if err != nil {
		return c.JSON(e.RetError(constant.CodeErrInternalServer, err.Error()))
	}
	return c.JSON(e.Succ(map[string]bool{"exist": true}))
}

// @Tags File
// @Summary Check file exist
// @Accept json
// @Param request body request.FilePathCheck true "request"
// @Success 200 {boolean} isOk
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /files/check [post]
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

// @Tags File
// @Summary Batch check file exist
// @Accept json
// @Param request body request.FilePathsCheck true "request"
// @Success 200 {array} response.ExistFileInfo
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /files/batch/check [post]
func BatchCheckFiles(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.FilePathsCheck](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	fileList := fileService.BatchCheckFiles(*req)
	return c.JSON(e.Succ(fileList))
}

// @Tags File
// @Summary Upload file
// @Accept multipart/form-data
// @Param file formData file true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /files/upload [post]
// @x-panel-log {"bodyKeys":["path"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"上传文件 [path]","formatEN":"Upload file [path]"}
func UploadFiles(c fiber.Ctx) error {
	form, err := c.MultipartForm()
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	uploadFiles := form.File["file"]
	paths := form.Value["path"]

	overwrite := true
	if ow, ok := form.Value["overwrite"]; ok {
		if len(ow) != 0 {
			parseBool, _ := strconv.ParseBool(ow[0])
			overwrite = parseBool
		}
	}

	if len(paths) == 0 || !strings.Contains(paths[0], "/") {
		return c.JSON(e.Fail(errors.New("error paths in request")))
	}

	if claims, ok := c.Locals(constant.AppAuthName).(*token.CustomClaims); ok && claims.Role == constant.UserRoleSubAdmin {
		baseDir := filepath.Clean(claims.FileBaseDir)
		if !strings.HasPrefix(filepath.Clean(paths[0]), baseDir) {
			return c.JSON(e.Fail(errors.New("permission denied: you can only access your designated workspace")))
		}
	}
	dir := path.Dir(paths[0])

	_, err = os.Stat(dir)
	if err != nil && os.IsNotExist(err) {
		mode, err := files.GetParentMode(dir)
		if err != nil {
			return c.JSON(e.Fail(err))
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
		dstFilename := path.Join(paths[0], file.Filename)
		dstDir := path.Dir(dstFilename)
		if !fileOp.Stat(dstDir) {
			if err = fileOp.CreateDir(dstDir, mode); err != nil {
				e := fmt.Errorf("create dir [%s] failed, err: %v", path.Dir(dstFilename), err)
				failures[file.Filename] = e
				global.LOG.Error(e)
				continue
			}
			_ = os.Chown(dstDir, uid, gid)
		}
		tmpFilename := dstFilename + ".tmp"

		if err := c.SaveFile(file, tmpFilename); err != nil {
			_ = os.Remove(tmpFilename)
			e := fmt.Errorf("upload [%s] file failed, err: %v", file.Filename, err)
			failures[file.Filename] = e
			global.LOG.Error(e)
			continue
		}
		dstInfo, statErr := os.Stat(dstFilename)
		if overwrite {
			_ = os.Remove(dstFilename)
		}

		err = os.Rename(tmpFilename, dstFilename)
		if err != nil {
			_ = os.Remove(tmpFilename)
			e := fmt.Errorf("upload [%s] file failed, err: %v", file.Filename, err)
			failures[file.Filename] = e
			global.LOG.Error(e)
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
		// helper.ErrorWithDetail(c, constant.CodeErrInternalServer, constant.ErrTypeInternalServer, failures)
		return c.JSON(e.Fail(errors.New("all files upload failed")))
	} else {
		return c.JSON(e.Succ(fmt.Sprintf("%d files upload success", success)))
	}
}

// @Tags File
// @Summary Chunk upload file
// @Accept multipart/form-data
// @Param file formData file true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /files/chunkUpload [post]
func UploadChunkFiles(c fiber.Ctx) error {
	var err error
	fileForm, err := c.FormFile("chunk")
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	uploadFile, err := fileForm.Open()
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	defer uploadFile.Close()

	type Chunk struct {
		ChunkIndex string `json:"chunkIndex" xml:"chunkIndex" form:"chunkIndex"`
		ChunkCount string `json:"chunkCount" xml:"chunkCount" form:"chunkCount"`
		Filename   string `json:"filename" xml:"filename" form:"filename"`
		Overwrite  string `json:"overwrite" xml:"overwrite" form:"overwrite"`
		Path       string `json:"path" xml:"path" form:"path"`
	}
	req := new(Chunk)
	if err = c.Bind().Body(req); err != nil {
		return c.JSON(e.Fail(err))
	}

	chunkIndex, err := strconv.Atoi(req.ChunkIndex)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	chunkCount, err := strconv.Atoi(req.ChunkCount)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	fileOp := files.NewFileOp()
	tmpDir := path.Join(global.CONF.System.TmpDir, "upload")
	if !fileOp.Stat(tmpDir) {
		if err := fileOp.CreateDir(tmpDir, 0755); err != nil {
			return c.JSON(e.Fail(err))
		}
	}
	filename := req.Filename
	fileDir := filepath.Join(tmpDir, filename)
	if chunkIndex == 0 {
		if fileOp.Stat(fileDir) {
			_ = fileOp.DeleteDir(fileDir)
		}
		_ = os.MkdirAll(fileDir, 0755)
	}
	filePath := filepath.Join(fileDir, filename)

	defer func() {
		if err != nil {
			_ = os.Remove(fileDir)
		}
	}()
	var (
		emptyFile *os.File
		chunkData []byte
	)

	emptyFile, err = os.Create(filePath)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	defer emptyFile.Close()

	chunkData, err = io.ReadAll(uploadFile)
	if err != nil {
		return c.JSON(e.Fail(buserr.WithMap(constant.ErrFileUpload, map[string]interface{}{"name": filename, "detail": err.Error()})))
	}

	chunkPath := filepath.Join(fileDir, fmt.Sprintf("%s.%d", filename, chunkIndex))
	err = os.WriteFile(chunkPath, chunkData, 0644)
	if err != nil {
		return c.JSON(e.Fail(buserr.WithMap(constant.ErrFileUpload, map[string]interface{}{"name": filename, "detail": err.Error()})))
	}

	if chunkIndex+1 == chunkCount {
		overwrite := true
		if ow := req.Overwrite; ow != "" {
			overwrite, _ = strconv.ParseBool(ow)
		}
		err = mergeChunks(filename, fileDir, req.Path, chunkCount, overwrite)
		if err != nil {
			return c.JSON(e.Fail(buserr.WithMap(constant.ErrFileUpload, map[string]interface{}{"name": filename, "detail": err.Error()})))
		}
		return c.JSON(e.Succ(true))
	} else {
		return c.JSON(e.Succ(false))
	}
}

func Ws(c *websocket.Conn) {
	wsClient := websocket2.NewWsClient("fileClient", c)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		wsClient.Read()
	}()
	go func() {
		defer wg.Done()
		wsClient.Write()
	}()
	wg.Wait() // 等待读写goroutine完成
}

func Keys(c fiber.Ctx) error {
	res := &response.FileProcessKeys{}
	keys, err := global.CACHE.PrefixScanKey("file-wget-")
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	res.Keys = keys
	return c.JSON(e.Succ(res))
}

// @Tags File
// @Summary Change file name
// @Accept json
// @Param request body request.FileRename true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /files/rename [post]
// @x-panel-log {"bodyKeys":["oldName","newName"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"重命名 [oldName] => [newName]","formatEN":"Rename [oldName] => [newName]"}
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

// @Tags File
// @Summary Wget file
// @Accept json
// @Param request body request.FileWget true "request"
// @Success 200 {object} response.FileWgetRes
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /files/wget [post]
// @x-panel-log {"bodyKeys":["url","path","name"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"下载 url => [path]/[name]","formatEN":"Download url => [path]/[name]"}
func WgetFile(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.FileWget](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	key, err := fileService.Wget(*req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(response.FileWgetRes{
		Key: key,
	}))

}

// @Tags File
// @Summary Move file
// @Accept json
// @Param request body request.FileMove true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /files/move [post]
// @x-panel-log {"bodyKeys":["oldPaths","newPath"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"移动文件 [oldPaths] => [newPath]","formatEN":"Move [oldPaths] => [newPath]"}
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

// @Tags File
// @Summary Download file
// @Accept json
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /files/download [get]
func Download(c fiber.Ctx) error {
	filePath := c.Query("path")
	if claims, ok := c.Locals(constant.AppAuthName).(*token.CustomClaims); ok && claims.Role == constant.UserRoleSubAdmin {
		baseDir := filepath.Clean(claims.FileBaseDir)
		if !strings.HasPrefix(filepath.Clean(filePath), baseDir) {
			return c.JSON(e.Fail(errors.New("permission denied: you can only access your designated workspace")))
		}
	}
	return c.Download(filePath)
}

// @Tags File
// @Summary Chunk Download file
// @Accept json
// @Param request body request.FileDownload true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /files/chunkdownload [post]
// @x-panel-log {"bodyKeys":["name"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"下载文件 [name]","formatEN":"Download file [name]"}
func DownloadChunkFiles(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.FileChunkDownload](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	fileOp := files.NewFileOp()
	if !fileOp.Stat(req.Path) {
		return c.JSON(e.Fail(err))
	}
	filePath := req.Path
	fstFile, err := fileOp.OpenFile(filePath)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	info, err := fstFile.Stat()
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if info.IsDir() {
		return c.JSON(e.Fail(err))
	}

	c.Set(fiber.HeaderContentDisposition, "attachment; filename="+req.Name)
	c.Set(fiber.HeaderContentType, "application/octet-stream")
	c.Set(fiber.HeaderAcceptRanges, "bytes")

	rangeHeader := c.Get(fiber.HeaderRange)
	if rangeHeader == "" {
		// 普通下载
		return c.SendFile(req.Path)
	}

	// --- 断点续传逻辑 ---
	const prefix = "bytes="
	if !strings.HasPrefix(rangeHeader, prefix) {
		return c.Status(fiber.StatusRequestedRangeNotSatisfiable).Send(nil)
	}

	ranges := strings.SplitN(strings.TrimPrefix(rangeHeader, prefix), "-", 2)
	fileSize := info.Size()

	var start, end int64
	if ranges[0] == "" {
		start = 0
	} else {
		start, _ = strconv.ParseInt(ranges[0], 10, 64)
	}
	if ranges[1] == "" {
		end = fileSize - 1
	} else {
		end, _ = strconv.ParseInt(ranges[1], 10, 64)
	}
	if start > end || start >= fileSize {
		return c.Status(fiber.StatusRequestedRangeNotSatisfiable).Send(nil)
	}
	c.Set(fiber.HeaderContentRange, fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
	c.Status(fiber.StatusPartialContent)

	f, err := os.Open(req.Path)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"msg": err.Error()})
	}
	defer f.Close()

	if _, err := f.Seek(start, 0); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"msg": err.Error()})
	}

	return c.SendStream(f, int(end-start+1))
}

// @Tags File
// @Summary Load file size
// @Accept json
// @Param request body request.DirSizeReq true "request"
// @Success 200 {object} response.DirSizeRes
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /files/size [post]
// @x-panel-log {"bodyKeys":["path"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"获取文件夹大小 [path]","formatEN":"Load file size [path]"}
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

func mergeChunks(fileName string, fileDir string, dstDir string, chunkCount int, overwrite bool) error {
	defer func() {
		_ = os.RemoveAll(fileDir)
	}()

	op := files.NewFileOp()
	dstDir = strings.TrimSpace(dstDir)
	mode, _ := files.GetParentMode(dstDir)
	if mode == 0 {
		mode = 0755
	}
	uid, gid := -1, -1
	if info, err := os.Stat(dstDir); err != nil {
		if os.IsNotExist(err) {
			if err = op.CreateDir(dstDir, mode); err != nil {
				return err
			}
		}
	} else {
		uid, gid = files.GetUidGid(info)
	}
	dstFileName := filepath.Join(dstDir, fileName)
	dstInfo, statErr := os.Stat(dstFileName)
	if statErr == nil {
		mode = dstInfo.Mode()
	} else {
		mode = 0644
	}

	if overwrite {
		_ = os.Remove(dstFileName)
	}
	targetFile, err := os.OpenFile(dstFileName, os.O_RDWR|os.O_CREATE, mode)
	if err != nil {
		return err
	}
	defer targetFile.Close()
	for i := 0; i < chunkCount; i++ {
		chunkPath := filepath.Join(fileDir, fmt.Sprintf("%s.%d", fileName, i))
		chunkData, err := os.ReadFile(chunkPath)
		if err != nil {
			return err
		}
		_, err = targetFile.Write(chunkData)
		if err != nil {
			return err
		}
		_ = os.Remove(chunkPath)
	}
	if uid != -1 && gid != -1 {
		_ = os.Chown(dstFileName, uid, gid)
	}

	return nil
}

// @Tags File
// @Summary Batch change file mode and owner
// @Accept json
// @Param request body request.FileRoleReq true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /files/batch/role [post]
// @x-panel-log {"bodyKeys":["paths","mode","user","group"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"批量修改文件权限和用户/组 [paths] => [mode]/[user]/[group]","formatEN":"Batch change file mode and owner [paths] => [mode]/[user]/[group]"}
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
