// Copyright 2021 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// this file contains pure benchmark functions
// these may be wrapped by different runners
package benchmark

import (
	"database/sql"
	"testing"
)

// all benchmark of N functions to be run
var allBenchmarksOfNRows = []bechmarkOfNRows{
	benchmarkInsert,
	benchmarkSelect,
}

// bechmarkOfNRows is a type for a function that is benchmarking something depending on rows count.
type bechmarkOfNRows func(b *testing.B, db *sql.DB, nRows int)

// benchmarkInsert measures insertion of nRows into empty test table
// the insertion is carried line by line, inside a transaction, using single prepared statement
// the passed db instance must be empty (fresh) and is NOT auto-closed inside the benchmark function
func benchmarkInsert(b *testing.B, db *sql.DB, nRows int) {
	// create test table (empty)
	createTestTable(b, db, 0)

	// prepare statement for insertion
	s, err := db.Prepare("insert into t values(?)")
	if err != nil {
		b.Fatal(err)
	}
	defer s.Close()

	// measure from here
	b.ResetTimer()

	// do N times
	for i := 0; i < b.N; i++ {
		// remove data from test table
		b.StopTimer()
		if _, err := db.Exec("delete from t"); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()

		// begin tx
		if _, err := db.Exec("begin"); err != nil {
			b.Fatal(err)
		}

		// insert nRows one by one via prepared statement
		for i := 0; i < nRows; i++ {
			if _, err := s.Exec(int64(i)); err != nil {
				b.Fatal(err)
			}
		}

		// commit tx
		if _, err := db.Exec("commit"); err != nil {
			b.Fatal(err)
		}

		b.StopTimer()
	}
}

// benchmarkSelect measures select of nRows from a test table
// the passed db instance must be empty (fresh) and is NOT auto-closed inside the benchmark function
func benchmarkSelect(b *testing.B, db *sql.DB, nRows int) {
	// create test table with data
	createTestTable(b, db, nRows)

	// scan destination
	dst := 0

	// measure from here
	b.ResetTimer()

	// do N times
	for i := 0; i < b.N; i++ {
		// prepare a rows to iterate on
		b.StopTimer()
		rows, err := db.Query("select * from t")
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()

		// iterate in rows
		for i := 0; i < nRows; i++ {
			if !rows.Next() {
				b.Fatal(rows.Err()) // createTestTable did not insert enough records? that's strange
			}

			// scan yet another value
			err = rows.Scan(&dst)
			if err != nil {
				b.Fatal(err)
			}
		}
		b.StopTimer()

		if rows.Next() {
			b.Fatal("expecting rows to be exhausted at this point")
		}
		rows.Close()
	}
}
