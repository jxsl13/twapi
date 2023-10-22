//go:build unix

package network

func castFd(fd uintptr) int {
	return int(fd)
}
