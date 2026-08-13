package api

import (
	"bufio"
	"bytes"
	"testing"
)

func TestWritePipelineSSEDataPreservesMultilineLog(t *testing.T) {
	var output bytes.Buffer
	writer := bufio.NewWriter(&output)
	if err := writePipelineSSEData(writer, "build output:\r\nfirst line\nsecond line"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Flush(); err != nil {
		t.Fatal(err)
	}
	want := "data: build output:\ndata: first line\ndata: second line\n\n"
	if output.String() != want {
		t.Fatalf("SSE output = %q, want %q", output.String(), want)
	}
}
