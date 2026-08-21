package i18n

import (
	"errors"
	"strings"

	"github.com/aihop/gopanel/pkg/i18n"
	"github.com/gofiber/fiber/v3"
	goI18n "github.com/nicksnyder/go-i18n/v2/i18n"
)

// fallbackLang 是默认回退语言，按计划保留中文 fallback 语义。
const fallbackLang = "zh"

func GetMsg(key string, details ...map[string]interface{}) string {
	return GetMsgByLang(fallbackLang, key, details...)
}

func GetErrMsg(key string, maps ...map[string]interface{}) error {
	return errors.New(GetMsgByLang(fallbackLang, key, maps...))
}

func GetMsgByKey(key string) string {
	return GetMsgByLang(fallbackLang, key)
}

func GetMsgWithMap(key string, maps map[string]interface{}) string {
	return GetMsgByLang(fallbackLang, key, maps)
}

// GetMsgByLang 按指定语言查询 i18n key；缺 key 时由 pkg/i18n 的 localizer 内部回退到默认语言。
func GetMsgByLang(lang string, key string, details ...map[string]interface{}) string {
	if key == "" {
		return ""
	}

	if len(details) == 0 {
		localize, err := i18n.Localize(key, lang)
		if err != nil {
			return key
		}
		return localize
	}

	newMap := make(map[string]interface{})
	for _, val := range details {
		for k, v := range val {
			newMap[k] = v
		}
	}
	content := i18n.MustLocalize(&goI18n.LocalizeConfig{
		MessageID:    key,
		TemplateData: newMap,
	}, lang)

	content = strings.ReplaceAll(content, ": <no value>", "")
	if content == "" {
		return key
	}
	return content
}

// GetMsgFromCtx 从 fiber.Ctx 中读取客户端语言并查找 key；ctx 为 nil 时退化为 GetMsg。
func GetMsgFromCtx(c fiber.Ctx, key string, details ...map[string]interface{}) string {
	lang := i18n.GetLang(c)
	if lang == "" {
		lang = fallbackLang
	}
	return GetMsgByLang(lang, key, details...)
}
