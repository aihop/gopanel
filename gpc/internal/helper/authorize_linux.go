//go:build linux && !windows

package helper

import (
	"net"

	"golang.org/x/sys/unix"
)

func authorizeUnixConn(c *net.UnixConn) error {
	rc, err := c.SyscallConn()
	if err != nil {
		return err
	}
	var gerr error
	_ = rc.Control(func(fd uintptr) {
		_, err := unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
		if err != nil {
			gerr = err
			return
		}
	})
	return gerr
}

