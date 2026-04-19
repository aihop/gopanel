package api

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/aihop/gopanel/gp-agent/app/service"
	"github.com/aihop/gopanel/gp-agent/global"
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
		if !global.CONF.EnableCaddy {
			return errResp(req.ID, proto.CodeNotInstalled, errors.New("caddy disabled"))
		}
		out, err := service.CaddyStatusJSON()
		if err != nil {
			return errResp(req.ID, proto.CodeInternal, err)
		}
		return proto.Response{ID: req.ID, OK: true, Code: proto.CodeOK, Output: out}
	case "CADDY_GET_CONFIG":
		if !global.CONF.EnableCaddy {
			return errResp(req.ID, proto.CodeNotInstalled, errors.New("caddy disabled"))
		}
		out, err := service.CaddyGetConfigJSON()
		if err != nil {
			return errResp(req.ID, proto.CodeInternal, err)
		}
		return proto.Response{ID: req.ID, OK: true, Code: proto.CodeOK, Output: out}
	case "CADDY_APPLY":
		if !global.CONF.EnableCaddy {
			return errResp(req.ID, proto.CodeNotInstalled, errors.New("caddy disabled"))
		}
		out, err := service.CaddyApply(ctx, req.Params)
		if err != nil {
			return errResp(req.ID, proto.CodeInternal, err)
		}
		return proto.Response{ID: req.ID, OK: true, Code: proto.CodeOK, Output: out}
	case "CADDY_STOP":
		if !global.CONF.EnableCaddy {
			return errResp(req.ID, proto.CodeNotInstalled, errors.New("caddy disabled"))
		}
		out, err := service.CaddyStop(ctx)
		if err != nil {
			return errResp(req.ID, proto.CodeInternal, err)
		}
		return proto.Response{ID: req.ID, OK: true, Code: proto.CodeOK, Output: out}
	case "DAEMON_STATUS":
		if !global.CONF.EnableDaemon {
			return errResp(req.ID, proto.CodeNotInstalled, errors.New("daemon disabled"))
		}
		out, err := service.DaemonStatusJSON()
		if err != nil {
			return errResp(req.ID, proto.CodeInternal, err)
		}
		return proto.Response{ID: req.ID, OK: true, Code: proto.CodeOK, Output: out}
	case "DAEMON_START":
		if !global.CONF.EnableDaemon {
			return errResp(req.ID, proto.CodeNotInstalled, errors.New("daemon disabled"))
		}
		out, err := service.DaemonStart(ctx)
		if err != nil {
			return errResp(req.ID, proto.CodeInternal, err)
		}
		return proto.Response{ID: req.ID, OK: true, Code: proto.CodeOK, Output: out}
	case "DAEMON_STOP":
		if !global.CONF.EnableDaemon {
			return errResp(req.ID, proto.CodeNotInstalled, errors.New("daemon disabled"))
		}
		out, err := service.DaemonStop(ctx)
		if err != nil {
			return errResp(req.ID, proto.CodeInternal, err)
		}
		return proto.Response{ID: req.ID, OK: true, Code: proto.CodeOK, Output: out}
	case "DAEMON_RELOAD":
		if !global.CONF.EnableDaemon {
			return errResp(req.ID, proto.CodeNotInstalled, errors.New("daemon disabled"))
		}
		out, err := service.DaemonReload(ctx)
		if err != nil {
			return errResp(req.ID, proto.CodeInternal, err)
		}
		return proto.Response{ID: req.ID, OK: true, Code: proto.CodeOK, Output: out}
	case "DAEMON_APP_LIST":
		if !global.CONF.EnableDaemon {
			return errResp(req.ID, proto.CodeNotInstalled, errors.New("daemon disabled"))
		}
		out, err := service.DaemonAppListJSON()
		if err != nil {
			return errResp(req.ID, proto.CodeInternal, err)
		}
		return proto.Response{ID: req.ID, OK: true, Code: proto.CodeOK, Output: out}
	case "DAEMON_APP_START":
		if !global.CONF.EnableDaemon {
			return errResp(req.ID, proto.CodeNotInstalled, errors.New("daemon disabled"))
		}
		out, err := service.DaemonAppStart(ctx, req.Params)
		if err != nil {
			return errResp(req.ID, proto.CodeInternal, err)
		}
		return proto.Response{ID: req.ID, OK: true, Code: proto.CodeOK, Output: out}
	case "DAEMON_APP_STOP":
		if !global.CONF.EnableDaemon {
			return errResp(req.ID, proto.CodeNotInstalled, errors.New("daemon disabled"))
		}
		out, err := service.DaemonAppStop(ctx, req.Params)
		if err != nil {
			return errResp(req.ID, proto.CodeInternal, err)
		}
		return proto.Response{ID: req.ID, OK: true, Code: proto.CodeOK, Output: out}
	case "DAEMON_APP_RESTART":
		if !global.CONF.EnableDaemon {
			return errResp(req.ID, proto.CodeNotInstalled, errors.New("daemon disabled"))
		}
		out, err := service.DaemonAppRestart(ctx, req.Params)
		if err != nil {
			return errResp(req.ID, proto.CodeInternal, err)
		}
		return proto.Response{ID: req.ID, OK: true, Code: proto.CodeOK, Output: out}
	case "DAEMON_APP_LOG":
		if !global.CONF.EnableDaemon {
			return errResp(req.ID, proto.CodeNotInstalled, errors.New("daemon disabled"))
		}
		out, err := service.DaemonAppLogJSON(ctx, req.Params)
		if err != nil {
			return errResp(req.ID, proto.CodeInternal, err)
		}
		return proto.Response{ID: req.ID, OK: true, Code: proto.CodeOK, Output: out}
	case "DAEMON_APP_LOG_CLEAR":
		if !global.CONF.EnableDaemon {
			return errResp(req.ID, proto.CodeNotInstalled, errors.New("daemon disabled"))
		}
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
