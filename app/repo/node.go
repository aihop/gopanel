package repo

import (
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

type NodeRepo struct{}

func NewNode() *NodeRepo {
	return &NodeRepo{}
}

func (r *NodeRepo) MigrateTable() error {
	return global.DB.AutoMigrate(&model.Node{})
}

func (r *NodeRepo) List() ([]model.Node, error) {
	var list []model.Node
	err := global.DB.Model(&model.Node{}).Order("sort asc").Order("id asc").Find(&list).Error
	return list, err
}

func (r *NodeRepo) GetByID(id uint) (model.Node, error) {
	var node model.Node
	err := global.DB.Where("id = ?", id).First(&node).Error
	return node, err
}

func (r *NodeRepo) CountByAddr(addr string, excludeID uint) (int64, error) {
	var count int64
	tx := global.DB.Model(&model.Node{}).Where("addr = ?", addr)
	if excludeID > 0 {
		tx = tx.Where("id <> ?", excludeID)
	}
	err := tx.Count(&count).Error
	return count, err
}

func (r *NodeRepo) Create(node *model.Node) error {
	return global.DB.Create(node).Error
}

func (r *NodeRepo) Save(node *model.Node) error {
	return global.DB.Save(node).Error
}

func (r *NodeRepo) DeleteByID(id uint) error {
	return global.DB.Delete(&model.Node{}, id).Error
}

// UpdateSummary 只更新采集结果相关字段，避免覆盖用户同时修改的配置字段
func (r *NodeRepo) UpdateSummary(id uint, fields map[string]interface{}) error {
	return global.DB.Model(&model.Node{}).Where("id = ?", id).Updates(fields).Error
}
