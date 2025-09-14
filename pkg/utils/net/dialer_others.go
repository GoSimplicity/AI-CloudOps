package net

// go: build !windows
// + build !windows

import (
	"net"
	"syscall"

	"golang.org/x/sys/unix"
)

func CheckDialer() *net.Dialer {
	return &net.Dialer{
		Control: func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				linger := &unix.Linger{
					Onoff:  1,
					Linger: 1,
				}
				_ = unix.SetsockoptLinger(int(fd), unix.SOL_SOCKET, unix.SO_LINGER, linger)
			})
		},
	}
}
