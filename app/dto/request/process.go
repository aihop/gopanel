package request

type ProcessReq struct {
	PID int32 `json:"PID"  validate:"required"`
}

type ProcessListReq struct {
	PID      int32  `json:"pid"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

type PortReq struct {
	Port uint32 `json:"port"  validate:"required"`
}
