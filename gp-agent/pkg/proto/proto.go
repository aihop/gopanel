package proto

type Request struct {
	ID     string                 `json:"id"`
	Action string                 `json:"action"`
	Params map[string]interface{} `json:"params"`
}

type Response struct {
	ID     string `json:"id"`
	OK     bool   `json:"ok"`
	Code   string `json:"code"`
	Output string `json:"output"`
	Error  string `json:"error"`
}

const (
	CodeOK                  = "OK"
	CodeUnsupportedPlatform = "ERR_UNSUPPORTED_PLATFORM"
	CodePermissionDenied    = "ERR_PERMISSION_DENIED"
	CodeNotFound            = "ERR_NOT_FOUND"
	CodeNotInstalled        = "ERR_NOT_INSTALLED"
	CodeInvalidParams       = "ERR_INVALID_PARAMS"
	CodeTimeout             = "ERR_TIMEOUT"
	CodeConflict            = "ERR_CONFLICT"
	CodeInternal            = "ERR_INTERNAL"
)

