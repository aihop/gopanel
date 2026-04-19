package api

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/aihop/gopanel/gp-agent/app/service"
	"github.com/aihop/gopanel/gp-agent/pkg/proto"
)

func Handle(ctx context.Context, req proto.Request) proto.Response {
	switch strings.ToUpper(req.Action) {
	case "PING":
		return proto.Response{ID: req.ID, OK: true, Code: proto.CodeOK, Output: "PONG"}
	case "AGENT_STATUS":
		out, err := service.GetAgentStatusJSON()
		if err != nil {
			return errResp(req.ID, proto.CodeInternal, err)
		}
		return proto.Response{ID: req.ID, OK: true, Code: proto.CodeOK, Output: out}
	case "STATUS":
		cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		st, err := service.GetLocalStatus(cctx)
		if err != nil {
			return errResp(req.ID, proto.CodeInternal, err)
		}
		b, err := json.Marshal(st)
		if err != nil {
			return errResp(req.ID, proto.CodeInternal, err)
		}
		return proto.Response{ID: req.ID, OK: true, Code: proto.CodeOK, Output: string(b)}
	case "CADDY_STATUS":
		out, err := service.CaddyStatusJSON()
		if err != nil {
			return errResp(req.ID, proto.CodeInternal, err)
		}
		return proto.Response{ID: req.ID, OK: true, Code: proto.CodeOK, Output: out}
	case "CADDY_CONFIG":
		out, err := service.CaddyGetConfigJSON()
		if err != nil {
			return errResp(req.ID, proto.CodeInternal, err)
		}
		return proto.Response{ID: req.ID, OK: true, Code: proto.CodeOK, Output: out}
	case "CADDY_FILE_JSON":
		out, err := service.CaddyFileToJson([]byte(req.Params["content"].(string)))
		if err != nil {
			return errResp(req.ID, proto.CodeInternal, err)
		}
		return proto.Response{ID: req.ID, OK: true, Code: proto.CodeOK, Output: string(out)}
	case "CADDY_APPLY": // 应用CaddyFile
		out, err := service.CaddyApply(ctx, req.Params)
		if err != nil {
			return errResp(req.ID, proto.CodeInternal, err)
		}
		return proto.Response{ID: req.ID, OK: true, Code: proto.CodeOK, Output: out}
	case "CADDY_STOP":
		out, err := service.CaddyStop(ctx)
		if err != nil {
			return errResp(req.ID, proto.CodeInternal, err)
		}
		return proto.Response{ID: req.ID, OK: true, Code: proto.CodeOK, Output: out}
	case "DAEMON_STATUS":
		out, err := service.DaemonStatusJSON()
		if err != nil {
			return errResp(req.ID, proto.CodeInternal, err)
		}
		return proto.Response{ID: req.ID, OK: true, Code: proto.CodeOK, Output: out}
	case "DAEMON_START":
		out, err := service.DaemonStart(ctx)
		if err != nil {
			return errResp(req.ID, proto.CodeInternal, err)
		}
		return proto.Response{ID: req.ID, OK: true, Code: proto.CodeOK, Output: out}
	case "DAEMON_STOP":
		out, err := service.DaemonStop(ctx)
		if err != nil {
			return errResp(req.ID, proto.CodeInternal, err)
		}
		return proto.Response{ID: req.ID, OK: true, Code: proto.CodeOK, Output: out}
	case "DAEMON_RELOAD":
		out, err := service.DaemonReload(ctx)
		if err != nil {
			return errResp(req.ID, proto.CodeInternal, err)
		}
		return proto.Response{ID: req.ID, OK: true, Code: proto.CodeOK, Output: out}
	case "DAEMON_APP_LIST":
		out, err := service.DaemonAppListJSON()
		if err != nil {
			return errResp(req.ID, proto.CodeInternal, err)
		}
		return proto.Response{ID: req.ID, OK: true, Code: proto.CodeOK, Output: out}
	case "DAEMON_APP_START":
		out, err := service.DaemonAppStart(ctx, req.Params)
		if err != nil {
			return errResp(req.ID, proto.CodeInternal, err)
		}
		return proto.Response{ID: req.ID, OK: true, Code: proto.CodeOK, Output: out}
	case "DAEMON_APP_STOP":
		out, err := service.DaemonAppStop(ctx, req.Params)
		if err != nil {
			return errResp(req.ID, proto.CodeInternal, err)
		}
		return proto.Response{ID: req.ID, OK: true, Code: proto.CodeOK, Output: out}
	case "DAEMON_APP_RESTART":
		out, err := service.DaemonAppRestart(ctx, req.Params)
		if err != nil {
			return errResp(req.ID, proto.CodeInternal, err)
		}
		return proto.Response{ID: req.ID, OK: true, Code: proto.CodeOK, Output: out}
	case "DAEMON_APP_LOG":
		out, err := service.DaemonAppLogJSON(ctx, req.Params)
		if err != nil {
			return errResp(req.ID, proto.CodeInternal, err)
		}
		return proto.Response{ID: req.ID, OK: true, Code: proto.CodeOK, Output: out}
	case "DAEMON_APP_LOG_CLEAR":
		out, err := service.DaemonAppLogClear(ctx, req.Params)
		if err != nil {
			return errResp(req.ID, proto.CodeInternal, err)
		}
		return proto.Response{ID: req.ID, OK: true, Code: proto.CodeOK, Output: out}
	default:
		return errResp(req.ID, proto.CodeNotFound, errors.New("unknown action"))
	}
}

func errResp(id, code string, err error) proto.Response {
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	return proto.Response{ID: id, OK: false, Code: code, Error: msg}
}
