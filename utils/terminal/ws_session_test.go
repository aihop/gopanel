package terminal

import (
	"bytes"
	"testing"
)

func TestSafeBufferTakeCopiesAndResets(t *testing.T) {
	buffer := &safeBuffer{}
	if _, err := buffer.Write([]byte("output")); err != nil {
		t.Fatal(err)
	}
	data := buffer.Take()
	if !bytes.Equal(data, []byte("output")) {
		t.Fatalf("unexpected output %q", data)
	}
	if remaining := buffer.Take(); len(remaining) != 0 {
		t.Fatalf("buffer was not reset: %q", remaining)
	}
}
