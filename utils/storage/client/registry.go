package client

import "github.com/aihop/gopanel/constant"

type StorageClient interface {
	ListBuckets() ([]interface{}, error)
	ListObjects(prefix string) ([]string, error)
	Exist(path string) (bool, error)
	Delete(path string) (bool, error)
	Upload(src, target string) (bool, error)
	Download(src, target string) (bool, error)
	Size(path string) (int64, error)
}

type Factory func(vars map[string]interface{}) (StorageClient, error)

var factories = map[string]Factory{}

func RegisterFactory(name string, factory Factory) {
	if factory == nil {
		return
	}
	factories[name] = factory
}

func NewClient(backupType string, vars map[string]interface{}) (StorageClient, error) {
	factory, ok := factories[backupType]
	if !ok {
		return nil, constant.ErrNotSupportType
	}
	return factory(vars)
}
