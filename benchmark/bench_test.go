// Copyright 2021 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// this file allows to run benchmarks via go test
package benchmark

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"runtime"
	"testing"

	_ "github.com/glebarez/go-sqlite"
	"github.com/klauspost/cpuid/v2"
	_ "github.com/mattn/go-sqlite3"
)

var (
	// flag, allows to run each benchmark multiple times and average the results. this may provide more stable results between runs
	reps uint

	// flag, whether to use in-memory SQLite
	inMemory bool

	// benchmark funcs to execute
	funcs = []func(*testing.B, *sql.DB){
		benchCreateIndex,
		benchSelectOnStringComparison,
		benchSelectWithIndex,
		benchSelectWithoutIndex,
		benchInsert,
		benchInsertInTransaction,
		benchInsertIntoIndexed,
		benchInsertFromSelect,
		benchUpdateTextWithIndex,
		benchUpdateWithIndex,
		benchUpdateWithoutIndex,
		benchDeleteWithoutIndex,
		benchDeleteWithIndex,

		// due to very long run of this benchmark, it is disabled
		// benchDropTable,
	}
)

func TestMain(m *testing.M) {
	flag.UintVar(&reps, "rep", 1, "allows to run each benchmark multiple times and average the results. this may provide more stable results between runs")
	flag.BoolVar(&inMemory, "mem", false, "if set, use in-memory SQLite")
	flag.Parse()
	os.Exit(m.Run())
}

func TestBenchmarkSQLite(t *testing.T) {
	// print info about CPU and OS
	fmt.Println()
	fmt.Printf("goos:   %s\n", runtime.GOOS)
	fmt.Printf("goarch: %s\n", runtime.GOARCH)
	if cpu := cpuid.CPU.BrandName; cpu != "" {
		fmt.Printf("cpu:    %s\n", cpu)
	}
	fmt.Printf("repeat: %d time(s)\n", reps)
	fmt.Printf("in-memory SQLite: %v\n", inMemory)
	fmt.Println()

	// loop on functions
	for _, f := range funcs {

		var (
			nsPerOpCGo    avgVal
			nsPerOpPureGo avgVal
		)

		// run benchmark against different drivers
		for r := uint(0); r < reps; r++ {
			// -- run bench against Cgo --
			db := createDB(t, inMemory, "sqlite3")
			br := testing.Benchmark(func(b *testing.B) { f(b, db) })

			// contribue metric to average
			nsPerOpCGo.contribInt(br.NsPerOp())

			// close DB
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}

			// -- run bench against Pure-go --
			db = createDB(t, inMemory, "sqlite")
			br = testing.Benchmark(func(b *testing.B) { f(b, db) })

			// contribue metric to average
			nsPerOpPureGo.contribInt(br.NsPerOp())
			// close DB
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}
		}

		// print result row
		fmt.Printf("%-35s | %5.2fx | CGo: %7.3f ms/op | Pure-Go: %7.3f ms/op\n",
			toSnakeCase(getFuncName(f)),
			nsPerOpPureGo.val/nsPerOpCGo.val, // factor
			nsPerOpCGo.val/1e6,               // ms/op
			nsPerOpPureGo.val/1e6,            // ms/op
		)
	}
}
