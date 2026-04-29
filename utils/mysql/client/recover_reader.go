package client

import (
	"bufio"
	"compress/gzip"
	"io"
	"os"
	"strings"
)

const recoverGTIDScanLines = 200

func openRecoverStream(sourceFile string) (io.ReadCloser, error) {
	file, err := os.Open(sourceFile)
	if err != nil {
		return nil, err
	}

	var src io.ReadCloser = file
	if strings.HasSuffix(sourceFile, ".gz") {
		gr, err := gzip.NewReader(file)
		if err != nil {
			_ = file.Close()
			return nil, err
		}
		src = &multiReadCloser{
			Reader: gr,
			closers: []io.Closer{
				gr,
				file,
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
