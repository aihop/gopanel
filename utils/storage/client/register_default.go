package client

import "github.com/aihop/gopanel/constant"

func init() {
	RegisterFactory(constant.Local, func(vars map[string]interface{}) (StorageClient, error) {
		return NewLocalClient(vars)
	})
}
