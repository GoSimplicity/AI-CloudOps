package net

import (
	"net"
	"syscall"

	"golang.org/x/sys/unix"
)

// go: build windows
// + build windows

func CheckDialer() *net.Dialer {
	return &net.Dialer{
		Control: func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				linger := &syscall.Linger{
					Onoff:  1,
					Linger: 1,
				}
				_ = syscall.SetsockoptLinger(int(fd), unix.SOL_SOCKET, unix.SO_LINGER, linger)
			})
		},
	}
}
