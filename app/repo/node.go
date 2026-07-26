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

// UpdateSummary 采集成功后只更新采集结果相关字段，避免覆盖用户同时修改的配置字段。
//
// 必须用结构体 + Select 更新，不能用 map[string]interface{}：GORM 的 map 更新分支
// 直接把 map 里的值当绑定参数塞给 driver，不会走字段上的 serializer:json，
// 结果就是 "unsupported type model.NodeSummary, a struct"。结构体分支走 field.ValueOf，
// serializer 才会生效。Select 同时保证 status_msg 的零值（空串）也能写进去。
func (r *NodeRepo) UpdateSummary(id uint, node model.Node) error {
	return global.DB.Model(&model.Node{}).Where("id = ?", id).
		Select("status", "status_msg", "version", "last_seen_at", "summary").
		Updates(node).Error
}

// UpdateStatus 采集失败时只更新状态，保留上一次的 summary 和 last_seen_at
func (r *NodeRepo) UpdateStatus(id uint, status string, statusMsg string) error {
	return global.DB.Model(&model.Node{}).Where("id = ?", id).
		Select("status", "status_msg").
		Updates(model.Node{Status: status, StatusMsg: statusMsg}).Error
}
