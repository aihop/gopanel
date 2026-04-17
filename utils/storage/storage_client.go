package storage

import (
	"github.com/aihop/gopanel/utils/storage/client"
)

type Client = client.StorageClient

func NewClient(backupType string, vars map[string]interface{}) (Client, error) {
	return client.NewClient(backupType, vars)
}
