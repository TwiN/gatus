// Copyright 2021 The Libc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !amd64 !linux

package libc // import "modernc.org/libc"

import (
	"sync/atomic"
	"unsafe"
)

var errno0 int32 // Temp errno for NewTLS

type TLS struct {
	ID                 int32
	errnop             uintptr
	reentryGuard       int32 // memgrind
	stack              stackHeader
	stackHeaderBalance int32
}

func NewTLS() *TLS {
	id := atomic.AddInt32(&tid, 1)
	t := &TLS{ID: id, errnop: uintptr(unsafe.Pointer(&errno0))}
	if memgrind {
		atomic.AddInt32(&tlsBalance, 1)
	}
	t.errnop = t.Alloc(int(unsafe.Sizeof(int32(0))))
	*(*int32)(unsafe.Pointer(t.errnop)) = 0
	return t
}
