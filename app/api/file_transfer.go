package api

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/dto/response"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/pkg/websocket"
	"github.com/aihop/gopanel/utils/common"
	"github.com/aihop/gopanel/utils/files"
	"github.com/aihop/gopanel/utils/token"
	websocket2 "github.com/aihop/gopanel/utils/websocket"
	"github.com/gofiber/fiber/v3"
)

// downloadCancelFuncs 存储下载任务的取消函数，key 为任务 key
var downloadCancelFuncs = sync.Map{}

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
		return c.JSON(e.Fail(errors.New("all files upload failed")))
	} else {
		return c.JSON(e.Succ(fmt.Sprintf("%d files upload success", success)))
	}
}
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
	wg.Wait()
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
func WgetFile(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.FileWget](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	key, err := fileService.Wget(*req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(response.FileWgetRes{Key: key}))
}

// WgetFileStream 异步下载远程文件，返回 task key，通过 /file/wget/logs 订阅 SSE 日志
func WgetFileStream(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.FileWget](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if claims, ok := c.Locals(constant.AppAuthName).(*token.CustomClaims); ok && claims.Role == constant.UserRoleSubAdmin {
		baseDir := filepath.Clean(claims.FileBaseDir)
		if !strings.HasPrefix(filepath.Clean(req.Path), baseDir) {
			return c.JSON(e.Fail(errors.New("permission denied: you can only access your designated workspace")))
		}
	}

	key := "download_" + common.RandStrAndNum(20)
	ctx, cancel := context.WithCancel(context.Background())
	downloadCancelFuncs.Store(key, cancel)

	logger := service.GetDownloadLogger(key)
	logger.Appendf("已提交远程下载任务：URL=%s，保存路径=%s/%s", req.Url, req.Path, req.Name)

	go func() {
		defer func() {
			downloadCancelFuncs.Delete(key)
			service.RemoveDownloadLogger(key)
		}()
		_ = fileService.WgetStream(ctx, *req, logger)
	}()

	return c.JSON(e.Succ(map[string]interface{}{"key": key}))
}

// WgetCancel 取消正在进行的远程下载任务
func WgetCancel(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.FileCancelReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if req.Key == "" {
		return c.JSON(e.Fail(errors.New("key is required")))
	}

	cancelVal, ok := downloadCancelFuncs.Load(req.Key)
	if !ok {
		return c.JSON(e.Fail(errors.New("task not found or already completed")))
	}

	cancel, ok := cancelVal.(context.CancelFunc)
	if !ok {
		return c.JSON(e.Fail(errors.New("invalid cancel function")))
	}

	cancel()
	return c.JSON(e.Succ(nil))
}

// WgetLogsStream SSE 实时推送远程下载任务的日志和进度
func WgetLogsStream(c fiber.Ctx) error {
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

		if !service.IsDownloadLoggerActive(key) {
			lines, err := service.ReadDownloadLogFromFile(key)
			if err == nil {
				for _, line := range lines {
					writeData(line)
				}
			}
			writeEvent("eof", "EOF")
			return
		}

		logger := service.GetDownloadLogger(key)
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
				case "progress":
					writeEvent("progress", fmt.Sprintf("%.2f", evt.Percent))
					if evt.Message != "" {
						writeData(evt.Message)
					}
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
		return c.SendFile(req.Path)
	}
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

// --- 断点续传逻辑 ---
