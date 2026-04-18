package request

import (
	"github.com/aihop/gopanel/app/dto"
)

type RuntimeSearch struct {
	dto.PageInfo
	Type   string `json:"type"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type RuntimeCreate struct {
	AppDetailID uint                   `json:"appDetailId"`
	Name        string                 `json:"name"`
	Params      map[string]interface{} `json:"params"`
	Resource    string                 `json:"resource"`
	Image       string                 `json:"image"`
	Type        string                 `json:"type"`
	Version     string                 `json:"version"`
	Source      string                 `json:"source"`
	CodeDir     string                 `json:"codeDir"`
	NodeConfig
}

type RuntimePHPExtensionsSearch struct {
	dto.PageInfo
	All bool `json:"all"`
}

type RuntimePHPExtensionsCreate struct {
	Name       string `json:"name" validate:"required"`
	Extensions string `json:"extensions" validate:"required"`
}

type RuntimePHPExtensionsUpdate struct {
	ID         uint   `json:"id" validate:"required"`
	Extensions string `json:"extensions" validate:"required"`
}

type RuntimePHPExtensionsDelete struct {
	ID uint `json:"id" validate:"required"`
}

type NodeConfig struct {
	Install      bool          `json:"install"`
	Clean        bool          `json:"clean"`
	Port         int           `json:"port"`
	ExposedPorts []ExposedPort `json:"exposedPorts"`
}
type ExposedPort struct {
	HostPort      int `json:"hostPort"`
	ContainerPort int `json:"containerPort"`
}
