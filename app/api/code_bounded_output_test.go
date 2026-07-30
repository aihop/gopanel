package api

import (
	"bytes"
	"testing"
)

func TestBoundedCodeOutputKeepsHeadAndTail(t *testing.T) {
	output := &boundedCodeOutput{}
	head := bytes.Repeat([]byte("h"), codeOutputHeadLimit)
	middle := bytes.Repeat([]byte("m"), maxCodeExecutionOutput)
	tail := []byte("final-answer")
	_, _ = output.Write(head)
	_, _ = output.Write(middle)
	_, _ = output.Write(tail)
	result := output.Bytes()
	if len(result) > maxCodeExecutionOutput+len(codeOutputTruncatedMarker) {
		t.Fatalf("bounded output grew to %d bytes", len(result))
	}
	if !bytes.HasPrefix(result, head) || !bytes.Contains(result, codeOutputTruncatedMarker) || !bytes.HasSuffix(result, tail) {
		t.Fatal("bounded output did not preserve head, marker and tail")
	}
}
