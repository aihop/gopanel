package client

import (
	"fmt"

	"github.com/aihop/gopanel/global"
)

func loadParamFromVars(key string, vars map[string]interface{}) string {
	if _, ok := vars[key]; !ok {
		if key != "bucket" && key != "port" {
			global.LOG.Errorf("load param %s from vars failed, err: not exist!", key)
		}
		return ""
	}

	return fmt.Sprintf("%v", vars[key])
}
