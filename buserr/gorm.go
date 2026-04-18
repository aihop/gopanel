package buserr

import (
	"regexp"

	"github.com/aihop/gopanel/i18n"
)

func GormErrorDuplicate(msg string) string {

	re, _ := regexp.Compile(`'([^']*)'`)
	matches := re.FindAllStringSubmatch(msg, -1)
	var info, key string
	if len(matches) == 2 {
		for i, match := range matches {
			if i == 0 {
				info = match[1]
			} else if i == 1 {
				key = match[1]
			}
		}
	}
	if info != "" && key != "" {
		msg = i18n.GetMsg(key, map[string]interface{}{"info": info})
	}

	return msg
}
