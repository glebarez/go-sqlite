// Copyright 2021 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package benchmark

import (
	"database/sql"
	"path"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func createDB(tb testing.TB, inMemory bool, driverName string) *sql.DB {
	var dsn string
	if inMemory {
		dsn = ":memory:"
	} else {
		dsn = path.Join(tb.TempDir(), "test.db")
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		tb.Fatal(err)
	}
	return db
}

func createTestTable(tb testing.TB, db *sql.DB, nRows int) {
	if _, err := db.Exec("drop table if exists t"); err != nil {
		tb.Fatal(err)
	}

	if _, err := db.Exec("create table t(i int)"); err != nil {
		tb.Fatal(err)
	}

	if nRows > 0 {
		s, err := db.Prepare("insert into t values(?)")
		if err != nil {
			tb.Fatal(err)
		}
		defer s.Close()

		if _, err := db.Exec("begin"); err != nil {
			tb.Fatal(err)
		}

		for i := 0; i < nRows; i++ {
			if _, err := s.Exec(int64(i)); err != nil {
				tb.Fatal(err)
			}
		}
		if _, err := db.Exec("commit"); err != nil {
			tb.Fatal(err)
		}
	}
}

func getFuncName(i interface{}) string {
	// get function name as "package.function"
	fn := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()

	// return last component
	comps := strings.Split(fn, ".")
	return comps[len(comps)-1]
}
