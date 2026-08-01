package api

import "time"

const (
	hostTerminalHistoryLimit = 1024 * 1024
	hostTerminalControlLease = 60 * time.Second
)

type hostTerminalEvent struct {
	Type           string `json:"type"`
	Sequence       uint64 `json:"sequence,omitempty"`
	Data           string `json:"data,omitempty"`
	HasControl     bool   `json:"hasControl,omitempty"`
	Truncated      bool   `json:"truncated,omitempty"`
	LeaseExpiresAt int64  `json:"leaseExpiresAt,omitempty"`
}

type hostTerminalSubscription struct {
	ID     string
	Events chan hostTerminalEvent
	UserID uint
	IP     string
}

type createHostTerminalRequest struct {
	Shell   string `json:"shell"`
	WorkDir string `json:"workDir"`
	Cols    uint16 `json:"cols"`
	Rows    uint16 `json:"rows"`
}

type hostTerminalCapabilities struct {
	DefaultShell string   `json:"defaultShell"`
	Shells       []string `json:"shells"`
}
