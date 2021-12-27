//go:build freebsd || dragonfly || darwin
// +build freebsd dragonfly darwin

package io

import (
	"unsafe"

	"golang.org/x/sys/unix"
)

// Writev invokes the writev system call directly.
//
// Note that SYS_WRITEV is about to be deprecated on Darwin
// and the Go team suggested to use libSystem wrappers instead of direct system-calls,
// hence, this way to implement the writev might not be backward-compatible in the future.
func Writev(fd int, bs [][]byte) (int, error) {
	if len(bs) == 0 {
		return 0, nil
	}
	iov := bytes2iovec(bs)
	n, _, err := unix.RawSyscall(unix.SYS_WRITEV, uintptr(fd), uintptr(unsafe.Pointer(&iov[0])), uintptr(len(iov)))
	if err != 0 {
		return int(n), err
	}
	return int(n), nil
}

// Readv invokes the readv system call directly.
//
// Note that SYS_READV is about to be deprecated on Darwin
// and the Go team suggested to use libSystem wrappers instead of direct system-calls,
// hence, this way to implement the readv might not be backward-compatible in the future.
func Readv(fd int, bs [][]byte) (int, error) {
	if len(bs) == 0 {
		return 0, nil
	}
	iov := bytes2iovec(bs)
	// syscall
	n, _, err := unix.RawSyscall(unix.SYS_READV, uintptr(fd), uintptr(unsafe.Pointer(&iov[0])), uintptr(len(iov)))
	if err != 0 {
		return int(n), err
	}
	return int(n), nil
}

var _zero uintptr

func bytes2iovec(bs [][]byte) []unix.Iovec {
	iovecs := make([]unix.Iovec, len(bs))
	for i, b := range bs {
		iovecs[i].SetLen(len(b))
		if len(b) > 0 {
			iovecs[i].Base = &b[0]
		} else {
			iovecs[i].Base = (*byte)(unsafe.Pointer(&_zero))
		}
	}
	return iovecs
}
