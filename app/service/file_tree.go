package service

import (
	"errors"
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/dto/response"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/common"
	"github.com/aihop/gopanel/utils/files"
	"os"
	"path/filepath"
	"strings"
)

func (f *FileService) GetFileTree(op request.FileOption) ([]response.FileTree, error) {
	var treeArray []response.FileTree
	if _, err := os.Stat(op.Path); err != nil && os.IsNotExist(err) {
		return treeArray, nil
	}
	info, err := files.NewFileInfo(op.FileOption)
	if err != nil {
		return nil, err
	}
	node := response.FileTree{ID: common.GetUuid(), Name: info.Name, Path: info.Path, IsDir: info.IsDir, Extension: info.Extension}
	err = f.buildFileTree(&node, info.Items, op, 2)
	if err != nil {
		return nil, err
	}
	return append(treeArray, node), nil
}
func shouldFilterPath(path string) bool {
	cleanedPath := filepath.Clean(path)
	for _, filteredPath := range filteredPaths {
		cleanedFilteredPath := filepath.Clean(filteredPath)
		if cleanedFilteredPath == cleanedPath || strings.HasPrefix(cleanedPath, cleanedFilteredPath+"/") {
			return true
		}
	}
	return false
}
func (f *FileService) buildFileTree(node *response.FileTree, items []*files.FileInfo, op request.FileOption, level int) error {
	for _, v := range items {
		if shouldFilterPath(v.Path) {
			global.LOG.Infof("File Tree: Skipping %s due to filter\n", v.Path)
			continue
		}
		childNode := response.FileTree{ID: common.GetUuid(), Name: v.Name, Path: v.Path, IsDir: v.IsDir, Extension: v.Extension}
		if level > 1 && v.IsDir {
			if err := f.buildChildNode(&childNode, v, op, level); err != nil {
				return err
			}
		}
		node.Children = append(node.Children, childNode)
	}
	return nil
}
func (f *FileService) buildChildNode(childNode *response.FileTree, fileInfo *files.FileInfo, op request.FileOption, level int) error {
	op.Path = fileInfo.Path
	subInfo, err := files.NewFileInfo(op.FileOption)
	if err != nil {
		if os.IsPermission(err) || errors.Is(err, os.ErrPermission) {
			global.LOG.Infof("File Tree: Skipping %s due to permission denied\n", fileInfo.Path)
			return nil
		}
		global.LOG.Errorf("File Tree: Skipping %s due to error: %s\n", fileInfo.Path, err.Error())
		return nil
	}
	return f.buildFileTree(childNode, subInfo.Items, op, level-1)
}
