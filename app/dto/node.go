package dto

import (
	"time"

	"github.com/aihop/gopanel/app/model"
)

// NodeCreateReq 主控侧新增节点
type NodeCreateReq struct {
	Name        string `json:"name" validate:"required"`
	Addr        string `json:"addr" validate:"required"`
	AccessToken string `json:"accessToken" validate:"required"`
	Entrance    string `json:"entrance"`
	SkipVerify  bool   `json:"skipVerify"`
	IsProd      bool   `json:"isProd"`
	Sort        int    `json:"sort"`
}

// NodeUpdateReq 更新节点。AccessToken 留空表示不修改。
type NodeUpdateReq struct {
	ID          uint   `json:"id" validate:"required"`
	Name        string `json:"name" validate:"required"`
	Addr        string `json:"addr" validate:"required"`
	AccessToken string `json:"accessToken"`
	Entrance    string `json:"entrance"`
	SkipVerify  bool   `json:"skipVerify"`
	IsProd      bool   `json:"isProd"`
	Sort        int    `json:"sort"`
}

// NodeWarning 节点告警项。只给出类型与数值，展示文案由前端按 type 做 i18n。
type NodeWarning struct {
	Type  string  `json:"type"`  // offline / disk / cert / container / unauthorized
	Level string  `json:"level"` // warn / danger
	Value float64 `json:"value"`
}

// NodeRes 节点列表项，供细条与抽屉渲染
type NodeRes struct {
	ID          uint              `json:"id"`
	Name        string            `json:"name"`
	Addr        string            `json:"addr"`
	Entrance    string            `json:"entrance"`
	ConnectMode string            `json:"connectMode"`
	SkipVerify  bool              `json:"skipVerify"`
	IsProd      bool              `json:"isProd"`
	Sort        int               `json:"sort"`
	Status      string            `json:"status"`
	StatusMsg   string            `json:"statusMsg"`
	Version     string            `json:"version"`
	LastSeenAt  time.Time         `json:"lastSeenAt"`
	Summary     model.NodeSummary `json:"summary"`
	Warnings    []NodeWarning     `json:"warnings"`
	HasToken    bool              `json:"hasToken"`
}

// NodeTokenRes 被控侧签发只读令牌的返回
type NodeTokenRes struct {
	AccessToken string `json:"accessToken"`
	Addr        string `json:"addr"`
	Hostname    string `json:"hostname"`
	Version     string `json:"version"`
}
