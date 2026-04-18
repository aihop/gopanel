package request

import "github.com/aihop/gopanel/pkg/gormx"

type CloudAccountCreate struct {
	Name          string      `json:"name" validate:"required"`
	Type          string      `json:"type" validate:"required"`
	Authorization interface{} `json:"authorization" validate:"required"`
}

type CloudAccountUpdate struct {
	ID            uint        `json:"id" validate:"required"`
	Name          string      `json:"name"`
	Type          string      `json:"type"`
	Authorization interface{} `json:"authorization"`
}

type CloudAccountSearch struct {
	gormx.Contextx
}
