//go:build storage_full

package client

import "github.com/aihop/gopanel/constant"

func init() {
	RegisterFactory(constant.OSS, func(vars map[string]interface{}) (StorageClient, error) {
		return NewOssClient(vars)
	})
	RegisterFactory(constant.Sftp, func(vars map[string]interface{}) (StorageClient, error) {
		return NewSftpClient(vars)
	})
	RegisterFactory(constant.WebDAV, func(vars map[string]interface{}) (StorageClient, error) {
		return NewWebDAVClient(vars)
	})
	RegisterFactory(constant.MinIo, func(vars map[string]interface{}) (StorageClient, error) {
		return NewMinIoClient(vars)
	})
	RegisterFactory(constant.Cos, func(vars map[string]interface{}) (StorageClient, error) {
		return NewCosClient(vars)
	})
	RegisterFactory(constant.Kodo, func(vars map[string]interface{}) (StorageClient, error) {
		return NewKodoClient(vars)
	})
	RegisterFactory(constant.OneDrive, func(vars map[string]interface{}) (StorageClient, error) {
		return NewOneDriveClient(vars)
	})
}
