package api

import (
	"bytes"
	"sync"
)

const (
	maxCodeExecutionOutput = 8 * 1024 * 1024
	codeOutputHeadLimit    = 512 * 1024
)

var codeOutputTruncatedMarker = []byte("\n[GoPanel: execution output truncated]\n")

type boundedCodeOutput struct {
	mu        sync.Mutex
	head      bytes.Buffer
	tail      []byte
	truncated bool
}

func (output *boundedCodeOutput) Write(data []byte) (int, error) {
	output.mu.Lock()
	defer output.mu.Unlock()
	written := len(data)
	if output.head.Len() < codeOutputHeadLimit {
		headRemaining := codeOutputHeadLimit - output.head.Len()
		if headRemaining > len(data) {
			headRemaining = len(data)
		}
		_, _ = output.head.Write(data[:headRemaining])
		data = data[headRemaining:]
	}
	if len(data) == 0 {
		return written, nil
	}
	tailLimit := maxCodeExecutionOutput - codeOutputHeadLimit
	output.tail = append(output.tail, data...)
	if len(output.tail) > tailLimit {
		overflow := len(output.tail) - tailLimit
		copy(output.tail, output.tail[overflow:])
		output.tail = output.tail[:tailLimit]
		output.truncated = true
	}
	return written, nil
}

func (output *boundedCodeOutput) Bytes() []byte {
	output.mu.Lock()
	defer output.mu.Unlock()
	result := make([]byte, 0, output.head.Len()+len(output.tail)+len(codeOutputTruncatedMarker))
	result = append(result, output.head.Bytes()...)
	if output.truncated {
		result = append(result, codeOutputTruncatedMarker...)
	}
	result = append(result, output.tail...)
	return result
}
