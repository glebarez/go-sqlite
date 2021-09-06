// Copyright 2021 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlite // import "modernc.org/sqlite"

import (
	"golang.org/x/sys/unix"
)

func setMaxOpenFiles(n int) error {
	var rLimit unix.Rlimit
	rLimit.Max = 1024
	rLimit.Cur = 1024
	return unix.Setrlimit(unix.RLIMIT_NOFILE, &rLimit)
}
