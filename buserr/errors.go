package buserr

import (
	"errors"
	"strings"

	"github.com/aihop/gopanel/i18n"
)

type BusinessError struct {
	Msg    string
	Detail interface{}
	Map    map[string]interface{}
	Err    error

	skip bool
}

func (e BusinessError) Error() string {
	if e.skip {
		return e.Msg
	}
	content := ""
	if e.Detail != nil {
		content = i18n.GetMsg(e.Msg, map[string]interface{}{"detail": e.Detail})
	} else if e.Map != nil {
		content = i18n.GetMsg(e.Msg, e.Map)
	} else {
		content = i18n.GetMsg(e.Msg)
	}
	if content == "" {
		if e.Err != nil {
			return e.Err.Error()
		}
		return errors.New(e.Msg).Error()
	}
	return content
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
