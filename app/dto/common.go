package dto

type Result struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

type StatusCount struct {
	Status int `json:"status"`
	Count  int `json:"count"`
}

type IdTotal struct {
	Id    uint `json:"id"`
	Total int  `json:"total"`
}

type IdParamsData struct {
	ID     string      `json:"id" validate:"required"` // ID
	Params []string    `json:"params"`
	Data   interface{} `json:"data"`
}

// 定位
type Location struct {
	Lng     string `json:"lng"`     //经度
	Lat     string `json:"lat"`     // 纬度
	Address string `json:"address"` // 显示地址
}

// 地址
type Position struct {
	Province    string `json:"province"`    // 省ID
	City        string `json:"city"`        // 城市ID
	District    string `json:"district"`    // 区域ID
	Street      string `json:"street"`      // 街道ID
	DistAddress string `json:"distAddress"` // 显示省市区的地址
	Remark      string `json:"remark"`      // 备注地址
	Address     string `json:"address"`     // 详细的地址
}

type WebhookEventParams struct {
	Types   string                 `json:"types"`
	Task    string                 `json:"task"`
	Url     string                 `json:"url"`
	Headers map[string]string      `json:"headers"`
	Params  map[string]interface{} `json:"params"`
	Name    string
}

type CaptchaResponse struct {
	CaptchaID string `json:"captchaId"`
	ImagePath string `json:"imagePath"`
}
