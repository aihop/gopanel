//go:build !windows

package api

import (
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestCountCodeTaskUntrackedLinesSkipsFIFO(t *testing.T) {
	fifoPath := filepath.Join(t.TempDir(), "blocked.pipe")
	if err := syscall.Mkfifo(fifoPath, 0600); err != nil {
		t.Fatal(err)
	}
	result := make(chan int, 1)
	go func() {
		result <- countCodeTaskUntrackedLines(fifoPath)
	}()
	select {
	case lines := <-result:
		if lines != 0 {
			t.Fatalf("FIFO line count = %d, want 0", lines)
		}
	case <-time.After(time.Second):
		t.Fatal("FIFO line count blocked task summary")
	}
}
