// Copyright (c) 2021 Andy Pan
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build linux
// +build linux

package io

import "golang.org/x/sys/unix"

// Writev calls writev() on Linux.
func Writev(fd int, iov [][]byte) (int, error) {
	if len(iov) == 0 {
		return 0, nil
	}
	return unix.Writev(fd, iov)
}

// Readv calls readv() on Linux.
func Readv(fd int, iov [][]byte) (int, error) {
	if len(iov) == 0 {
		return 0, nil
	}
	return unix.Readv(fd, iov)
}

// Write calls write() on Linux.
func Write(fd int, b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	return unix.Write(fd, b)
}

// Read calls read() on Linux.
func Read(fd int, b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	return unix.Read(fd, b)
}