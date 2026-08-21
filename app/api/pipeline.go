package api

import (
	"bufio"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/i18n"
	"github.com/gofiber/fiber/v3"
)

func requirePipelineManagePermission(c fiber.Ctx) error {
	claims, err := middleware.JwtClaims(c)
	if err != nil {
		return err
	}
	if claims.Role != constant.UserRoleAdmin && claims.Role != constant.UserRoleSuper {
		return buserr.New(constant.ErrPipelinePermissionDenied)
	}
	return nil
}

func PipelinePage(c fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	appSvc := service.NewPipelineApplication(global.DB)
	total, list, err := appSvc.Page(context.Background(), page, limit)
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
		item.RunnerConfig = ""
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
	if err := validatePipelineCodeProjectAccess(c, req.SourceType, req.CodeProjectID); err != nil {
		return c.JSON(e.Fail(err))
	}

	appSvc := service.NewPipelineApplication(global.DB)
	if err := appSvc.Create(*req); err != nil {
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
	if err := validatePipelineCodeProjectAccess(c, req.SourceType, req.CodeProjectID); err != nil {
		return c.JSON(e.Fail(err))
	}

	appSvc := service.NewPipelineApplication(global.DB)
	if err := appSvc.Update(*req); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

func PipelineDetectRunnerPreset(c fiber.Ctx) error {
	if err := requirePipelineManagePermission(c); err != nil {
		return c.JSON(e.Auth(err.Error()))
	}
	req, err := e.BodyToStruct[request.PipelineDetect](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := validatePipelineCodeProjectAccess(c, req.SourceType, req.CodeProjectID); err != nil {
		return c.JSON(e.Fail(err))
	}
	result, err := service.NewPipelineService(global.DB).DetectRunnerPreset(c.Context(), *req)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(result))
}

func validatePipelineCodeProjectAccess(c fiber.Ctx, sourceType string, projectID uint) error {
	if !strings.EqualFold(strings.TrimSpace(sourceType), "code") {
		return nil
	}
	claims, err := middleware.JwtClaims(c)
	if err != nil {
		return err
	}
	var project model.AIProject
	if err := global.DB.First(&project, projectID).Error; err != nil {
		return buserr.New(constant.ErrPipelineCodeProjectNotFound)
	}
	if claims.Role != constant.UserRoleSuper && project.CreatorID != claims.UserId {
		return buserr.New(constant.ErrPipelinePermissionDenied)
	}
	return nil
}

func PipelineDelete(c fiber.Ctx) error {
	if err := requirePipelineManagePermission(c); err != nil {
		return c.JSON(e.Auth(err.Error()))
	}
	id, err := strconv.Atoi(c.Query("id"))
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	appSvc := service.NewPipelineApplication(global.DB)
	if err := appSvc.Delete(uint(id)); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

func PipelineForceDelete(c fiber.Ctx) error {
	if err := requirePipelineManagePermission(c); err != nil {
		return c.JSON(e.Auth(err.Error()))
	}
	req, err := e.BodyToStruct[request.PipelineForceDelete](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	result, err := service.NewPipelineApplication(global.DB).ForceDelete(req.ID, req.ConfirmName)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(result))
}

func PipelineRun(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.PipelineRun](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	appSvc := service.NewPipelineApplication(global.DB)
	recordID, err := appSvc.Run(req.ID, req.Version, req.ExpectedCommit)
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

	service.NewPipelineApplication(global.DB).Stop(req.ID)
	return c.JSON(e.Succ())
}

func PipelineRecordPage(c fiber.Ctx) error {
	pipelineId, _ := strconv.Atoi(c.Query("pipelineId"))
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	appSvc := service.NewPipelineApplication(global.DB)
	total, list, err := appSvc.RecordPage(context.Background(), uint(pipelineId), page, limit)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{
		"total": total,
		"items": list,
	}))
}

func PipelineReleasePage(c fiber.Ctx) error {
	pipelineId, _ := strconv.Atoi(c.Query("pipelineId"))
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	appSvc := service.NewPipelineApplication(global.DB)
	total, list, err := appSvc.ReleasePage(uint(pipelineId), page, limit)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{
		"total": total,
		"items": list,
	}))
}

func PipelineReleaseGet(c fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Query("id"))
	if id <= 0 {
		return c.JSON(e.Fail(buserr.New(constant.ErrPipelineInvalidReleaseID)))
	}

	appSvc := service.NewPipelineApplication(global.DB)
	item, err := appSvc.ReleaseGet(uint(id))
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(item))
}

func PipelineReleasePublish(c fiber.Ctx) error {
	if err := requirePipelineManagePermission(c); err != nil {
		return c.JSON(e.Auth(err.Error()))
	}
	req, err := e.BodyToStruct[request.CommonID](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	appSvc := service.NewPipelineApplication(global.DB)
	item, err := appSvc.PublishRecord(req.ID)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(item))
}

func PipelineRecordDelete(c fiber.Ctx) error {
	if err := requirePipelineManagePermission(c); err != nil {
		return c.JSON(e.Auth(err.Error()))
	}
	recordId, err := strconv.Atoi(c.Query("id"))
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	appSvc := service.NewPipelineApplication(global.DB)
	if err := appSvc.DeleteRecord(uint(recordId)); err != nil {
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
		logger, logs, ch, active := service.SubscribePipelineLogger(uint(recordId))
		if !active {
			logs, err := service.ReadPipelineLogFromFile(uint(recordId))
			if err == nil {
				// 对于历史日志，如果行数超过 3000 行，截取最后 2000 行，防止前端瞬间卡死
				if len(logs) > 3000 {
					_ = writePipelineSSEData(w, i18n.GetMsgWithMap(constant.ErrPipelineLogFoldedNote, map[string]interface{}{"total": len(logs)}))
					logs = logs[len(logs)-2000:]
				}
				for _, log := range logs {
					_ = writePipelineSSEData(w, log)
				}
			}
			_ = writePipelineSSEData(w, "EOF")
			w.Flush()
			return
		}
		defer logger.Unsubscribe(ch)
		if len(logs) > 3000 {
			_ = writePipelineSSEData(w, i18n.GetMsgWithMap(constant.ErrPipelineStreamFoldedNote, map[string]interface{}{"total": len(logs)}))
			logs = logs[len(logs)-2000:]
		}
		for _, log := range logs {
			_ = writePipelineSSEData(w, log)
		}
		w.Flush()

		// 由于 fiber/v3 的 StreamWriter 没有直接暴露 client close 的 channel，
		// 我们可以通过定期 ping 或者检测 Write error 来判断客户端是否断开
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case logLine, ok := <-ch:
				if !ok || logLine == "EOF" || logLine == "[\"EOF\"]" {
					_ = writePipelineSSEData(w, "EOF")
					_ = w.Flush()
					return
				}
				if err := writePipelineSSEData(w, logLine); err != nil {
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

func writePipelineSSEData(w *bufio.Writer, value string) error {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	for _, line := range strings.Split(value, "\n") {
		if _, err := fmt.Fprintf(w, "data: %s\n", line); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(w)
	return err
}
