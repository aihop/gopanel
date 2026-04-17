package i18n

import (
	"errors"
	"strings"

	"github.com/aihop/gopanel/pkg/i18n"
	goI18n "github.com/nicksnyder/go-i18n/v2/i18n"
)

func GetMsg(key string, details ...map[string]interface{}) string {
	if key == "" {
		return ""
	}

	var content string
	if len(details) == 0 {
		localize, err := i18n.Localize(key, "zh")
		if err != nil {
			return key
		}
		return localize
	} else {
		newMap := make(map[string]interface{})
		for _, val := range details {
			for k, v := range val {
				newMap[k] = v
			}
		}
		content = i18n.MustLocalize(&goI18n.LocalizeConfig{
			MessageID:    key,
			TemplateData: newMap,
		}, "zh")
	}

	if content == "" {
		return key
	}
	return content
}

func GetErrMsg(key string, maps ...map[string]interface{}) error {
	return errors.New(GetMsg(key, maps...))
}

func GetMsgByKey(key string) string {
	if key == "" {
		return ""
	}
	localize, err := i18n.Localize(key, "zh")
	if err != nil {
		return key
	}
	return localize
}

func GetMsgWithMap(key string, maps map[string]interface{}) string {
	var content string
	if maps == nil {
		content, _ = i18n.Localize(&goI18n.LocalizeConfig{
			MessageID: key,
		}, "zh")
	} else {
		content, _ = i18n.Localize(&goI18n.LocalizeConfig{
			MessageID:    key,
			TemplateData: maps,
		}, "zh")
	}
	content = strings.ReplaceAll(content, ": <no value>", "")
	if content == "" {
		return key
	} else {
		return content
	}
}
