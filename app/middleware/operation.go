package middleware

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"

	_ "embed"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

//go:embed x-log.json
var xLogJson []byte

var swagger map[string]operationJson

func init() {
	_ = json.Unmarshal(xLogJson, &swagger)
}

type operationJson struct {
	BodyKeys        []string       `json:"bodyKeys"`
	ParamKeys       []string       `json:"paramKeys"`
	BeforeFunctions []functionInfo `json:"beforeFunctions"`
	FormatZH        string         `json:"formatZH"`
	FormatEN        string         `json:"formatEN"`
}

type functionInfo struct {
	InputColumn  string `json:"input_column"`
	InputValue   string `json:"input_value"`
	IsList       bool   `json:"isList"`
	DB           string `json:"db"`
	OutputColumn string `json:"output_column"`
	OutputValue  string `json:"output_value"`
}

type response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func OperationLog() fiber.Handler {
	return func(c fiber.Ctx) error {
		if isReadOnlyMethod(c.Method()) {
			return c.Next()
		}

		pathItem := c.Path()
		// strip /api prefix to match swagger definitions
		matchPath := strings.TrimPrefix(pathItem, "/api")
		// match swagger mapping
		operationDic, hasPath := swagger[matchPath]

		record := &model.OperationLog{
			Source:    "local",
			IP:        c.IP(),
			Method:    strings.ToLower(c.Method()),
			Path:      pathItem,
			UserAgent: string(c.Request().Header.UserAgent()),
		}

		if hasPath && operationDic.FormatZH != "" {
			formatMap := make(map[string]interface{})
			bodyMap := make(map[string]interface{})
			if len(operationDic.BodyKeys) != 0 {
				_ = json.Unmarshal(c.Body(), &bodyMap)
				for _, key := range operationDic.BodyKeys {
					if val, ok := bodyMap[key]; ok {
						formatMap[key] = val
					}
				}
			}
			fillOperationDetail(&operationDic, formatMap)
			record.DetailEN = strings.ReplaceAll(operationDic.FormatEN, "[]", "")
			record.DetailZH = strings.ReplaceAll(operationDic.FormatZH, "[]", "")
		} else {
			record.DetailEN = fmt.Sprintf("%s %s", c.Method(), pathItem)
			record.DetailZH = fmt.Sprintf("%s %s", c.Method(), pathItem)
		}

		now := time.Now()

		// Execute next handlers
		err := c.Next()
		fillOperationIdentity(c, record)

		var res response
		isJSONResponse := strings.Contains(string(c.Response().Header.Peek("Content-Type")), "application/json")
		if isJSONResponse && !c.Response().IsBodyStream() {
			datas := c.Response().Body()
			if string(c.Response().Header.Peek("Content-Encoding")) == "gzip" {
				reader, gzipErr := gzip.NewReader(bytes.NewReader(datas))
				if gzipErr == nil {
					defer reader.Close()
					datas, _ = io.ReadAll(reader)
				}
			}
			_ = json.Unmarshal(datas, &res)
			if res.Code == 200 || res.Code == 0 {
				record.Status = constant.StatusSuccess
			} else {
				record.Status = constant.StatusFailed
				record.Message = res.Message
			}
		} else {
			statusCode := c.Response().StatusCode()
			if statusCode >= 200 && statusCode < 400 {
				record.Status = constant.StatusSuccess
			} else {
				record.Status = constant.StatusFailed
				record.Message = http.StatusText(statusCode)
			}
		}

		if err != nil {
			record.Status = constant.StatusFailed
			record.Message = err.Error()
		}

		record.Latency = time.Since(now)

		logService := service.NewLogService()
		if err := logService.CreateOperationLog(record); err != nil {
			global.LOG.Errorf("create operation record failed, err: %v", err)
		}

		return err
	}
}

func isReadOnlyMethod(method string) bool {
	return method == fiber.MethodGet || method == fiber.MethodHead || method == fiber.MethodOptions
}

func fillOperationIdentity(c fiber.Ctx, record *model.OperationLog) {
	if claims, ok := c.Locals(constant.AppAuthName).(*token.CustomClaims); ok {
		record.UserID = claims.UserId
	}
	if authMethod, ok := c.Locals(constant.AuthMethodName).(string); ok {
		record.AuthMethod = authMethod
		switch authMethod {
		case constant.AuthMethodMobile, constant.AuthMethodNodeProxy, constant.AuthMethodAPIKey:
			record.Source = authMethod
		}
	}
}

func fillOperationDetail(operationDic *operationJson, formatMap map[string]interface{}) {
	for key, value := range formatMap {
		if !strings.Contains(operationDic.FormatEN, "["+key+"]") {
			continue
		}
		t := reflect.TypeOf(value)
		if t == nil || (t.Kind() != reflect.Array && t.Kind() != reflect.Slice) {
			operationDic.FormatZH = strings.ReplaceAll(operationDic.FormatZH, "["+key+"]", fmt.Sprintf("[%v]", value))
			operationDic.FormatEN = strings.ReplaceAll(operationDic.FormatEN, "["+key+"]", fmt.Sprintf("[%v]", value))
			continue
		}

		val := reflect.ValueOf(value)
		length := val.Len()
		elements := make([]string, 0, length)
		for i := 0; i < length; i++ {
			elements = append(elements, fmt.Sprintf("%v", val.Index(i).Interface()))
		}
		replaced := fmt.Sprintf("[%v]", strings.Join(elements, ","))
		operationDic.FormatZH = strings.ReplaceAll(operationDic.FormatZH, "["+key+"]", replaced)
		operationDic.FormatEN = strings.ReplaceAll(operationDic.FormatEN, "["+key+"]", replaced)
	}
}
