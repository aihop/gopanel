package client

import (
	"compress/gzip"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func openPostgresqlRecoverStream(sourceFile string, progress func(readBytes, totalBytes int64)) (io.ReadCloser, error) {
	file, err := os.Open(sourceFile)
	if err != nil {
		return nil, err
	}
	totalBytes := int64(0)
	if info, statErr := file.Stat(); statErr == nil {
		totalBytes = info.Size()
	}

	var source io.ReadCloser = file
	if progress != nil {
		source = newPostgresqlProgressReadCloser(file, totalBytes, progress)
	}
	if !strings.HasSuffix(sourceFile, ".gz") {
		return source, nil
	}
	gzipReader, err := gzip.NewReader(source)
	if err != nil {
		_ = source.Close()
		return nil, err
	}
	return &postgresqlMultiReadCloser{Reader: gzipReader, closers: []io.Closer{gzipReader, source}}, nil
}

type postgresqlMultiReadCloser struct {
	io.Reader
	closers []io.Closer
}

func (m *postgresqlMultiReadCloser) Close() error {
	var firstErr error
	for _, closer := range m.closers {
		if err := closer.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

type postgresqlProgressReadCloser struct {
	reader io.ReadCloser
	total  int64
	fn     func(readBytes, totalBytes int64)

	readBytes atomic.Int64
	stopOnce  sync.Once
	stopCh    chan struct{}
	doneCh    chan struct{}
}

func newPostgresqlProgressReadCloser(reader io.ReadCloser, total int64, fn func(readBytes, totalBytes int64)) *postgresqlProgressReadCloser {
	progressReader := &postgresqlProgressReadCloser{reader: reader, total: total, fn: fn, stopCh: make(chan struct{}), doneCh: make(chan struct{})}
	go progressReader.reportLoop()
	return progressReader
}

func (p *postgresqlProgressReadCloser) Read(buffer []byte) (int, error) {
	read, err := p.reader.Read(buffer)
	if read > 0 {
		p.readBytes.Add(int64(read))
	}
	if err == io.EOF {
		p.stopReporting()
	}
	return read, err
}

func (p *postgresqlProgressReadCloser) Close() error {
	p.stopReporting()
	return p.reader.Close()
}

func (p *postgresqlProgressReadCloser) reportLoop() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	defer close(p.doneCh)
	for {
		select {
		case <-ticker.C:
			p.fn(p.readBytes.Load(), p.total)
		case <-p.stopCh:
			p.fn(p.readBytes.Load(), p.total)
			return
		}
	}
}

func (p *postgresqlProgressReadCloser) stopReporting() {
	p.stopOnce.Do(func() { close(p.stopCh) })
	<-p.doneCh
}

type postgresqlOutputWriter struct {
	output  func(string)
	tail    []byte
	pending string
	mu      sync.Mutex
}

const postgresqlRecoverErrorTailBytes = 64 * 1024

func newPostgresqlOutputWriter(output func(string)) *postgresqlOutputWriter {
	return &postgresqlOutputWriter{output: output}
}

func (w *postgresqlOutputWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.tail = append(w.tail, data...)
	if len(w.tail) > postgresqlRecoverErrorTailBytes {
		w.tail = append([]byte(nil), w.tail[len(w.tail)-postgresqlRecoverErrorTailBytes:]...)
	}
	w.pending += string(data)
	lines := strings.Split(w.pending, "\n")
	w.pending = lines[len(lines)-1]
	for _, line := range lines[:len(lines)-1] {
		w.emit(line)
	}
	return len(data), nil
}

func (w *postgresqlOutputWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return string(w.tail)
}

func (w *postgresqlOutputWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.pending != "" {
		w.emit(w.pending)
		w.pending = ""
	}
	return nil
}

func (w *postgresqlOutputWriter) emit(line string) {
	if line = strings.TrimSpace(line); line != "" && w.output != nil {
		w.output(line)
	}
}
