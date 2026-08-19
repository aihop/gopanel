package dto

import (
	"time"

	"github.com/aihop/gopanel/app/model"
)

// MobileNodeRes 是手机端可见的只读节点快照，不包含地址、入口或令牌信息。
type MobileNodeRes struct {
	ID              uint              `json:"id"`
	Name            string            `json:"name"`
	IsLocal         bool              `json:"isLocal"`
	IsProd          bool              `json:"isProd"`
	Status          string            `json:"status"`
	Version         string            `json:"version"`
	LastSeenAt      *time.Time        `json:"lastSeenAt,omitempty"`
	Summary         model.NodeSummary `json:"summary"`
	Warnings        []NodeWarning     `json:"warnings"`
	HasControlToken bool              `json:"hasControlToken"`
}
