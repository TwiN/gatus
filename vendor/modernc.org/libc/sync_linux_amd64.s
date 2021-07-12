// Copyright 2021 The Libc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#include "textflag.h"

TEXT ·__sync_synchronize(SB), NOSPLIT, $0
	MFENCE
	RET
