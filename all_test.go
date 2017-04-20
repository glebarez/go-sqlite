// Copyright 2017 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlite

import (
	"bytes"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func caller(s string, va ...interface{}) {
	if s == "" {
		s = strings.Repeat("%v ", len(va))
	}
	_, fn, fl, _ := runtime.Caller(2)
	fmt.Fprintf(os.Stderr, "# caller: %s:%d: ", path.Base(fn), fl)
	fmt.Fprintf(os.Stderr, s, va...)
	fmt.Fprintln(os.Stderr)
	_, fn, fl, _ = runtime.Caller(1)
	fmt.Fprintf(os.Stderr, "# \tcallee: %s:%d: ", path.Base(fn), fl)
	fmt.Fprintln(os.Stderr)
	os.Stderr.Sync()
}

func dbg(s string, va ...interface{}) {
	if s == "" {
		s = strings.Repeat("%v ", len(va))
	}
	_, fn, fl, _ := runtime.Caller(1)
	fmt.Fprintf(os.Stderr, "# dbg %s:%d: ", path.Base(fn), fl)
	fmt.Fprintf(os.Stderr, s, va...)
	fmt.Fprintln(os.Stderr)
	os.Stderr.Sync()
}

func TODO(...interface{}) string { //TODOOK
	_, fn, fl, _ := runtime.Caller(1)
	return fmt.Sprintf("# TODO: %s:%d:\n", path.Base(fn), fl) //TODOOK
}

func use(...interface{}) {}

func init() {
	use(caller, dbg, TODO) //TODOOK
}

// ============================================================================

func tempDB(t *testing.T) (string, *sql.DB) {
	dir, err := ioutil.TempDir("", "sqlite-test-")
	if err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open(driverName, filepath.Join(dir, "tmp.db"))
	if err != nil {
		os.RemoveAll(dir)
		t.Fatal(err)
	}

	return dir, db
}

func TestScalar(t *testing.T) {
	dir, db := tempDB(t)
	defer func() {
		db.Close()
		os.RemoveAll(dir)
	}()

	t1 := time.Date(2017, 4, 20, 1, 2, 3, 56789, time.UTC)
	t2 := time.Date(2018, 5, 21, 2, 3, 4, 98765, time.UTC)
	r, err := db.Exec(`
	create table t(i int, f double, b bool, s text, t time);
	insert into t values(12, 3.14, ?, "foo", ?), (34, 2.78, ?, "bar", ?);
	`,
		true, t1,
		false, t2,
	)
	if err != nil {
		t.Fatal(err)
	}

	n, err := r.RowsAffected()
	if err != nil {
		t.Fatal(err)
	}

	if g, e := n, int64(2); g != e {
		t.Fatal(g, e)
	}

	rows, err := db.Query("select * from t")
	if err != nil {
		t.Fatal(err)
	}

	type rec struct {
		i int
		f float64
		b bool
		s string
		t string
	}
	var a []rec
	for rows.Next() {
		var r rec
		if err := rows.Scan(&r.i, &r.f, &r.b, &r.s, &r.t); err != nil {
			t.Fatal(err)
		}

		a = append(a, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	if g, e := len(a), 2; g != e {
		t.Fatal(g, e)
	}

	if g, e := a[0], (rec{12, 3.14, true, "foo", t1.String()}); g != e {
		t.Fatal(g, e)
	}

	if g, e := a[1], (rec{34, 2.78, false, "bar", t2.String()}); g != e {
		t.Fatal(g, e)
	}
}

func TestBlob(t *testing.T) {
	dir, db := tempDB(t)
	defer func() {
		db.Close()
		os.RemoveAll(dir)
	}()

	b1 := []byte(time.Now().String())
	b2 := []byte("\x00foo\x00bar\x00")
	if _, err := db.Exec(`
	create table t(b blob);
	insert into t values(?), (?);
	`, b1, b2,
	); err != nil {
		t.Fatal(err)
	}

	rows, err := db.Query("select * from t")
	if err != nil {
		t.Fatal(err)
	}

	type rec struct {
		b []byte
	}
	var a []rec
	for rows.Next() {
		var r rec
		if err := rows.Scan(&r.b); err != nil {
			t.Fatal(err)
		}

		a = append(a, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	if g, e := len(a), 2; g != e {
		t.Fatal(g, e)
	}

	if g, e := a[0].b, b1; !bytes.Equal(g, e) {
		t.Fatal(g, e)
	}

	if g, e := a[1].b, b2; !bytes.Equal(g, e) {
		t.Fatal(g, e)
	}
}
