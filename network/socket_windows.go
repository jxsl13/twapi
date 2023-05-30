package network

import (
	"syscall"
)

func castFd(fd uintptr) syscall.Handle {
	return syscall.Handle(fd)
}
