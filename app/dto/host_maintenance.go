package dto

type HostMemoryClearReq struct {
	Mode int `json:"mode"`
}

type HostMemoryClearRes struct {
	Stdout        string `json:"stdout"`
	NeedPrivilege bool   `json:"needPrivilege"`
	Message       string `json:"message"`
}

type HostCPURelieveReq struct {
	Level int `json:"level"`
}

type HostCPURelieveRes struct {
	Level   int    `json:"level"`
	Message string `json:"message"`
}
