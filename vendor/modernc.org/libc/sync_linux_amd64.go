// Copyright 2021 The Libc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package libc // import "modernc.org/libc"

// __sync_synchronize();
func X__sync_synchronize(t *TLS) {
	__sync_synchronize()
}

func __sync_synchronize()
