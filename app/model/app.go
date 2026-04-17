package model

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/aihop/gopanel/constant"
	"github.com/gofiber/fiber/v3"
)

type App struct {
	BaseModel
	Name                 string  `json:"name" gorm:"not null"`
	Key                  string  `json:"key" gorm:"not null;"`
	ShortDescZh          string  `json:"shortDescZh" yaml:"shortDescZh"`
	ShortDescEn          string  `json:"shortDescEn" yaml:"shortDescEn"`
	Description          string  `json:"description"`
	Icon                 string  `json:"icon"`
	Type                 string  `json:"type" gorm:"not null"`
	Status               string  `json:"status" gorm:"not null"`
	Required             string  `json:"required"`
	CrossVersionUpdate   bool    `json:"crossVersionUpdate" yaml:"crossVersionUpdate"`
	Limit                int     `json:"limit" gorm:"not null"`
	Website              string  `json:"website" gorm:"not null"`
	Github               string  `json:"github" gorm:"not null"`
	Document             string  `json:"document" gorm:"not null"`
	Recommend            int     `json:"recommend" gorm:"not null"`
	Resource             string  `json:"resource" gorm:"not null;default:remote"`
	ReadMe               string  `json:"readMe"`
	LastModified         int     `json:"lastModified"`
	Architectures        string  `json:"architectures"`
	MemoryRequired       int     `json:"memoryRequired"`
	GpuSupport           bool    `json:"gpuSupport"`
	RequiredPanelVersion float64 `json:"requiredPanelVersion"`
	BatchInstallSupport  bool    `json:"batchInstallSupport" yaml:"batchInstallSupport"`

	Details []AppDetail `json:"-" gorm:"-:migration"`
	TagsKey []string    `json:"tags" yaml:"tags" gorm:"-"`
	AppTags []AppTag    `json:"-" gorm:"-:migration"`
}

func (i *App) IsLocalApp() bool {
	return i.Resource == "local"
}
func (i *App) IsCustomApp() bool {
	return i.Resource == "custom"
}

func (i *App) GetAppResourcePath() string {
	if i.IsLocalApp() {
		return filepath.Join(constant.LocalAppResourceDir, strings.TrimPrefix(i.Key, "local"))
	}
	if i.IsCustomApp() {
		return filepath.Join(constant.AppResourceDir, "custom", i.Key)
	}
	return filepath.Join(constant.AppResourceDir, i.Resource, i.Key)
}

func getLang(c fiber.Ctx) string {
	if c == nil {
		return "en"
	}
	lang := c.Get("Accept-Language")
	if lang == "" {
		lang = "en"
	}
	return lang
}

func (i *App) GetDescription(ctx fiber.Ctx) string {
	var translations = make(map[string]string)
	_ = json.Unmarshal([]byte(i.Description), &translations)
	lang := strings.ToLower(getLang(ctx))
	if desc, ok := translations[lang]; ok && desc != "" {
		return desc
	}
	if lang == "zh" && i.ShortDescZh != "" {
		return i.ShortDescZh
	}
	if desc, ok := translations["en"]; ok && desc != "" {
		return desc
	}
	return i.ShortDescEn
}
