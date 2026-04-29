package client

import (
	"bufio"
	"compress/gzip"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const recoverGTIDScanLines = 200

func openRecoverStream(sourceFile string, progress func(readBytes, totalBytes int64)) (io.ReadCloser, error) {
	totalBytes := int64(0)
	if info, err := os.Stat(sourceFile); err == nil {
		totalBytes = info.Size()
	}
	file, err := os.Open(sourceFile)
	if err != nil {
		return nil, err
	}

	var source io.ReadCloser = file
	if progress != nil {
		source = newProgressReadCloser(file, totalBytes, progress)
	}

	var src io.ReadCloser = source
	if strings.HasSuffix(sourceFile, ".gz") {
		gr, err := gzip.NewReader(source)
		if err != nil {
			_ = source.Close()
			return nil, err
		}
		src = &multiReadCloser{
			Reader: gr,
			closers: []io.Closer{
				gr,
				source,
			},
		}
	}

	return filterRecoverGTIDStream(src), nil
}

func filterRecoverGTIDStream(src io.ReadCloser) io.ReadCloser {
	pr, pw := io.Pipe()
	go func() {
		defer src.Close()

		reader := bufio.NewReader(src)
		scanned := 0
		for scanned < recoverGTIDScanLines {
			line, err := reader.ReadBytes('\n')
			if len(line) > 0 {
				scanned++
				if !shouldSkipRecoverLine(line) {
					if _, werr := pw.Write(line); werr != nil {
						_ = pw.CloseWithError(werr)
						return
					}
				}
			}
			if err == io.EOF {
				_ = pw.Close()
				return
			}
			if err != nil {
				_ = pw.CloseWithError(err)
				return
			}
		}

		if _, err := io.Copy(pw, reader); err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		_ = pw.Close()
	}()
	return pr
}

func shouldSkipRecoverLine(line []byte) bool {
	text := strings.TrimSpace(string(line))
	if text == "" {
		return false
	}
	upper := strings.ToUpper(text)
	return strings.Contains(upper, "GTID_PURGED")
}

type multiReadCloser struct {
	io.Reader
	closers []io.Closer
}

func (m *multiReadCloser) Close() error {
	var firstErr error
	for _, closer := range m.closers {
		if closer == nil {
			continue
		}
		if err := closer.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

type progressReadCloser struct {
	reader io.ReadCloser
	total  int64
	fn     func(readBytes, totalBytes int64)

	readBytes atomic.Int64
	stopOnce  sync.Once
	stopCh    chan struct{}
}

func newProgressReadCloser(reader io.ReadCloser, total int64, fn func(readBytes, totalBytes int64)) *progressReadCloser {
	p := &progressReadCloser{
		reader: reader,
		total:  total,
		fn:     fn,
		stopCh: make(chan struct{}),
	}
	go p.reportLoop()
	return p
}

func (p *progressReadCloser) Read(buf []byte) (int, error) {
	n, err := p.reader.Read(buf)
	if n > 0 {
		p.readBytes.Add(int64(n))
	}
	if err == io.EOF {
		p.stopReporting()
	}
	return n, err
}

func (p *progressReadCloser) Close() error {
	p.stopReporting()
	return p.reader.Close()
}

func (p *progressReadCloser) reportLoop() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			p.emit()
		case <-p.stopCh:
			p.emit()
			return
		}
	}
}

func (p *progressReadCloser) emit() {
	if p.fn == nil {
		return
	}
	p.fn(p.readBytes.Load(), p.total)
}

func (p *progressReadCloser) stopReporting() {
	p.stopOnce.Do(func() {
		close(p.stopCh)
	})
}
