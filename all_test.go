// Copyright 2017 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlite

import (
	"bytes"
	"database/sql"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
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

var (
	recsPerSec = flag.Bool("recs_per_sec_as_mbps", false, "Show records per second as MB/s.")
)

func tempDB(t testing.TB) (string, *sql.DB) {
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

func BenchmarkInsertMemory(b *testing.B) {
	db, err := sql.Open(driverName, "file::memory:")
	if err != nil {
		b.Fatal(err)
	}

	defer func() {
		db.Close()
	}()

	if _, err := db.Exec(`
	create table t(i int);
	begin;
	`); err != nil {
		b.Fatal(err)
	}

	s, err := db.Prepare("insert into t values(?)")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := s.Exec(int64(i)); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	if *recsPerSec {
		b.SetBytes(1e6)
	}
	if _, err := db.Exec(`commit;`); err != nil {
		b.Fatal(err)
	}
}

func BenchmarkNextMemory(b *testing.B) {
	db, err := sql.Open(driverName, "file::memory:")
	if err != nil {
		b.Fatal(err)
	}

	defer func() {
		db.Close()
	}()

	if _, err := db.Exec(`
	create table t(i int);
	begin;
	`); err != nil {
		b.Fatal(err)
	}

	s, err := db.Prepare("insert into t values(?)")
	if err != nil {
		b.Fatal(err)
	}

	defer s.Close()

	for i := 0; i < b.N; i++ {
		if _, err := s.Exec(int64(i)); err != nil {
			b.Fatal(err)
		}
	}
	if _, err := db.Exec("commit"); err != nil {
		b.Fatal(err)
	}

	r, err := db.Query("select * from t")
	if err != nil {

	}

	defer r.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !r.Next() {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	if *recsPerSec {
		b.SetBytes(1e6)
	}
}

// https://github.com/cznic/sqlite/issues/11
func TestIssue11(t *testing.T) {
	const N = 6570
	dir, db := tempDB(t)

	defer func() {
		db.Close()
		os.RemoveAll(dir)
	}()

	if _, err := db.Exec(`
	CREATE TABLE t1 (t INT);
	BEGIN;
`,
	); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < N; i++ {
		if _, err := db.Exec("INSERT INTO t1 (t) VALUES (?)", i); err != nil {
			t.Fatalf("#%v: %v", i, err)
		}
	}
	if _, err := db.Exec("COMMIT;"); err != nil {
		t.Fatal(err)
	}
}

// https://github.com/cznic/sqlite/issues/12
func TestMemDB(t *testing.T) {
	// Verify we can create out-of-the heap memory DB instance.
	db, err := sql.Open(driverName, "file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		db.Close()
	}()

	v := strings.Repeat("a", 1024)
	if _, err := db.Exec(`
	create table t(s string);
	begin;
	`); err != nil {
		t.Fatal(err)
	}

	s, err := db.Prepare("insert into t values(?)")
	if err != nil {
		t.Fatal(err)
	}

	// Heap used to be fixed at 32MB.
	for i := 0; i < (64<<20)/len(v); i++ {
		if _, err := s.Exec(v); err != nil {
			t.Fatalf("%v * %v= %v: %v", i, len(v), i*len(v), err)
		}
	}
	if _, err := db.Exec(`commit;`); err != nil {
		t.Fatal(err)
	}
}

func TestMP(t *testing.T) {
	dir, err := ioutil.TempDir("", "sqlite-test-")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Fatal(err)
		}
	}()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatal(err)
		}
	}()

	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	if out, err := exec.Command("go", "build", "-o", "mptest", "github.com/cznic/sqlite/internal/mptest").CombinedOutput(); err != nil {
		t.Fatalf("go build mptest: %s\n%s", err, out)
	}

	pat := filepath.Join(wd, filepath.FromSlash("testdata/mptest"), "*.test")
	m, err := filepath.Glob(pat)
	if err != nil {
		t.Fatal(err)
	}

	if len(m) == 0 {
		t.Fatalf("%s: no files", pat)
	}

	cmd := filepath.FromSlash("./mptest")
	for _, v := range m {
		os.Remove("db")
		out, err := exec.Command(cmd, "db", v).CombinedOutput()
		t.Logf("%s", out)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestThread1(t *testing.T) {
	for i := 0; i < 10; i++ {
		dir, err := ioutil.TempDir("", "sqlite-test-")
		if err != nil {
			t.Fatal(err)
		}

		defer func() {
			if err := os.RemoveAll(dir); err != nil {
				t.Fatal(err)
			}
		}()

		wd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}

		defer func() {
			if err := os.Chdir(wd); err != nil {
				t.Fatal(err)
			}
		}()

		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}

		if out, err := exec.Command("go", "build", "-o", "threadtest1", "github.com/cznic/sqlite/internal/threadtest1").CombinedOutput(); err != nil {
			t.Fatalf("go build mptest: %s\n%s", err, out)
		}

		for j := 0; j <= 20; j++ {
			out, err := exec.Command("./threadtest1", strconv.Itoa(j), "-v").CombinedOutput()
			t.Logf("%v, %v: %s", i, j, out)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestThread2(t *testing.T) {
	dir, err := ioutil.TempDir("", "sqlite-test-")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Fatal(err)
		}
	}()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatal(err)
		}
	}()

	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	if out, err := exec.Command("go", "build", "-o", "threadtest2", "github.com/cznic/sqlite/internal/threadtest2").CombinedOutput(); err != nil {
		t.Fatalf("go build mptest: %s\n%s", err, out)
	}

	out, err := exec.Command("./threadtest2").CombinedOutput()
	t.Logf("%s", out)
	if err != nil {
		t.Fatal(err)
	}
}

func TestThread4(t *testing.T) {
	dir, err := ioutil.TempDir("", "sqlite-test-")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Fatal(err)
		}
	}()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatal(err)
		}
	}()

	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	if out, err := exec.Command("go", "build", "-o", "threadtest4", "github.com/cznic/sqlite/internal/threadtest4").CombinedOutput(); err != nil {
		t.Fatalf("go build mptest: %s\n%s", err, out)
	}

	for _, opts := range [][]string{
		{},
		{"-wal"},
		{"-serialized"},
		{"-serialized", "-wal"},
		{"--multithread"},
		{"--multithread", "-wal"},
		{"--multithread", "-serialized"},
		{"--multithread", "-serialized", "-wal"},
	} {
		for i := 2; i <= 10; i++ {
			out, err := exec.Command("./threadtest4", append(opts, strconv.Itoa(i))...).CombinedOutput()
			t.Logf("%v: %v %s", i, opts, out)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}
