package api

import (
	"bufio"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/gofiber/fiber/v3"
)

func requirePipelineManagePermission(c fiber.Ctx) error {
	claims, err := middleware.JwtClaims(c)
	if err != nil {
		return err
	}
	if claims.Role != constant.UserRoleAdmin && claims.Role != constant.UserRoleSuper {
		return errors.New("permission denied")
	}
	return nil
}

func PipelinePage(c fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("pageSize", "10"))

	pipelineRepo := repo.NewPipeline(global.DB)
	total, list, err := pipelineRepo.Page(page, pageSize)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if claims, err := middleware.JwtClaims(c); err == nil && claims.Role == constant.UserRoleSubAdmin {
		list = sanitizePipelineListForSubAdmin(list)
	}
	return c.JSON(e.Succ(fiber.Map{
		"total": total,
		"items": list,
	}))
}

func sanitizePipelineListForSubAdmin(list []model.Pipeline) []model.Pipeline {
	if len(list) == 0 {
		return list
	}
	sanitized := make([]model.Pipeline, 0, len(list))
	for _, item := range list {
		item.AuthData = ""
		item.BuildScript = ""
		item.RepoUrl = ""
		sanitized = append(sanitized, item)
	}
	return sanitized
}

func PipelineCreate(c fiber.Ctx) error {
	if err := requirePipelineManagePermission(c); err != nil {
		return c.JSON(e.Auth(err.Error()))
	}
	req, err := e.BodyToStruct[request.PipelineCreate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	pipelineRepo := repo.NewPipeline(global.DB)
	pipeline := &model.Pipeline{
		Name:         req.Name,
		Description:  req.Description,
		RepoUrl:      req.RepoUrl,
		Branch:       req.Branch,
		Version:      req.Version,
		AuthType:     req.AuthType,
		AuthData:     req.AuthData,
		BuildImage:   req.BuildImage,
		BuildScript:  req.BuildScript,
		OutputImage:  req.OutputImage,
		ArtifactPath: req.ArtifactPath,
		ExposePort:   req.ExposePort,
	}

	if err := pipelineRepo.Create(pipeline); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

func PipelineUpdate(c fiber.Ctx) error {
	if err := requirePipelineManagePermission(c); err != nil {
		return c.JSON(e.Auth(err.Error()))
	}
	req, err := e.BodyToStruct[request.PipelineUpdate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	pipelineRepo := repo.NewPipeline(global.DB)
	pipeline, err := pipelineRepo.Get(req.ID)
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	pipeline.Name = req.Name
	pipeline.Description = req.Description
	pipeline.RepoUrl = req.RepoUrl
	pipeline.Branch = req.Branch
	pipeline.Version = req.Version
	pipeline.AuthType = req.AuthType
	pipeline.AuthData = req.AuthData
	pipeline.BuildImage = req.BuildImage
	pipeline.BuildScript = req.BuildScript
	pipeline.OutputImage = req.OutputImage
	pipeline.ArtifactPath = req.ArtifactPath
	pipeline.ExposePort = req.ExposePort

	if err := pipelineRepo.Update(pipeline); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

func PipelineDelete(c fiber.Ctx) error {
	if err := requirePipelineManagePermission(c); err != nil {
		return c.JSON(e.Auth(err.Error()))
	}
	id, err := strconv.Atoi(c.Query("id"))
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	pipelineRepo := repo.NewPipeline(global.DB)
	if err := pipelineRepo.Delete(uint(id)); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

func PipelineRun(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.PipelineRun](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	pipelineSvc := service.NewPipelineService(global.DB)
	recordID, err := pipelineSvc.RunPipeline(req.ID, req.Version)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{
		"recordId": recordID,
	}))
}

func PipelineStop(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.CommonID](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	service.StopPipeline(req.ID)
	return c.JSON(e.Succ())
}

func PipelineRecordPage(c fiber.Ctx) error {
	pipelineId, _ := strconv.Atoi(c.Query("pipelineId"))
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("pageSize", "10"))

	recordRepo := repo.NewPipelineRecord(global.DB)
	total, list, err := recordRepo.PageByPipeline(uint(pipelineId), page, pageSize)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{
		"total": total,
		"items": list,
	}))
}

func PipelineRecordDelete(c fiber.Ctx) error {
	if err := requirePipelineManagePermission(c); err != nil {
		return c.JSON(e.Auth(err.Error()))
	}
	recordId, err := strconv.Atoi(c.Query("id"))
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	recordRepo := repo.NewPipelineRecord(global.DB)
	record, err := recordRepo.Get(uint(recordId))
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	if record.Status == "pending" || record.Status == "cloning" || record.Status == "building" || record.Status == "deploying" {
		return c.JSON(e.Fail(fmt.Errorf("执行中的记录不允许删除")))
	}

	if err := recordRepo.Delete(uint(recordId)); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

func PipelineLogs(c fiber.Ctx) error {
	recordId, err := strconv.Atoi(c.Query("recordId"))
	if err != nil || recordId <= 0 {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid recordId")
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	c.Status(200)
	ctxRaw := c.RequestCtx()
	ctxRaw.SetBodyStreamWriter(func(w *bufio.Writer) {
		// 如果记录对应的 Logger 已经结束/清理了，说明任务已经执行完了
		// 直接从文件读取历史日志并返回
		if !service.IsPipelineLoggerActive(uint(recordId)) {
			logs, err := service.ReadPipelineLogFromFile(uint(recordId))
			if err == nil {
				// 对于历史日志，如果行数超过 3000 行，截取最后 2000 行，防止前端瞬间卡死
				if len(logs) > 3000 {
					fmt.Fprintf(w, "data: ... 之前的日志已折叠，总计 %d 行，这里只显示最新 2000 行 ...\n\n", len(logs))
					logs = logs[len(logs)-2000:]
				}
				for _, log := range logs {
					fmt.Fprintf(w, "data: %s\n\n", log)
				}
			}
			fmt.Fprintf(w, "data: EOF\n\n")
			w.Flush()
			return
		}

		// 如果任务还在执行，从内存获取当前的并在通道监听增量日志
		logger := service.GetPipelineLogger(uint(recordId))

		// 先发送已有日志，如果内存里累积了太多，也截断
		logs := logger.GetLogs()
		if len(logs) > 3000 {
			fmt.Fprintf(w, "data: ... 之前的实时日志已折叠，总计 %d 行，这里只显示最新 2000 行 ...\n\n", len(logs))
			logs = logs[len(logs)-2000:]
		}
		for _, log := range logs {
			fmt.Fprintf(w, "data: %s\n\n", log)
		}
		w.Flush()

		// 订阅实时日志
		ch := logger.Subscribe()
		defer logger.Unsubscribe(ch)

		// 由于 fiber/v3 的 StreamWriter 没有直接暴露 client close 的 channel，
		// 我们可以通过定期 ping 或者检测 Write error 来判断客户端是否断开
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case logLine, ok := <-ch:
				if !ok || logLine == "EOF" || logLine == "[\"EOF\"]" {
					fmt.Fprintf(w, "data: EOF\n\n")
					_ = w.Flush()
					return
				}
				if _, err := fmt.Fprintf(w, "data: %s\n\n", logLine); err != nil {
					return
				}
				w.Flush()
			case <-ticker.C:
				// 发送 ping 保持连接
				if _, err := fmt.Fprintf(w, "event: ping\ndata: ping\n\n"); err != nil {
					return
				}
				w.Flush()
			}
		}
	})

	return nil
}
