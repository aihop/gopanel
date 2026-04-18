package e

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"

	"github.com/aihop/gopanel/constant"

	"github.com/aihop/gopanel/app/dto"

	"github.com/aihop/gopanel/pkg/gormx"

	"github.com/go-playground/validator/v10"
)

func Result(err error, dataList ...interface{}) *dto.Result {
	if err == nil {
		return &dto.Result{Code: constant.StatusCodeSuccess, Msg: "success"}
	}
	var data interface{}
	if len(dataList) > 0 {
		data = dataList[0]
	}
	return &dto.Result{Code: constant.StatusCodeFail, Msg: err.Error(), Data: data}
}

func Fail(err error) *dto.Result {
	return &dto.Result{Code: constant.StatusCodeFullFail, Msg: err.Error()}
}

func Error(err error) *dto.Result {
	return &dto.Result{Code: constant.StatusCodeError, Msg: err.Error()}
}

func Auth(data ...interface{}) *dto.Result {
	return RetError(constant.StatusCodeAuthInvalid, data...)
}

func Succ(data ...interface{}) *dto.Result {
	if len(data) > 1 {
		return &dto.Result{Code: constant.StatusCodeSuccess, Msg: data[1].(string), Data: data[0]}
	} else if len(data) == 1 {
		return &dto.Result{Code: constant.StatusCodeSuccess, Msg: "Success", Data: data[0]}
	} else {
		return &dto.Result{Code: constant.StatusCodeSuccess, Msg: "Success"}
	}
}

func RetError(code int, data ...interface{}) *dto.Result {
	if len(data) >= 1 {
		return &dto.Result{Code: code, Msg: data[0].(string)}
	} else {
		return &dto.Result{Msg: "Fail", Code: code}
	}
}

func BodyParser(body []byte, ptr any) error {
	return json.Unmarshal(body, ptr)
}

func BodyToContext(body []byte) (ctx gormx.Contextx, err error) {
	if len(body) == 0 {
		return
	}
	BodyParser(body, &ctx)
	if ctx.Page == 0 {
		ctx.Page = 1
	}
	if ctx.Limit == 0 {
		ctx.Limit = 20
	}
	if ctx.Order == "" {
		ctx.Order = "id DESC"
	}
	return ctx, nil
}

func BodyToWhere(body []byte) (res gormx.Wherex, err error) {
	if len(body) == 0 {
		return
	}
	err = BodyParser(body, &res)
	return
}

func BodyToStrSet(body []byte) (res []string, err error) {
	err = BodyParser(body, &res)
	return
}

func BodyToUintSet(body []byte) (res []uint, err error) {
	err = BodyParser(body, &res)
	return
}

func BodyToStruct[M any](body []byte) (_ *M, err error) {
	var obj M
	ref := reflect.TypeOf(&obj).Elem()
	if err = BodyParser(body, &obj); err != nil {
		return nil, errors.New("parameter failed：" + err.Error())
	}
	if err := validator.New().Struct(&obj); err != nil {
		invalid, ok := err.(*validator.InvalidValidationError)
		if ok {
			return nil, errors.New("parameter failed：" + invalid.Error())
		}
		validationErrs := err.(validator.ValidationErrors) // 断言是ValidationErrors
		var errorMessages []string
		for _, validationErr := range validationErrs {
			fieldName := validationErr.Field()      //获取是哪个字段不符合格式
			field, ok := ref.FieldByName(fieldName) //通过反射获取filed
			if ok {
				errorInfo := field.Tag.Get("error") //info tag值
				if errorInfo != "" {
					errorMessages = append(errorMessages, errorInfo)
					continue
				}
			}
			errorMessages = append(errorMessages, fieldName+" error")
		}
		if len(errorMessages) > 0 {
			return nil, errors.New(strings.Join(errorMessages, "; "))
		}
	}
	return &obj, nil
}

func QueriesToStruct[M any](maps map[string]string) (_ *M, err error) {
	body, _ := json.Marshal(maps)
	return BodyToStruct[M](body)
}

func MapToStruct[M any](maps map[string]any) (_ *M, err error) {
	body, _ := json.Marshal(maps)
	return BodyToStruct[M](body)
}

func BodyToContextNotLimit(body []byte) (ctx gormx.Contextx, err error) {
	if len(body) == 0 {
		return
	}
	BodyParser(body, &ctx)
	if ctx.Page == 0 {
		ctx.Page = 1
	}
	if ctx.Limit == 0 {
		ctx.Limit = 20
	}
	return ctx, nil
}

type H map[string]any
