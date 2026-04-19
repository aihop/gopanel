//go:build !windows && !linux

package helper

import "net"

func authorizeUnixConn(c *net.UnixConn) error {
	_ = c
	return nil
}

