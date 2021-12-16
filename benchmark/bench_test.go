// Copyright 2021 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// this file allows to run benchmarks via go test
package benchmark

import (
	"math"
	"testing"
)

func BenchmarkSelect(b *testing.B) {
	doBenchmarkOfNrows(b, benchmarkSelect)
}

// https://gitlab.com/cznic/sqlite/-/issues/39
func BenchmarkInsert(b *testing.B) {
	doBenchmarkOfNrows(b, benchmarkInsert)
}

func doBenchmarkOfNrows(b *testing.B, benchFunc bechmarkOfNRows) {
	for _, isMemoryDB := range inMemory { // in-memory: on/off
		for _, e := range rowCountsE { // number of rows in table
			for _, driverName := range drivers { // drivers

				// create new DB
				db := createDB(b, isMemoryDB, driverName)

				// run benchmark
				b.Run(
					makeName(isMemoryDB, driverName, e),
					func(b *testing.B) {
						benchFunc(b, db, int(math.Pow10(e)))
					},
				)

				// close DB
				if err := db.Close(); err != nil {
					b.Fatal(err)
				}
			}
		}
	}
}
