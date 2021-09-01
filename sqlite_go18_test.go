// Copyright 2017 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build go1.8
// +build go1.8

package sqlite // import "modernc.org/sqlite"

import (
	"database/sql"
	"os"
	"reflect"
	"testing"
)

func TestNamedParameters(t *testing.T) {
	dir, db := tempDB(t)
	defer func() {
		db.Close()
		os.RemoveAll(dir)
	}()

	_, err := db.Exec(`
	create table t(s1 varchar(32), s2 varchar(32), s3 varchar(32), s4 varchar(32));
	insert into t values(?, @aa, $aa, @bb);
	`, "1", sql.Named("aa", "one"), sql.Named("bb", "two"))

	if err != nil {
		t.Fatal(err)
	}

	rows, err := db.Query("select * from t")
	if err != nil {
		t.Fatal(err)
	}

	rec := make([]string, 4)
	for rows.Next() {
		if err := rows.Scan(&rec[0], &rec[1], &rec[2], &rec[3]); err != nil {
			t.Fatal(err)
		}
	}

	w := []string{"1", "one", "one", "two"}
	if !reflect.DeepEqual(rec, w) {
		t.Fatal(rec, w)
	}
}
