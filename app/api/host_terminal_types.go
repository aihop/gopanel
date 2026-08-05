package api

import "time"

const (
	hostTerminalHistoryLimit       = 1024 * 1024
	hostTerminalBaselineChunkLimit = 64 * 1024
	hostTerminalControlLease       = 60 * time.Second
	hostTerminalHandoverGrace      = 2 * time.Minute
)

type hostTerminalEvent struct {
	Type           string `json:"type"`
	Sequence       uint64 `json:"sequence,omitempty"`
	Data           string `json:"data,omitempty"`
	HasControl     bool   `json:"hasControl,omitempty"`
	Truncated      bool   `json:"truncated,omitempty"`
	LeaseExpiresAt int64  `json:"leaseExpiresAt,omitempty"`
	ChunkIndex     int    `json:"chunkIndex,omitempty"`
	ChunkCount     int    `json:"chunkCount,omitempty"`
}

func splitHostTerminalBaseline(event hostTerminalEvent) []hostTerminalEvent {
	if event.Type != "baseline" || len(event.Data) <= hostTerminalBaselineChunkLimit {
		return []hostTerminalEvent{event}
	}
	data := []byte(event.Data)
	chunks := make([]hostTerminalEvent, 0, (len(event.Data)+hostTerminalBaselineChunkLimit-1)/hostTerminalBaselineChunkLimit)
	for start := 0; start < len(data); {
		end := terminalChunkEnd(data, start, hostTerminalBaselineChunkLimit)
		chunk := event
		chunk.Data = string(data[start:end])
		chunk.ChunkIndex = len(chunks)
		chunk.Truncated = event.Truncated && start == 0
		chunks = append(chunks, chunk)
		start = end
	}
	for index := range chunks {
		chunks[index].ChunkCount = len(chunks)
	}
	return chunks
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
