package buserr

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/aihop/gopanel/i18n"
	paneli18n "github.com/aihop/gopanel/pkg/i18n"
)

const fallbackLang = "zh"

type BusinessError struct {
	Msg    string
	Detail interface{}
	Map    map[string]interface{}
	Err    error

	// Ctx 用于在 Error() 渲染时读取客户端语言；可为 nil。
	Ctx fiber.Ctx
	// Lang 显式指定语言，优先级高于 Ctx，便于 SSE/WS 等无 ctx 场景。
	Lang string

	skip bool
}

func (e BusinessError) Error() string {
	if e.skip {
		return e.Msg
	}
	lang := e.resolveLang()

	content := ""
	switch {
	case e.Detail != nil:
		content = i18n.GetMsgByLang(lang, e.Msg, map[string]interface{}{"detail": e.Detail})
	case e.Map != nil:
		content = i18n.GetMsgByLang(lang, e.Msg, e.Map)
	default:
		content = i18n.GetMsgByLang(lang, e.Msg)
	}
	if content == "" {
		if e.Err != nil {
			return e.Err.Error()
		}
		return errors.New(e.Msg).Error()
	}
	return content
}

// resolveLang 解析最终渲染语言：Lang > Ctx.locals > 默认 zh。
func (e BusinessError) resolveLang() string {
	if e.Lang != "" {
		return e.Lang
	}
	if e.Ctx != nil {
		if lang := paneli18n.GetLang(e.Ctx); lang != "" {
			return lang
		}
	}
	return fallbackLang
}

func New(key string) BusinessError {
	return BusinessError{
		Msg:    key,
		Detail: nil,
		Err:    nil,
	}
}

func Err(err error) BusinessError {
	key := err.Error()
	var skip bool
	if !strings.HasPrefix(key, "Err") {
		skip = true
	}
	return BusinessError{
		Msg:    key,
		Detail: "",
		Err:    err,
		skip:   skip,
	}
}

func WithDetail(key string, detail interface{}, errs ...error) BusinessError {
	var err error
	if len(errs) >= 1 {
		err = errs[0]
	}
	return BusinessError{
		Msg:    key,
		Detail: detail,
		Err:    err,
	}
}

func WithErr(key string, err error) BusinessError {
	return BusinessError{
		Msg:    key,
		Detail: "",
		Err:    err,
	}
}

func WithMap(key string, maps map[string]interface{}, errs ...error) BusinessError {
	var err error
	if len(errs) >= 1 {
		err = errs[0]
	}
	return BusinessError{
		Msg: key,
		Map: maps,
		Err: err,
	}
}

func WithNameAndErr(key string, name string, err error) BusinessError {
	paramMap := map[string]interface{}{}
	if name != "" {
		paramMap["name"] = name
	}
	if err != nil {
		paramMap["err"] = err.Error()
	}
	return BusinessError{
		Msg: key,
		Map: paramMap,
		Err: err,
	}
}

func WithName(key string, name string) BusinessError {
	paramMap := map[string]interface{}{}
	if name != "" {
		paramMap["name"] = name
	}
	return BusinessError{
		Msg: key,
		Map: paramMap,
	}
}

func WithNameNoCtx(key string, name string) BusinessError {
	paramMap := map[string]interface{}{}
	if name != "" {
		paramMap["name"] = name
	}
	return BusinessError{
		Msg: key,
		Map: paramMap,
	}
}

// NewWithCtx 创建携带 ctx 的 BusinessError，渲染时会按客户端语言查找。
func NewWithCtx(c fiber.Ctx, key string) BusinessError {
	return BusinessError{Msg: key, Ctx: c}
}

// WithDetailWithCtx 创建携带 ctx 的 BusinessError，并附带 detail。
func WithDetailWithCtx(c fiber.Ctx, key string, detail interface{}, errs ...error) BusinessError {
	var err error
	if len(errs) >= 1 {
		err = errs[0]
	}
	return BusinessError{Msg: key, Detail: detail, Err: err, Ctx: c}
}

// WithMapWithCtx 创建携带 ctx 的 BusinessError，并附带模板变量。
func WithMapWithCtx(c fiber.Ctx, key string, maps map[string]interface{}, errs ...error) BusinessError {
	var err error
	if len(errs) >= 1 {
		err = errs[0]
	}
	return BusinessError{Msg: key, Map: maps, Err: err, Ctx: c}
}

// WithNameWithCtx 创建携带 ctx 的 BusinessError，并附带 name 变量。
func WithNameWithCtx(c fiber.Ctx, key string, name string) BusinessError {
	paramMap := map[string]interface{}{}
	if name != "" {
		paramMap["name"] = name
	}
	return BusinessError{Msg: key, Map: paramMap, Ctx: c}
}

// NewWithLang 显式指定语言，常用于 SSE/WS 等无 ctx 场景。
func NewWithLang(lang string, key string) BusinessError {
	return BusinessError{Msg: key, Lang: lang}
}

// WithDetailWithLang 显式指定语言，并附带 detail。
func WithDetailWithLang(lang string, key string, detail interface{}, errs ...error) BusinessError {
	var err error
	if len(errs) >= 1 {
		err = errs[0]
	}
	return BusinessError{Msg: key, Detail: detail, Err: err, Lang: lang}
}

// WithMapWithLang 显式指定语言，并附带模板变量。
func WithMapWithLang(lang string, key string, maps map[string]interface{}, errs ...error) BusinessError {
	var err error
	if len(errs) >= 1 {
		err = errs[0]
	}
	return BusinessError{Msg: key, Map: maps, Err: err, Lang: lang}
}

// WithNameWithLang 显式指定语言，并附带 name 变量。
func WithNameWithLang(lang string, key string, name string) BusinessError {
	paramMap := map[string]interface{}{}
	if name != "" {
		paramMap["name"] = name
	}
	return BusinessError{Msg: key, Map: paramMap, Lang: lang}
}
