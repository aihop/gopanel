package client

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestOpenPostgresqlRecoverStreamReportsCompressedBytes(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "backup.sql.gz")
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatal(err)
	}
	gzipWriter := gzip.NewWriter(file)
	want := []byte("PGDMP restore payload")
	if _, err := gzipWriter.Write(want); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	var readBytes, totalBytes int64
	stream, err := openPostgresqlRecoverStream(filePath, func(read, total int64) {
		readBytes, totalBytes = read, total
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(stream)
	if err != nil {
		t.Fatal(err)
	}
	if err := stream.Close(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected restore payload: %q", got)
	}
	if readBytes == 0 || readBytes != totalBytes {
		t.Fatalf("unexpected progress: read=%d total=%d", readBytes, totalBytes)
	}
}

func TestPostgresqlOutputWriterEmitsCompleteLines(t *testing.T) {
	var lines []string
	writer := newPostgresqlOutputWriter(func(line string) {
		lines = append(lines, line)
	})
	if _, err := writer.Write([]byte("first\nsec")); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("ond\nthird")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	want := []string{"first", "second", "third"}
	if !reflect.DeepEqual(lines, want) {
		t.Fatalf("unexpected output lines: %#v", lines)
	}
}
