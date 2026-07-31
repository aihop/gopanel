package api

import "io"

type nativeTerminal interface {
	io.ReadWriteCloser
	Resize(cols, rows uint16) error
}
