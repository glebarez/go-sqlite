// Copyright 2021 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package benchmark

import (
	"math/rand"
	"testing"
)

func test_pronounceNum(t *testing.T) {
	// this is only for visual testing
	for i := 0; i < 10; i++ {
		n := rand.Int31()
		t.Logf("%d: %s\n", n, pronounceNum(uint32(n)))
	}
}

func Benchmark_pronounceNum(b *testing.B) {
	for i := 0; i < b.N; i++ {
		n := rand.Int31()
		pronounceNum(uint32(n))
	}
}
