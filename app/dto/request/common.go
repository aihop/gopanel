package request

type ID struct {
	ID uint `json:"id" validate:"required"`
}

type IDOrContainerName struct {
	ID            uint   `json:"id"`
	ContainerName string `json:"containerName"`
}

type IDStr struct {
	ID string `json:"id" validate:"required"`
}

type IdStatus struct {
	ID     uint   `json:"id" validate:"required"`
	Status int    `json:"status"` // 状态 10=停用 20=启用
	Remark string `json:"remark"`
}

type Ids struct {
	Ids []uint `json:"ids" validate:"required"`
}

type ValStr struct {
	Val string `json:"val" validate:"required"`
}

type ArticleId struct {
	ArticleId uint `json:"articleId" validate:"required"`
}

type Name struct {
	Name string `json:"name" validate:"required"`
}

type Slug struct {
	Slug string `json:"slug" validate:"required"`
}

type TypesNotRequired struct {
	Types string `json:"types"`
}

type Path struct {
	Path string `json:"path" validate:"required"`
}

type IDData struct {
	ID   string                 `json:"id" validate:"required"`
	Data map[string]interface{} `json:"data" validate:"required"`
}

type Types struct {
	Types string `json:"types" validate:"required"`
}

type MenuId struct {
	MenuId uint `json:"menuId" validate:"required"`
}

type AppType struct {
	AppType string `json:"appType" validate:"required"`
}

type DictId struct {
	DictId uint `json:"dictId"`
}

type UniqueNames struct {
	UniqueNames []string `json:"uniqueNames"`
}

type UniqueName struct {
	UniqueName string `json:"uniqueName"`
}

type Lang struct {
	Lang string `json:"lang"`
}

type OtherList struct {
	Limit int `json:"limit"`
	Page  int `json:"page"`
	Query int `json:"query"`
}

type License struct {
	License string `json:"license"`
}

type FollowUserId struct {
	FollowUserId uint `json:"followUserId" validate:"required"`
}

type UserId struct {
	UserId uint `json:"userId" validate:"required"`
}

type FilePath struct {
	FilePath string `json:"filePath" validate:"required"`
}

type CommonID struct {
	ID uint `json:"id"`
}
