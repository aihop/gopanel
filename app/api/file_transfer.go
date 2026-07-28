package api

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/dto/response"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/pkg/websocket"
	"github.com/aihop/gopanel/utils/files"
	websocket2 "github.com/aihop/gopanel/utils/websocket"
	"github.com/gofiber/fiber/v3"
)

var downloadCancelFuncs = sync.Map{}

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
	if err := requireFileAccess(c, req.Path, filepath.Join(req.Path, req.Name)); err != nil {
		return c.JSON(e.Fail(err))
	}
	key, err := fileService.Wget(*req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(response.FileWgetRes{Key: key}))
}

func WgetFileStream(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.FileWget](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := requireFileAccess(c, req.Path, filepath.Join(req.Path, req.Name)); err != nil {
		return c.JSON(e.Fail(err))
	}

	key, err := newFileTaskKey(c, "download_")
	if err != nil {
		return c.JSON(e.Fail(err))
	}
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

func WgetCancel(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.FileCancelReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := requireFileTaskAccess(c, req.Key, "download_"); err != nil {
		return c.JSON(e.Fail(err))
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

func WgetLogsStream(c fiber.Ctx) error {
	key := strings.TrimSpace(c.Query("key"))
	if key == "" {
		return c.JSON(e.Fail(errors.New("key is required")))
	}
	if err := requireFileTaskAccess(c, key, "download_"); err != nil {
		return c.JSON(e.Fail(err))
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
	if err := requireFileAccess(c, filePath); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.Download(filePath)
}

func DownloadChunkFiles(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.FileChunkDownload](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := requireFileAccess(c, req.Path); err != nil {
		return c.JSON(e.Fail(err))
	}
	fileOp := files.NewFileOp()
	if !fileOp.Stat(req.Path) {
		return c.JSON(e.Fail(errors.New("file not found")))
	}
	fstFile, err := fileOp.OpenFile(req.Path)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	defer fstFile.Close()
	info, err := fstFile.Stat()
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if info.IsDir() {
		return c.JSON(e.Fail(errors.New("path is a directory")))
	}
	c.Set(fiber.HeaderContentDisposition, "attachment; filename="+filepath.Base(req.Name))
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
	if len(ranges) != 2 {
		return c.Status(fiber.StatusRequestedRangeNotSatisfiable).Send(nil)
	}
	fileSize := info.Size()
	var start, end int64
	if ranges[0] != "" {
		start, err = strconv.ParseInt(ranges[0], 10, 64)
		if err != nil {
			return c.Status(fiber.StatusRequestedRangeNotSatisfiable).Send(nil)
		}
	}
	if ranges[1] == "" {
		end = fileSize - 1
	} else {
		end, err = strconv.ParseInt(ranges[1], 10, 64)
		if err != nil {
			return c.Status(fiber.StatusRequestedRangeNotSatisfiable).Send(nil)
		}
	}
	if start < 0 || start > end || start >= fileSize {
		return c.Status(fiber.StatusRequestedRangeNotSatisfiable).Send(nil)
	}
	if end >= fileSize {
		end = fileSize - 1
	}
	c.Set(fiber.HeaderContentRange, fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
	c.Status(fiber.StatusPartialContent)
	file, err := os.Open(req.Path)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"msg": err.Error()})
	}
	defer file.Close()
	if _, err := file.Seek(start, 0); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"msg": err.Error()})
	}
	return c.SendStream(file, int(end-start+1))
}
