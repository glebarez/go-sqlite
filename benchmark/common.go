// Copyright 2021 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package benchmark

import (
	"fmt"

	_ "github.com/glebarez/go-sqlite"
	_ "github.com/mattn/go-sqlite3"
)

var (
	// driver names
	drivers = []string{
		"sqlite3", // CGo SQLite
		"sqlite",  // pure-go SQLite
	}

	// whether in-memory DB used
	inMemory = []bool{
		true,
		false,
	}

	// row counts will be 1eX, where X is taken from this slice
	rowCountsE = []int{1, 2, 3, 4, 5, 6}
)

// makeName generates name for a benchmark
func makeName(inMemory bool, driver string, e int) string {
	var name string

	if driver == "sqlite" {
		name = "Go"
	} else {
		name = "CGo"
	}

	if inMemory {
		name += "_Memory"
	} else {
		name += "_OnDisk"
	}

	return fmt.Sprintf("%s_1e%d", name, e)
}
