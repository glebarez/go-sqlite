// Copyright 2017 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlite // import "modernc.org/sqlite"

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"embed"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/google/pprof/profile"
	"modernc.org/libc"
	"modernc.org/mathutil"
	sqlite3 "modernc.org/sqlite/lib"
	"modernc.org/sqlite/vfs"
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

func stack() string { return string(debug.Stack()) }

func use(...interface{}) {}

func init() {
	use(caller, dbg, stack, todo, trc) //TODOOK
}

func origin(skip int) string {
	pc, fn, fl, _ := runtime.Caller(skip)
	f := runtime.FuncForPC(pc)
	var fns string
	if f != nil {
		fns = f.Name()
		if x := strings.LastIndex(fns, "."); x > 0 {
			fns = fns[x+1:]
		}
	}
	return fmt.Sprintf("%s:%d:%s", fn, fl, fns)
}

func todo(s string, args ...interface{}) string { //TODO-
	switch {
	case s == "":
		s = fmt.Sprintf(strings.Repeat("%v ", len(args)), args...)
	default:
		s = fmt.Sprintf(s, args...)
	}
	r := fmt.Sprintf("%s: TODOTODO %s", origin(2), s) //TODOOK
	fmt.Fprintf(os.Stdout, "%s\n", r)
	os.Stdout.Sync()
	return r
}

func trc(s string, args ...interface{}) string { //TODO-
	switch {
	case s == "":
		s = fmt.Sprintf(strings.Repeat("%v ", len(args)), args...)
	default:
		s = fmt.Sprintf(s, args...)
	}
	r := fmt.Sprintf("\n%s: TRC %s", origin(2), s)
	fmt.Fprintf(os.Stdout, "%s\n", r)
	os.Stdout.Sync()
	return r
}

// ============================================================================

var (
	oRecsPerSec = flag.Bool("recs_per_sec_as_mbps", false, "Show records per second as MB/s.")
	oXTags      = flag.String("xtags", "", "passed to go build of testfixture in TestTclTest")
	tempDir     string
)

func TestMain(m *testing.M) {
	fmt.Printf("test binary compiled for %s/%s\n", runtime.GOOS, runtime.GOARCH)
	flag.Parse()
	libc.MemAuditStart()
	os.Exit(testMain(m))
}

func testMain(m *testing.M) int {
	var err error
	tempDir, err = os.MkdirTemp("", "sqlite-test-")
	if err != nil {
		panic(err) //TODOOK
	}

	defer os.RemoveAll(tempDir)

	return m.Run()
}

func tempDB(t testing.TB) (string, *sql.DB) {
	dir, err := os.MkdirTemp("", "sqlite-test-")
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

// https://gitlab.com/cznic/sqlite/issues/118
func TestIssue118(t *testing.T) {
	// Many iterations generate enough objects to ensure pprof
	// profile captures the samples that we are seeking below
	for i := 0; i < 10000; i++ {
		func() {
			db, err := sql.Open("sqlite", ":memory:")
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if _, err := db.Exec(`CREATE TABLE t1(v TEXT)`); err != nil {
				t.Fatal(err)
			}
			var val []byte
			if _, err := db.Exec(`INSERT INTO t1(v) VALUES(?)`, val); err != nil {
				t.Fatal(err)
			}
			var count int
			err = db.QueryRow("SELECT MAX(_ROWID_) FROM t1").Scan(&count)
			if err != nil || count <= 0 {
				t.Fatalf("Query failure: %d, %s", count, err)
			}
		}()
	}

	// Dump & read heap sample
	var buf bytes.Buffer
	if err := pprof.Lookup("heap").WriteTo(&buf, 0); err != nil {
		t.Fatalf("Error dumping heap profile: %s", err)
	}
	heapProfile, err := profile.Parse(&buf)
	if err != nil {
		t.Fatalf("Error parsing heap profile: %s", err)
	}

	// Profile.SampleType indexes map into Sample.Values below. We are
	// looking for "inuse_*" values, and skip the "alloc_*" ones
	inUseIndexes := make([]int, 0, 2)
	for i, t := range heapProfile.SampleType {
		if strings.HasPrefix(t.Type, "inuse_") {
			inUseIndexes = append(inUseIndexes, i)
		}
	}

	// Look for samples from "libc.NewTLS" and insure that they have nothing in-use
	for _, sample := range heapProfile.Sample {
		isInUse := false
		for _, idx := range inUseIndexes {
			isInUse = isInUse || sample.Value[idx] > 0
		}
		if !isInUse {
			continue
		}

		isNewTLS := false
		sampleStack := []string{}
		for _, location := range sample.Location {
			for _, line := range location.Line {
				sampleStack = append(sampleStack, fmt.Sprintf("%s (%s:%d)", line.Function.Name, line.Function.Filename, line.Line))
				isNewTLS = isNewTLS || strings.Contains(line.Function.Name, "libc.NewTLS")
			}
		}
		if isNewTLS {
			t.Errorf("Memory leak via libc.NewTLS:\n%s\n", strings.Join(sampleStack, "\n"))
		}
	}
}

// https://gitlab.com/cznic/sqlite/issues/100
func TestIssue100(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE t1(v TEXT)`); err != nil {
		t.Fatal(err)
	}
	var val []byte
	if _, err := db.Exec(`INSERT INTO t1(v) VALUES(?)`, val); err != nil {
		t.Fatal(err)
	}
	var res sql.NullString
	if err = db.QueryRow(`SELECT v FROM t1 LIMIT 1`).Scan(&res); err != nil {
		t.Fatal(err)
	}
	if res.Valid {
		t.Fatalf("got non-NULL result: %+v", res)
	}

	if _, err := db.Exec(`CREATE TABLE t2(
		v TEXT check(v is NULL OR(json_valid(v) AND json_type(v)='array'))
	)`); err != nil {
		t.Fatal(err)
	}
	for _, val := range [...][]byte{nil, []byte(`["a"]`)} {
		if _, err := db.Exec(`INSERT INTO t2(v) VALUES(?)`, val); err != nil {
			t.Fatalf("inserting value %v (%[1]q): %v", val, err)
		}
	}
}

// https://gitlab.com/cznic/sqlite/issues/98
func TestIssue98(t *testing.T) {
	dir, db := tempDB(t)

	defer func() {
		db.Close()
		os.RemoveAll(dir)
	}()

	if _, err := db.Exec("create table t(b mediumblob not null)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("insert into t values (?)", []byte{}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("insert into t values (?)", nil); err == nil {
		t.Fatal("expected statement to fail")
	}
}

// https://gitlab.com/cznic/sqlite/issues/97
func TestIssue97(t *testing.T) {
	name := filepath.Join(t.TempDir(), "tmp.db")

	db, err := sql.Open(driverName, fmt.Sprintf("file:%s", name))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec("create table t(b int)"); err != nil {
		t.Fatal(err)
	}

	rodb, err := sql.Open(driverName, fmt.Sprintf("file:%s?mode=ro", name))
	if err != nil {
		t.Fatal(err)
	}
	defer rodb.Close()

	_, err = rodb.Exec("drop table t")
	if err == nil {
		t.Fatal("expected drop table statement to fail on a read only database")
	} else if err.Error() != "attempt to write a readonly database (8)" {
		t.Fatal("expected drop table statement to fail because its a readonly database")
	}
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
	insert into t values(12, 3.14, ?, 'foo', ?), (34, 2.78, ?, 'bar', ?);
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

	if g, e := a[0], (rec{12, 3.14, true, "foo", t1.Format(parseTimeFormats[0])}); g != e {
		t.Fatal(g, e)
	}

	if g, e := a[1], (rec{34, 2.78, false, "bar", t2.Format(parseTimeFormats[0])}); g != e {
		t.Fatal(g, e)
	}
}

func TestRedefineUserDefinedFunction(t *testing.T) {
	dir, db := tempDB(t)
	ctx := context.Background()

	defer func() {
		db.Close()
		os.RemoveAll(dir)
	}()

	connection, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	var r int
	funName := "test"

	if err = connection.Raw(func(driverConn interface{}) error {
		c := driverConn.(*conn)

		name, err := libc.CString(funName)
		if err != nil {
			return err
		}

		return c.createFunctionInternal(&userDefinedFunction{
			zFuncName: name,
			nArg:      0,
			eTextRep:  sqlite3.SQLITE_UTF8 | sqlite3.SQLITE_DETERMINISTIC,
			xFunc: func(tls *libc.TLS, ctx uintptr, argc int32, argv uintptr) {
				sqlite3.Xsqlite3_result_int(tls, ctx, 1)
			},
		})
	}); err != nil {
		t.Fatal(err)
	}
	row := connection.QueryRowContext(ctx, "select test()")

	if err := row.Scan(&r); err != nil {
		t.Fatal(err)
	}

	if g, e := r, 1; g != e {
		t.Fatal(g, e)
	}

	if err = connection.Raw(func(driverConn interface{}) error {
		c := driverConn.(*conn)

		name, err := libc.CString(funName)
		if err != nil {
			return err
		}

		return c.createFunctionInternal(&userDefinedFunction{
			zFuncName: name,
			nArg:      0,
			eTextRep:  sqlite3.SQLITE_UTF8 | sqlite3.SQLITE_DETERMINISTIC,
			xFunc: func(tls *libc.TLS, ctx uintptr, argc int32, argv uintptr) {
				sqlite3.Xsqlite3_result_int(tls, ctx, 2)
			},
		})
	}); err != nil {
		t.Fatal(err)
	}
	row = connection.QueryRowContext(ctx, "select test()")

	if err := row.Scan(&r); err != nil {
		t.Fatal(err)
	}

	if g, e := r, 2; g != e {
		t.Fatal(g, e)
	}
}

func TestRegexpUserDefinedFunction(t *testing.T) {
	dir, db := tempDB(t)
	ctx := context.Background()

	defer func() {
		db.Close()
		os.RemoveAll(dir)
	}()

	connection, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if err = connection.Raw(func(driverConn interface{}) error {
		c := driverConn.(*conn)

		name, err := libc.CString("regexp")
		if err != nil {
			return err
		}

		return c.createFunctionInternal(&userDefinedFunction{
			zFuncName: name,
			nArg:      2,
			eTextRep:  sqlite3.SQLITE_UTF8 | sqlite3.SQLITE_DETERMINISTIC,
			xFunc: func(tls *libc.TLS, ctx uintptr, argc int32, argv uintptr) {
				const sqliteValPtrSize = unsafe.Sizeof(&sqlite3.Sqlite3_value{})

				argvv := make([]uintptr, argc)
				for i := int32(0); i < argc; i++ {
					argvv[i] = *(*uintptr)(unsafe.Pointer(argv + uintptr(i)*sqliteValPtrSize))
				}

				setErrorResult := func(res error) {
					errmsg, cerr := libc.CString(res.Error())
					if cerr != nil {
						panic(cerr)
					}
					defer libc.Xfree(tls, errmsg)
					sqlite3.Xsqlite3_result_error(tls, ctx, errmsg, -1)
					sqlite3.Xsqlite3_result_error_code(tls, ctx, sqlite3.SQLITE_ERROR)
				}

				var s1 string
				switch sqlite3.Xsqlite3_value_type(tls, argvv[0]) {
				case sqlite3.SQLITE_TEXT:
					s1 = libc.GoString(sqlite3.Xsqlite3_value_text(tls, argvv[0]))
				default:
					setErrorResult(errors.New("expected argv[0] to be text"))
					return
				}

				var s2 string
				switch sqlite3.Xsqlite3_value_type(tls, argvv[1]) {
				case sqlite3.SQLITE_TEXT:
					s2 = libc.GoString(sqlite3.Xsqlite3_value_text(tls, argvv[1]))
				default:
					setErrorResult(errors.New("expected argv[1] to be text"))
					return
				}

				matched, err := regexp.MatchString(s1, s2)
				if err != nil {
					setErrorResult(fmt.Errorf("bad regular expression: %q", err))
					return
				}
				sqlite3.Xsqlite3_result_int(tls, ctx, libc.Bool32(matched))
			},
		})
	}); err != nil {
		t.Fatal(err)
	}

	t.Run("regexp filter", func(tt *testing.T) {
		t1 := "seafood"
		t2 := "fruit"

		connection.ExecContext(ctx, `
create table t(b text);
insert into t values(?), (?);
`, t1, t2)

		rows, err := connection.QueryContext(ctx, "select * from t where b regexp 'foo.*'")
		if err != nil {
			tt.Fatal(err)
		}

		type rec struct {
			b string
		}
		var a []rec
		for rows.Next() {
			var r rec
			if err := rows.Scan(&r.b); err != nil {
				tt.Fatal(err)
			}

			a = append(a, r)
		}
		if err := rows.Err(); err != nil {
			tt.Fatal(err)
		}

		if g, e := len(a), 1; g != e {
			tt.Fatal(g, e)
		}

		if g, e := a[0].b, t1; g != e {
			tt.Fatal(g, e)
		}
	})

	t.Run("regexp matches", func(tt *testing.T) {
		row := connection.QueryRowContext(ctx, "select 'seafood' regexp 'foo.*'")

		var r int
		if err := row.Scan(&r); err != nil {
			tt.Fatal(err)
		}

		if g, e := r, 1; g != e {
			tt.Fatal(g, e)
		}
	})

	t.Run("regexp does not match", func(tt *testing.T) {
		row := connection.QueryRowContext(ctx, "select 'fruit' regexp 'foo.*'")

		var r int
		if err := row.Scan(&r); err != nil {
			tt.Fatal(err)
		}

		if g, e := r, 0; g != e {
			tt.Fatal(g, e)
		}
	})

	t.Run("errors on bad regexp", func(tt *testing.T) {
		err := connection.QueryRowContext(ctx, "select 'seafood' regexp 'a(b'").Scan()
		if err == nil {
			tt.Fatal(errors.New("expected error, got none"))
		}
	})

	t.Run("errors on bad first argument", func(tt *testing.T) {
		err := connection.QueryRowContext(ctx, "SELECT 1 REGEXP 'a(b'").Scan()
		if err == nil {
			tt.Fatal(errors.New("expected error, got none"))
		}
	})

	t.Run("errors on bad second argument", func(tt *testing.T) {
		err := connection.QueryRowContext(ctx, "SELECT 'seafood' REGEXP 1").Scan()
		if err == nil {
			tt.Fatal(errors.New("expected error, got none"))
		}
	})
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

func benchmarkInsertMemory(b *testing.B, n int) {
	db, err := sql.Open(driverName, "file::memory:")
	if err != nil {
		b.Fatal(err)
	}

	defer func() {
		db.Close()
	}()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		if _, err := db.Exec(`
		drop table if exists t;
		create table t(i int);
		begin;
		`); err != nil {
			b.Fatal(err)
		}

		s, err := db.Prepare("insert into t values(?)")
		if err != nil {
			b.Fatal(err)
		}

		b.StartTimer()
		for i := 0; i < n; i++ {
			if _, err := s.Exec(int64(i)); err != nil {
				b.Fatal(err)
			}
		}
		b.StopTimer()
		if _, err := db.Exec(`commit;`); err != nil {
			b.Fatal(err)
		}
	}
	if *oRecsPerSec {
		b.SetBytes(1e6 * int64(n))
	}
}

func BenchmarkInsertMemory(b *testing.B) {
	for i, n := range []int{1e1, 1e2, 1e3, 1e4, 1e5, 1e6} {
		b.Run(fmt.Sprintf("1e%d", i+1), func(b *testing.B) { benchmarkInsertMemory(b, n) })
	}
}

var staticInt int

func benchmarkNextMemory(b *testing.B, n int) {
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

	for i := 0; i < n; i++ {
		if _, err := s.Exec(int64(i)); err != nil {
			b.Fatal(err)
		}
	}
	if _, err := db.Exec(`commit;`); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		r, err := db.Query("select * from t")
		if err != nil {
			b.Fatal(err)
		}

		b.StartTimer()
		for i := 0; i < n; i++ {
			if !r.Next() {
				b.Fatal(err)
			}
			if err := r.Scan(&staticInt); err != nil {
				b.Fatal(err)
			}
		}
		b.StopTimer()
		if err := r.Err(); err != nil {
			b.Fatal(err)
		}

		r.Close()
	}
	if *oRecsPerSec {
		b.SetBytes(1e6 * int64(n))
	}
}

func BenchmarkNextMemory(b *testing.B) {
	for i, n := range []int{1e1, 1e2, 1e3, 1e4, 1e5, 1e6} {
		b.Run(fmt.Sprintf("1e%d", i+1), func(b *testing.B) { benchmarkNextMemory(b, n) })
	}
}

// https://gitlab.com/cznic/sqlite/issues/11
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

// https://gitlab.com/cznic/sqlite/issues/12
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

func TestConcurrentGoroutines(t *testing.T) {
	const (
		ngoroutines = 8
		nrows       = 5000
	)

	dir, err := os.MkdirTemp("", "sqlite-test-")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		os.RemoveAll(dir)
	}()

	db, err := sql.Open(driverName, filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}

	defer db.Close()

	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := tx.Exec("create table t(i)"); err != nil {
		t.Fatal(err)
	}

	prep, err := tx.Prepare("insert into t values(?)")
	if err != nil {
		t.Fatal(err)
	}

	rnd := make(chan int, 100)
	go func() {
		lim := ngoroutines * nrows
		rng, err := mathutil.NewFC32(0, lim-1, false)
		if err != nil {
			panic(fmt.Errorf("internal error: %v", err))
		}

		for i := 0; i < lim; i++ {
			rnd <- rng.Next()
		}
	}()

	start := make(chan int)
	var wg sync.WaitGroup
	for i := 0; i < ngoroutines; i++ {
		wg.Add(1)

		go func(id int) {

			defer wg.Done()

		next:
			for i := 0; i < nrows; i++ {
				n := <-rnd
				var err error
				for j := 0; j < 10; j++ {
					if _, err := prep.Exec(n); err == nil {
						continue next
					}
				}

				t.Errorf("id %d, seq %d: %v", id, i, err)
				return
			}
		}(i)
	}
	t0 := time.Now()
	close(start)
	wg.Wait()
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	d := time.Since(t0)
	rows, err := db.Query("select * from t order by i")
	if err != nil {
		t.Fatal(err)
	}

	var i int
	for ; rows.Next(); i++ {
		var j int
		if err := rows.Scan(&j); err != nil {
			t.Fatalf("seq %d: %v", i, err)
		}

		if g, e := j, i; g != e {
			t.Fatalf("seq %d: got %d, exp %d", i, g, e)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	if g, e := i, ngoroutines*nrows; g != e {
		t.Fatalf("got %d rows, expected %d", g, e)
	}

	t.Logf("%d goroutines concurrently inserted %d rows in %v", ngoroutines, ngoroutines*nrows, d)
}

func TestConcurrentProcesses(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	dir, err := os.MkdirTemp("", "sqlite-test-")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		os.RemoveAll(dir)
	}()

	m, err := filepath.Glob(filepath.FromSlash("internal/mptest/*"))
	if err != nil {
		t.Fatal(err)
	}

	for _, v := range m {
		if s := filepath.Ext(v); s != ".test" && s != ".subtest" {
			continue
		}

		b, err := os.ReadFile(v)
		if err != nil {
			t.Fatal(err)
		}

		if runtime.GOOS == "windows" {
			// reference tests are in *nix format --
			// but git on windows does line-ending xlation by default
			// if someone has it 'off' this has no impact.
			// '\r\n'  -->  '\n'
			b = bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
		}

		if err := os.WriteFile(filepath.Join(dir, filepath.Base(v)), b, 0666); err != nil {
			t.Fatal(err)
		}
	}

	bin := "./mptest"
	if runtime.GOOS == "windows" {
		bin += "mptest.exe"
	}
	args := []string{"build", "-o", filepath.Join(dir, bin)}
	if s := *oXTags; s != "" {
		args = append(args, "-tags", s)
	}
	args = append(args, "modernc.org/sqlite/internal/mptest")
	out, err := exec.Command("go", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("%s\n%v", out, err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	defer os.Chdir(wd)

	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

outer:
	for _, script := range m {
		script = filepath.Base(script)
		if filepath.Ext(script) != ".test" {
			continue
		}

		fmt.Printf("exec: %s db %s\n", filepath.FromSlash(bin), script)
		out, err := exec.Command(filepath.FromSlash(bin), "db", "--timeout", "6000000", script).CombinedOutput()
		if err != nil {
			t.Fatalf("%s\n%v", out, err)
		}

		// just remove it so we don't get a
		// file busy race-condition
		// when we spin up the next script
		if runtime.GOOS == "windows" {
			_ = os.Remove("db")
		}

		a := strings.Split(string(out), "\n")
		for _, v := range a {
			if strings.HasPrefix(v, "Summary:") {
				b := strings.Fields(v)
				if len(b) < 2 {
					t.Fatalf("unexpected format of %q", v)
				}

				n, err := strconv.Atoi(b[1])
				if err != nil {
					t.Fatalf("unexpected format of %q", v)
				}

				if n != 0 {
					t.Errorf("%s", out)
				}

				t.Logf("%v: %v", script, v)
				continue outer
			}

		}
		t.Fatalf("%s\nerror: summary line not found", out)
	}
}

// https://gitlab.com/cznic/sqlite/issues/19
func TestIssue19(t *testing.T) {
	const (
		drop = `
drop table if exists products;
`

		up = `
CREATE TABLE IF NOT EXISTS "products" (
	"id"	VARCHAR(255),
	"user_id"	VARCHAR(255),
	"name"	VARCHAR(255),
	"description"	VARCHAR(255),
	"created_at"	BIGINT,
	"credits_price"	BIGINT,
	"enabled"	BOOLEAN,
	PRIMARY KEY("id")
);
`

		productInsert = `
INSERT INTO "products" ("id", "user_id", "name", "description", "created_at", "credits_price", "enabled") VALUES ('9be4398c-d527-4efb-93a4-fc532cbaf804', '16935690-348b-41a6-bb20-f8bb16011015', 'dqdwqdwqdwqwqdwqd', 'qwdwqwqdwqdwqdwqd', '1577448686', '1', '0');
INSERT INTO "products" ("id", "user_id", "name", "description", "created_at", "credits_price", "enabled") VALUES ('759f10bd-9e1d-4ec7-b764-0868758d7b85', '16935690-348b-41a6-bb20-f8bb16011015', 'qdqwqwdwqdwqdwqwqd', 'wqdwqdwqdwqdwqdwq', '1577448692', '1', '1');
INSERT INTO "products" ("id", "user_id", "name", "description", "created_at", "credits_price", "enabled") VALUES ('512956e7-224d-4b2a-9153-b83a52c4aa38', '16935690-348b-41a6-bb20-f8bb16011015', 'qwdwqwdqwdqdwqwqd', 'wqdwdqwqdwqdwqdwqdwqdqw', '1577448699', '2', '1');
INSERT INTO "products" ("id", "user_id", "name", "description", "created_at", "credits_price", "enabled") VALUES ('02cd138f-6fa6-4909-9db7-a9d0eca4a7b7', '16935690-348b-41a6-bb20-f8bb16011015', 'qdwqdwqdwqwqdwdq', 'wqddwqwqdwqdwdqwdqwq', '1577448706', '3', '1');
`
	)

	dir, err := os.MkdirTemp("", "sqlite-test-")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		os.RemoveAll(dir)
	}()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	defer os.Chdir(wd)

	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("sqlite", "test.db")
	if err != nil {
		t.Fatal("failed to connect database")
	}

	defer db.Close()

	db.SetMaxOpenConns(1)

	if _, err = db.Exec(drop); err != nil {
		t.Fatal(err)
	}

	if _, err = db.Exec(up); err != nil {
		t.Fatal(err)
	}

	if _, err = db.Exec(productInsert); err != nil {
		t.Fatal(err)
	}

	var count int64
	if err = db.QueryRow("select count(*) from products where user_id = ?", "16935690-348b-41a6-bb20-f8bb16011015").Scan(&count); err != nil {
		t.Fatal(err)
	}

	if count != 4 {
		t.Fatalf("expected result for the count query %d, we received %d\n", 4, count)
	}

	rows, err := db.Query("select * from products where user_id = ?", "16935690-348b-41a6-bb20-f8bb16011015")
	if err != nil {
		t.Fatal(err)
	}

	count = 0
	for rows.Next() {
		count++
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	if count != 4 {
		t.Fatalf("expected result for the select query %d, we received %d\n", 4, count)
	}

	rows, err = db.Query("select * from products where enabled = ?", true)
	if err != nil {
		t.Fatal(err)
	}

	count = 0
	for rows.Next() {
		count++
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	if count != 3 {
		t.Fatalf("expected result for the enabled select query %d, we received %d\n", 3, count)
	}
}

func mustExec(t *testing.T, db *sql.DB, sql string, args ...interface{}) sql.Result {
	res, err := db.Exec(sql, args...)
	if err != nil {
		t.Fatalf("Error running %q: %v", sql, err)
	}

	return res
}

// https://gitlab.com/cznic/sqlite/issues/20
func TestIssue20(t *testing.T) {
	const TablePrefix = "gosqltest_"

	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		os.RemoveAll(tempDir)
	}()

	// go1.20rc1, linux/ppc64le VM
	// 10000 FAIL
	// 20000 FAIL
	// 40000 PASS
	// 30000 PASS
	// 25000 PASS
	db, err := sql.Open("sqlite", filepath.Join(tempDir, "foo.db")+"?_pragma=busy_timeout%3d50000")
	if err != nil {
		t.Fatalf("foo.db open fail: %v", err)
	}

	defer db.Close()

	mustExec(t, db, "CREATE TABLE "+TablePrefix+"t (count INT)")
	sel, err := db.PrepareContext(context.Background(), "SELECT count FROM "+TablePrefix+"t ORDER BY count DESC")
	if err != nil {
		t.Fatalf("prepare 1: %v", err)
	}

	ins, err := db.PrepareContext(context.Background(), "INSERT INTO "+TablePrefix+"t (count) VALUES (?)")
	if err != nil {
		t.Fatalf("prepare 2: %v", err)
	}

	for n := 1; n <= 3; n++ {
		if _, err := ins.Exec(n); err != nil {
			t.Fatalf("insert(%d) = %v", n, err)
		}
	}

	const nRuns = 10
	ch := make(chan bool)
	for i := 0; i < nRuns; i++ {
		go func() {
			defer func() {
				ch <- true
			}()
			for j := 0; j < 10; j++ {
				count := 0
				if err := sel.QueryRow().Scan(&count); err != nil && err != sql.ErrNoRows {
					t.Errorf("Query: %v", err)
					return
				}

				if _, err := ins.Exec(rand.Intn(100)); err != nil {
					t.Errorf("Insert: %v", err)
					return
				}
			}
		}()
	}
	for i := 0; i < nRuns; i++ {
		<-ch
	}
}

func TestNoRows(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		os.RemoveAll(tempDir)
	}()

	db, err := sql.Open("sqlite", filepath.Join(tempDir, "foo.db"))
	if err != nil {
		t.Fatalf("foo.db open fail: %v", err)
	}

	defer func() {
		db.Close()
	}()

	stmt, err := db.Prepare("create table t(i);")
	if err != nil {
		t.Fatal(err)
	}

	defer stmt.Close()

	if _, err := stmt.Query(); err != nil {
		t.Fatal(err)
	}
}

func TestColumns(t *testing.T) {
	db, err := sql.Open("sqlite", "file::memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec("create table t1(a integer, b text, c blob)"); err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec("insert into t1 (a) values (1)"); err != nil {
		t.Fatal(err)
	}

	rows, err := db.Query("select * from t1")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	got, err := rows.Columns()
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got columns %v, want %v", got, want)
	}
}

// https://gitlab.com/cznic/sqlite/-/issues/32
func TestColumnsNoRows(t *testing.T) {
	db, err := sql.Open("sqlite", "file::memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec("create table t1(a integer, b text, c blob)"); err != nil {
		t.Fatal(err)
	}

	rows, err := db.Query("select * from t1")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	got, err := rows.Columns()
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got columns %v, want %v", got, want)
	}
}

// https://gitlab.com/cznic/sqlite/-/issues/28
func TestIssue28(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		os.RemoveAll(tempDir)
	}()

	db, err := sql.Open("sqlite", filepath.Join(tempDir, "test.db"))
	if err != nil {
		t.Fatalf("test.db open fail: %v", err)
	}

	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE test (foo TEXT)`); err != nil {
		t.Fatal(err)
	}

	row := db.QueryRow(`SELECT foo FROM test`)
	var foo string
	if err = row.Scan(&foo); err != sql.ErrNoRows {
		t.Fatalf("got %T(%[1]v), expected %T(%[2]v)", err, sql.ErrNoRows)
	}
}

// https://gitlab.com/cznic/sqlite/-/issues/30
func TestColumnTypes(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		os.RemoveAll(tempDir)
	}()

	db, err := sql.Open("sqlite", filepath.Join(tempDir, "test.db"))
	if err != nil {
		t.Fatalf("test.db open fail: %v", err)
	}

	defer db.Close()

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS `userinfo` (`uid` INTEGER PRIMARY KEY AUTOINCREMENT,`username` VARCHAR(64) NULL, `departname` VARCHAR(64) NULL, `created` DATE NULL);")
	if err != nil {
		t.Fatal(err)
	}

	insertStatement := `INSERT INTO userinfo(username, departname, created) values("astaxie", "研发部门", "2012-12-09")`
	_, err = db.Exec(insertStatement)
	if err != nil {
		t.Fatal(err)
	}

	rows2, err := db.Query("SELECT * FROM userinfo")
	if err != nil {
		t.Fatal(err)
	}
	rows2.Next() // trigger statement execution
	defer rows2.Close()

	columnTypes, err := rows2.ColumnTypes()
	if err != nil {
		t.Fatal(err)
	}

	var b strings.Builder
	for index, value := range columnTypes {
		precision, scale, precisionOk := value.DecimalSize()
		length, lengthOk := value.Length()
		nullable, nullableOk := value.Nullable()
		fmt.Fprintf(&b, "Col %d: DatabaseTypeName %q, DecimalSize %v %v %v, Length %v %v, Name %q, Nullable %v %v, ScanType %q\n",
			index,
			value.DatabaseTypeName(),
			precision, scale, precisionOk,
			length, lengthOk,
			value.Name(),
			nullable, nullableOk,
			value.ScanType(),
		)
	}
	if err := rows2.Err(); err != nil {
		t.Fatal(err)
	}

	if g, e := b.String(), `Col 0: DatabaseTypeName "INTEGER", DecimalSize 0 0 false, Length 0 false, Name "uid", Nullable true true, ScanType "int64"
Col 1: DatabaseTypeName "VARCHAR(64)", DecimalSize 0 0 false, Length 9223372036854775807 true, Name "username", Nullable true true, ScanType "string"
Col 2: DatabaseTypeName "VARCHAR(64)", DecimalSize 0 0 false, Length 9223372036854775807 true, Name "departname", Nullable true true, ScanType "string"
Col 3: DatabaseTypeName "DATE", DecimalSize 0 0 false, Length 9223372036854775807 true, Name "created", Nullable true true, ScanType "time.Time"
`; g != e {
		t.Fatalf("---- got\n%s\n----expected\n%s", g, e)
	}
	t.Log(b.String())
}

// https://gitlab.com/cznic/sqlite/-/issues/32
func TestColumnTypesNoRows(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		os.RemoveAll(tempDir)
	}()

	db, err := sql.Open("sqlite", filepath.Join(tempDir, "test.db"))
	if err != nil {
		t.Fatalf("test.db open fail: %v", err)
	}

	defer db.Close()

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS `userinfo` (`uid` INTEGER PRIMARY KEY AUTOINCREMENT,`username` VARCHAR(64) NULL, `departname` VARCHAR(64) NULL, `created` DATE NULL);")
	if err != nil {
		t.Fatal(err)
	}

	rows2, err := db.Query("SELECT * FROM userinfo")
	if err != nil {
		t.Fatal(err)
	}
	defer rows2.Close()

	columnTypes, err := rows2.ColumnTypes()
	if err != nil {
		t.Fatal(err)
	}

	var b strings.Builder
	for index, value := range columnTypes {
		precision, scale, precisionOk := value.DecimalSize()
		length, lengthOk := value.Length()
		nullable, nullableOk := value.Nullable()
		fmt.Fprintf(&b, "Col %d: DatabaseTypeName %q, DecimalSize %v %v %v, Length %v %v, Name %q, Nullable %v %v, ScanType %q\n",
			index,
			value.DatabaseTypeName(),
			precision, scale, precisionOk,
			length, lengthOk,
			value.Name(),
			nullable, nullableOk,
			value.ScanType(),
		)
	}
	if err := rows2.Err(); err != nil {
		t.Fatal(err)
	}

	if g, e := b.String(), `Col 0: DatabaseTypeName "INTEGER", DecimalSize 0 0 false, Length 0 false, Name "uid", Nullable true true, ScanType %!q(<nil>)
Col 1: DatabaseTypeName "VARCHAR(64)", DecimalSize 0 0 false, Length 0 false, Name "username", Nullable true true, ScanType %!q(<nil>)
Col 2: DatabaseTypeName "VARCHAR(64)", DecimalSize 0 0 false, Length 0 false, Name "departname", Nullable true true, ScanType %!q(<nil>)
Col 3: DatabaseTypeName "DATE", DecimalSize 0 0 false, Length 0 false, Name "created", Nullable true true, ScanType %!q(<nil>)
`; g != e {
		t.Fatalf("---- got\n%s\n----expected\n%s", g, e)
	}
	t.Log(b.String())
}

// https://gitlab.com/cznic/sqlite/-/issues/35
func TestTime(t *testing.T) {
	types := []string{
		"DATE",
		"DATETIME",
		"Date",
		"DateTime",
		"TIMESTAMP",
		"TimeStamp",
		"date",
		"datetime",
		"timestamp",
	}
	db, err := sql.Open(driverName, "file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		db.Close()
	}()

	for _, typ := range types {
		if _, err := db.Exec(fmt.Sprintf(`
		drop table if exists mg;
		create table mg (applied_at %s);
		`, typ)); err != nil {
			t.Fatal(err)
		}

		now := time.Now()
		_, err = db.Exec(`INSERT INTO mg (applied_at) VALUES (?)`, &now)
		if err != nil {
			t.Fatal(err)
		}

		var appliedAt time.Time
		err = db.QueryRow("SELECT applied_at FROM mg").Scan(&appliedAt)
		if err != nil {
			t.Fatal(err)
		}

		if g, e := appliedAt, now; !g.Equal(e) {
			t.Fatal(g, e)
		}
	}
}

// https://gitlab.com/cznic/sqlite/-/issues/46
func TestTimeScan(t *testing.T) {
	ref := time.Date(2021, 1, 2, 16, 39, 17, 123456789, time.UTC)

	cases := []struct {
		s string
		w time.Time
	}{
		{s: "2021-01-02 12:39:17 -0400 ADT m=+00000", w: ref.Truncate(time.Second)},
		{s: "2021-01-02 16:39:17 +0000 UTC m=+0.000000001", w: ref.Truncate(time.Second)},
		{s: "2021-01-02 12:39:17.123456 -0400 ADT m=+00000", w: ref.Truncate(time.Microsecond)},
		{s: "2021-01-02 16:39:17.123456 +0000 UTC m=+0.000000001", w: ref.Truncate(time.Microsecond)},
		{s: "2021-01-02 16:39:17Z", w: ref.Truncate(time.Second)},
		{s: "2021-01-02 16:39:17+00:00", w: ref.Truncate(time.Second)},
		{s: "2021-01-02T16:39:17.123456+00:00", w: ref.Truncate(time.Microsecond)},
		{s: "2021-01-02 16:39:17.123456+00:00", w: ref.Truncate(time.Microsecond)},
		{s: "2021-01-02 16:39:17.123456Z", w: ref.Truncate(time.Microsecond)},
		{s: "2021-01-02 12:39:17-04:00", w: ref.Truncate(time.Second)},
		{s: "2021-01-02 16:39:17", w: ref.Truncate(time.Second)},
		{s: "2021-01-02T16:39:17", w: ref.Truncate(time.Second)},
		{s: "2021-01-02 16:39", w: ref.Truncate(time.Minute)},
		{s: "2021-01-02T16:39", w: ref.Truncate(time.Minute)},
		{s: "2021-01-02", w: ref.Truncate(24 * time.Hour)},
	}

	db, err := sql.Open(driverName, "file::memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	for _, colType := range []string{"DATE", "DATETIME", "TIMESTAMP"} {
		for _, tc := range cases {
			if _, err := db.Exec("drop table if exists x; create table x (y " + colType + ")"); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec("insert into x (y) values (?)", tc.s); err != nil {
				t.Fatal(err)
			}

			var got time.Time
			if err := db.QueryRow("select y from x").Scan(&got); err != nil {
				t.Fatal(err)
			}
			if !got.Equal(tc.w) {
				t.Errorf("scan(%q as %q) = %s, want %s", tc.s, colType, got, tc.w)
			}
		}
	}
}

// https://gitlab.com/cznic/sqlite/-/issues/49
func TestTimeLocaltime(t *testing.T) {
	db, err := sql.Open(driverName, "file::memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec("select datetime('now', 'localtime')"); err != nil {
		t.Fatal(err)
	}
}

func TestTimeFormat(t *testing.T) {
	ref := time.Date(2021, 1, 2, 16, 39, 17, 123456789, time.UTC)

	cases := []struct {
		f string
		w string
	}{
		{f: "", w: "2021-01-02 16:39:17.123456789+00:00"},
		{f: "sqlite", w: "2021-01-02 16:39:17.123456789+00:00"},
	}
	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			dsn := "file::memory:"
			if c.f != "" {
				q := make(url.Values)
				q.Set("_time_format", c.f)
				dsn += "?" + q.Encode()
			}
			db, err := sql.Open(driverName, dsn)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()

			if _, err := db.Exec("drop table if exists x; create table x (y text)"); err != nil {
				t.Fatal(err)
			}

			if _, err := db.Exec(`insert into x values (?)`, ref); err != nil {
				t.Fatal(err)
			}

			var got string
			if err := db.QueryRow(`select y from x`).Scan(&got); err != nil {
				t.Fatal(err)
			}

			if got != c.w {
				t.Fatal(got, c.w)
			}
		})
	}
}

func TestTimeFormatBad(t *testing.T) {
	db, err := sql.Open(driverName, "file::memory:?_time_format=bogus")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Error doesn't appear until a connection is opened.
	_, err = db.Exec("select 1")
	if err == nil {
		t.Fatal("wanted error")
	}

	want := `unknown _time_format "bogus"`
	if got := err.Error(); got != want {
		t.Fatalf("got error %q, want %q", got, want)
	}
}

// https://sqlite.org/lang_expr.html#varparam
// https://gitlab.com/cznic/sqlite/-/issues/42
func TestBinding(t *testing.T) {
	t.Run("DB", func(t *testing.T) {
		testBinding(t, func(db *sql.DB, query string, args ...interface{}) (*sql.Row, func()) {
			return db.QueryRow(query, args...), func() {}
		})
	})

	t.Run("Prepare", func(t *testing.T) {
		testBinding(t, func(db *sql.DB, query string, args ...interface{}) (*sql.Row, func()) {
			stmt, err := db.Prepare(query)
			if err != nil {
				t.Fatal(err)
			}
			return stmt.QueryRow(args...), func() { stmt.Close() }
		})
	})
}

func testBinding(t *testing.T, query func(db *sql.DB, query string, args ...interface{}) (*sql.Row, func())) {
	db, err := sql.Open(driverName, "file::memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	for _, tc := range []struct {
		q  string
		in []interface{}
		w  []int
	}{
		{
			q:  "?, ?, ?",
			in: []interface{}{1, 2, 3},
			w:  []int{1, 2, 3},
		},
		{
			q:  "?1, ?2, ?3",
			in: []interface{}{1, 2, 3},
			w:  []int{1, 2, 3},
		},
		{
			q:  "?1, ?, ?3",
			in: []interface{}{1, 2, 3},
			w:  []int{1, 2, 3},
		},
		{
			q:  "?3, ?2, ?1",
			in: []interface{}{1, 2, 3},
			w:  []int{3, 2, 1},
		},
		{
			q:  "?1, ?1, ?2",
			in: []interface{}{1, 2},
			w:  []int{1, 1, 2},
		},
		{
			q:  ":one, :two, :three",
			in: []interface{}{sql.Named("one", 1), sql.Named("two", 2), sql.Named("three", 3)},
			w:  []int{1, 2, 3},
		},
		{
			q:  ":one, :one, :two",
			in: []interface{}{sql.Named("one", 1), sql.Named("two", 2)},
			w:  []int{1, 1, 2},
		},
		{
			q:  "@one, @two, @three",
			in: []interface{}{sql.Named("one", 1), sql.Named("two", 2), sql.Named("three", 3)},
			w:  []int{1, 2, 3},
		},
		{
			q:  "@one, @one, @two",
			in: []interface{}{sql.Named("one", 1), sql.Named("two", 2)},
			w:  []int{1, 1, 2},
		},
		{
			q:  "$one, $two, $three",
			in: []interface{}{sql.Named("one", 1), sql.Named("two", 2), sql.Named("three", 3)},
			w:  []int{1, 2, 3},
		},
		{
			// A common usage that should technically require sql.Named but
			// does not.
			q:  "$1, $2, $3",
			in: []interface{}{1, 2, 3},
			w:  []int{1, 2, 3},
		},
		{
			q:  "$one, $one, $two",
			in: []interface{}{sql.Named("one", 1), sql.Named("two", 2)},
			w:  []int{1, 1, 2},
		},
		{
			q:  ":one, @one, $one",
			in: []interface{}{sql.Named("one", 1)},
			w:  []int{1, 1, 1},
		},
	} {
		got := make([]int, len(tc.w))
		ptrs := make([]interface{}, len(got))
		for i := range got {
			ptrs[i] = &got[i]
		}

		row, cleanup := query(db, "select "+tc.q, tc.in...)
		defer cleanup()

		if err := row.Scan(ptrs...); err != nil {
			t.Errorf("query(%q, %+v) = %s", tc.q, tc.in, err)
			continue
		}

		if !reflect.DeepEqual(got, tc.w) {
			t.Errorf("query(%q, %+v) = %#+v, want %#+v", tc.q, tc.in, got, tc.w)
		}
	}
}

func TestBindingError(t *testing.T) {
	t.Run("DB", func(t *testing.T) {
		testBindingError(t, func(db *sql.DB, query string, args ...interface{}) (*sql.Row, func()) {
			return db.QueryRow(query, args...), func() {}
		})
	})

	t.Run("Prepare", func(t *testing.T) {
		testBindingError(t, func(db *sql.DB, query string, args ...interface{}) (*sql.Row, func()) {
			stmt, err := db.Prepare(query)
			if err != nil {
				t.Fatal(err)
			}
			return stmt.QueryRow(args...), func() { stmt.Close() }
		})
	})
}

func testBindingError(t *testing.T, query func(db *sql.DB, query string, args ...interface{}) (*sql.Row, func())) {
	db, err := sql.Open(driverName, "file::memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	for _, tc := range []struct {
		q  string
		in []interface{}
	}{
		{
			q:  "?",
			in: []interface{}{},
		},
		{
			q:  "?500, ?",
			in: []interface{}{1, 2},
		},
		{
			q:  ":one",
			in: []interface{}{1},
		},
		{
			q:  "@one",
			in: []interface{}{1},
		},
		{
			q:  "$one",
			in: []interface{}{1},
		},
	} {
		got := make([]int, 2)
		ptrs := make([]interface{}, len(got))
		for i := range got {
			ptrs[i] = &got[i]
		}

		row, cleanup := query(db, "select "+tc.q, tc.in...)
		defer cleanup()

		err := row.Scan(ptrs...)
		if err == nil || (!strings.Contains(err.Error(), "missing argument with index") && !strings.Contains(err.Error(), "missing named argument")) {
			t.Errorf("query(%q, %+v) unexpected error %+v", tc.q, tc.in, err)
		}
	}
}

// https://gitlab.com/cznic/sqlite/-/issues/51
func TestIssue51(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		os.RemoveAll(tempDir)
	}()

	fn := filepath.Join(tempDir, "test_issue51.db")
	db, err := sql.Open(driverName, fn)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		db.Close()
	}()

	if _, err := db.Exec(`
CREATE TABLE fileHash (
	"hash" TEXT NOT NULL PRIMARY KEY,
	"filename" TEXT,
	"lastChecked" INTEGER
 );`); err != nil {
		t.Fatal(err)
	}

	t0 := time.Now()
	n := 0
	for time.Since(t0) < time.Minute {
		hash := randomString()
		if _, err = lookupHash(fn, hash); err != nil {
			t.Fatal(err)
		}

		if err = saveHash(fn, hash, hash+".temp"); err != nil {
			t.Error(err)
			break
		}
		n++
	}
	t.Logf("cycles: %v", n)
	row := db.QueryRow("select count(*) from fileHash")
	if err := row.Scan(&n); err != nil {
		t.Fatal(err)
	}

	t.Logf("DB records: %v", n)
}

func saveHash(dbFile string, hash string, fileName string) (err error) {
	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return fmt.Errorf("could not open database: %v", err)
	}

	defer func() {
		if err2 := db.Close(); err2 != nil && err == nil {
			err = fmt.Errorf("could not close the database: %s", err2)
		}
	}()

	query := `INSERT OR REPLACE INTO fileHash(hash, fileName, lastChecked)
			VALUES(?, ?, ?);`
	rows, err := executeSQL(db, query, hash, fileName, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("error saving hash to database: %v", err)
	}
	defer rows.Close()

	return nil
}

func executeSQL(db *sql.DB, query string, values ...interface{}) (*sql.Rows, error) {
	statement, err := db.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("could not prepare statement: %v", err)
	}
	defer statement.Close()

	return statement.Query(values...)
}

func lookupHash(dbFile string, hash string) (ok bool, err error) {
	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return false, fmt.Errorf("could not open database: %n", err)
	}

	defer func() {
		if err2 := db.Close(); err2 != nil && err == nil {
			err = fmt.Errorf("could not close the database: %v", err2)
		}
	}()

	query := `SELECT hash, fileName, lastChecked
				FROM fileHash
				WHERE hash=?;`
	rows, err := executeSQL(db, query, hash)
	if err != nil {
		return false, fmt.Errorf("error checking database for hash: %n", err)
	}

	defer func() {
		if err2 := rows.Close(); err2 != nil && err == nil {
			err = fmt.Errorf("could not close DB rows: %v", err2)
		}
	}()

	var (
		dbHash      string
		fileName    string
		lastChecked int64
	)
	for rows.Next() {
		err = rows.Scan(&dbHash, &fileName, &lastChecked)
		if err != nil {
			return false, fmt.Errorf("could not read DB row: %v", err)
		}
	}
	return false, rows.Err()
}

func randomString() string {
	b := make([]byte, 32)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// https://gitlab.com/cznic/sqlite/-/issues/53
func TestIssue53(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		os.RemoveAll(tempDir)
	}()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	defer os.Chdir(wd)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	const fn = "testissue53.sqlite"

	db, err := sql.Open(driverName, fn)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		db.Close()
	}()

	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS loginst (
     instid INTEGER PRIMARY KEY,
     name   VARCHAR UNIQUE
);
`); err != nil {
		t.Fatal(err)
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 5000; i++ {
		x := fmt.Sprintf("foo%d", i)
		var id int
		if err := tx.QueryRow("INSERT OR IGNORE INTO loginst (name) VALUES (?); SELECT instid FROM loginst WHERE name = ?", x, x).Scan(&id); err != nil {
			t.Fatal(err)
		}
	}

}

// https://gitlab.com/cznic/sqlite/-/issues/37
func TestPersistPragma(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		os.RemoveAll(tempDir)
	}()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	defer os.Chdir(wd)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	pragmas := []pragmaCfg{
		{"foreign_keys", "on", int64(1)},
		{"analysis_limit", "1000", int64(1000)},
		{"application_id", "214", int64(214)},
		{"encoding", "'UTF-16le'", "UTF-16le"}}

	if err := testPragmas("testpersistpragma.sqlite", "testpersistpragma.sqlite", pragmas); err != nil {
		t.Fatal(err)
	}
	if err := testPragmas("file::memory:", "", pragmas); err != nil {
		t.Fatal(err)
	}
	if err := testPragmas(":memory:", "", pragmas); err != nil {
		t.Fatal(err)
	}
}

type pragmaCfg struct {
	name     string
	value    string
	expected interface{}
}

func testPragmas(name, diskFile string, pragmas []pragmaCfg) error {
	if diskFile != "" {
		os.Remove(diskFile)
	}

	q := url.Values{}
	for _, pragma := range pragmas {
		q.Add("_pragma", pragma.name+"="+pragma.value)
	}

	dsn := name + "?" + q.Encode()
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return err
	}

	db.SetMaxOpenConns(1)

	if err := checkPragmas(db, pragmas); err != nil {
		return err
	}

	c, err := db.Conn(context.Background())
	if err != nil {
		return err
	}

	// Kill the connection to spawn a new one. Pragma configs should persist
	c.Raw(func(interface{}) error { return driver.ErrBadConn })

	if err := checkPragmas(db, pragmas); err != nil {
		return err
	}

	if diskFile == "" {
		// Make sure in memory databases aren't being written to disk
		return testInMemory(db)
	}

	return nil
}

func checkPragmas(db *sql.DB, pragmas []pragmaCfg) error {
	for _, pragma := range pragmas {
		row := db.QueryRow(`PRAGMA ` + pragma.name)

		var result interface{}
		if err := row.Scan(&result); err != nil {
			return err
		}
		if result != pragma.expected {
			return fmt.Errorf("expected PRAGMA %s to return %v but got %v", pragma.name, pragma.expected, result)
		}
	}
	return nil
}

func TestInMemory(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		os.RemoveAll(tempDir)
	}()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	defer os.Chdir(wd)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	if err := testMemoryPath(":memory:"); err != nil {
		t.Fatal(err)
	}
	if err := testMemoryPath("file::memory:"); err != nil {
		t.Fatal(err)
	}

	// This parameter should be ignored
	q := url.Values{}
	q.Add("mode", "readonly")
	if err := testMemoryPath(":memory:?" + q.Encode()); err != nil {
		t.Fatal(err)
	}
}

func testMemoryPath(mPath string) error {
	db, err := sql.Open(driverName, mPath)
	if err != nil {
		return err
	}
	defer db.Close()

	return testInMemory(db)
}

func testInMemory(db *sql.DB) error {
	_, err := db.Exec(`
	create table in_memory_test(i int, f double);
	insert into in_memory_test values(12, 3.14);
	`)
	if err != nil {
		return err
	}

	dirEntries, err := os.ReadDir("./")
	if err != nil {
		return err
	}

	for _, dirEntry := range dirEntries {
		if strings.Contains(dirEntry.Name(), "memory") {
			return fmt.Errorf("file was created for in memory database")
		}
	}

	return nil
}

func emptyDir(s string) error {
	m, err := filepath.Glob(filepath.FromSlash(s + "/*"))
	if err != nil {
		return err
	}

	for _, v := range m {
		fi, err := os.Stat(v)
		if err != nil {
			return err
		}

		switch {
		case fi.IsDir():
			if err = os.RemoveAll(v); err != nil {
				return err
			}
		default:
			if err = os.Remove(v); err != nil {
				return err
			}
		}
	}
	return nil
}

// https://gitlab.com/cznic/sqlite/-/issues/70
func TestIssue70(t *testing.T) {
	db, err := sql.Open(driverName, "file::memory:")
	if _, err = db.Exec(`create table t (foo)`); err != nil {
		t.Fatalf("create: %v", err)
	}

	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("conn close: %v", err)
		}
	}()

	r, err := db.Query("select * from t")
	if err != nil {
		t.Errorf("select a: %v", err)
		return
	}

	if err := r.Close(); err != nil {
		t.Errorf("rows close: %v", err)
		return
	}

	if _, err := db.Query("select * from t"); err != nil {
		t.Errorf("select b: %v", err)
	}
}

// https://gitlab.com/cznic/sqlite/-/issues/66
func TestIssue66(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		os.RemoveAll(tempDir)
	}()

	fn := filepath.Join(tempDir, "testissue66.db")
	db, err := sql.Open(driverName, fn)

	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("conn close: %v", err)
		}
	}()

	if _, err = db.Exec(`CREATE TABLE IF NOT EXISTS verdictcache (sha1 text);`); err != nil {
		t.Fatalf("create: %v", err)
	}

	// ab
	// 00	ok
	// 01	ok
	// 10	ok
	// 11	hangs with old implementation of conn.step().

	// a
	if _, err = db.Exec("INSERT OR REPLACE INTO verdictcache (sha1) VALUES ($1)", "a"); err != nil {
		t.Fatalf("insert: %v", err)
	}

	// b
	if _, err := db.Query("SELECT * FROM verdictcache WHERE sha1=$1", "a"); err != nil {
		t.Fatalf("select: %v", err)
	}

	// c
	if _, err = db.Exec("INSERT OR REPLACE INTO verdictcache (sha1) VALUES ($1)", "b"); err != nil {

		// https://www.sqlite.org/rescode.html#busy
		// ----------------------------------------------------------------------------
		// The SQLITE_BUSY result code indicates that the database file could not be
		// written (or in some cases read) because of concurrent activity by some other
		// database connection, usually a database connection in a separate process.
		// ----------------------------------------------------------------------------
		//
		// The SQLITE_BUSY error is _expected_.
		//
		// According to the above, performing c after b's result was not yet
		// consumed/closed is not possible. Mattn's driver seems to resort to
		// autoclosing the driver.Rows returned by b in this situation, but I don't
		// think that's correct (jnml).

		t.Logf("insert 2: %v", err)
		if !strings.Contains(err.Error(), "database is locked (5) (SQLITE_BUSY)") {
			t.Fatalf("insert 2: %v", err)
		}
	}
}

// https://gitlab.com/cznic/sqlite/-/issues/65
func TestIssue65(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		os.RemoveAll(tempDir)
	}()

	db, err := sql.Open("sqlite", filepath.Join(tempDir, "testissue65.sqlite"))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	testIssue65(t, db, true)

	// go1.20rc1, linux/ppc64le VM
	// 10000 FAIL
	// 20000 PASS, FAIL
	// 40000 FAIL
	// 80000 PASS, PASS
	if db, err = sql.Open("sqlite", filepath.Join(tempDir, "testissue65b.sqlite")+"?_pragma=busy_timeout%3d80000"); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	testIssue65(t, db, false)
}

func testIssue65(t *testing.T, db *sql.DB, canFail bool) {
	defer db.Close()

	ctx := context.Background()

	if _, err := db.Exec("CREATE TABLE foo (department INTEGER, profits INTEGER)"); err != nil {
		t.Fatal("Failed to create table:", err)
	}

	if _, err := db.Exec("INSERT INTO foo VALUES (1, 10), (1, 20), (1, 45), (2, 42), (2, 115)"); err != nil {
		t.Fatal("Failed to insert records:", err)
	}

	readFunc := func(ctx context.Context) error {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("read error: %v", err)
		}

		defer tx.Rollback()

		var dept, count int64
		if err := tx.QueryRowContext(ctx, "SELECT department, COUNT(*) FROM foo GROUP BY department").Scan(
			&dept,
			&count,
		); err != nil {
			return fmt.Errorf("read error: %v", err)
		}

		return nil
	}

	writeFunc := func(ctx context.Context) error {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("write error: %v", err)
		}

		defer tx.Rollback()

		if _, err := tx.ExecContext(
			ctx,
			"INSERT INTO foo(department, profits) VALUES (@department, @profits)",
			sql.Named("department", rand.Int()),
			sql.Named("profits", rand.Int()),
		); err != nil {
			return fmt.Errorf("write error: %v", err)
		}

		return tx.Commit()
	}

	var wg sync.WaitGroup
	wg.Add(2)

	const cycles = 100

	errCh := make(chan error, 2)

	go func() {
		defer wg.Done()

		for i := 0; i < cycles; i++ {
			if err := readFunc(ctx); err != nil {
				err = fmt.Errorf("readFunc(%v): %v", canFail, err)
				t.Log(err)
				if !canFail {
					errCh <- err
				}
				return
			}
		}
	}()

	go func() {
		defer wg.Done()

		for i := 0; i < cycles; i++ {
			if err := writeFunc(ctx); err != nil {
				err = fmt.Errorf("writeFunc(%v): %v", canFail, err)
				t.Log(err)
				if !canFail {
					errCh <- err
				}
				return
			}
		}
	}()

	wg.Wait()
	for {
		select {
		case err := <-errCh:
			t.Error(err)
		default:
			return
		}
	}
}

// https://gitlab.com/cznic/sqlite/-/issues/73
func TestConstraintPrimaryKeyError(t *testing.T) {
	db, err := sql.Open(driverName, "file::memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS hash (hashval TEXT PRIMARY KEY NOT NULL)`)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec("INSERT INTO hash (hashval) VALUES (?)", "somehashval")
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec("INSERT INTO hash (hashval) VALUES (?)", "somehashval")
	if err == nil {
		t.Fatal("wanted error")
	}

	if errs, want := err.Error(), "constraint failed: UNIQUE constraint failed: hash.hashval (1555)"; errs != want {
		t.Fatalf("got error string %q, want %q", errs, want)
	}
}

func TestConstraintUniqueError(t *testing.T) {
	db, err := sql.Open(driverName, "file::memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS hash (hashval TEXT UNIQUE)`)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec("INSERT INTO hash (hashval) VALUES (?)", "somehashval")
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec("INSERT INTO hash (hashval) VALUES (?)", "somehashval")
	if err == nil {
		t.Fatal("wanted error")
	}

	if errs, want := err.Error(), "constraint failed: UNIQUE constraint failed: hash.hashval (2067)"; errs != want {
		t.Fatalf("got error string %q, want %q", errs, want)
	}
}

// https://gitlab.com/cznic/sqlite/-/issues/92
func TestBeginMode(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		os.RemoveAll(tempDir)
	}()

	tests := []struct {
		mode string
		want int32
	}{
		{"deferred", sqlite3.SQLITE_TXN_NONE},
		{"immediate", sqlite3.SQLITE_TXN_WRITE},
		// TODO: how to verify "exclusive" is working differently from immediate,
		// short of concurrently trying to open the database again? This is only
		// different in non-WAL journal modes.
		{"exclusive", sqlite3.SQLITE_TXN_WRITE},
	}

	for _, tt := range tests {
		tt := tt
		for _, jm := range []string{"delete", "wal"} {
			jm := jm
			t.Run(jm+"/"+tt.mode, func(t *testing.T) {
				// t.Parallel()

				qs := fmt.Sprintf("?_txlock=%s&_pragma=journal_mode(%s)", tt.mode, jm)
				db, err := sql.Open("sqlite", filepath.Join(tempDir, fmt.Sprintf("testbeginmode-%s.sqlite", tt.mode))+qs)
				if err != nil {
					t.Fatalf("Failed to open database: %v", err)
				}
				defer db.Close()
				connection, err := db.Conn(context.Background())
				if err != nil {
					t.Fatalf("Failed to open connection: %v", err)
				}

				tx, err := connection.BeginTx(context.Background(), nil)
				if err != nil {
					t.Fatalf("Failed to begin transaction: %v", err)
				}
				defer tx.Rollback()
				if err := connection.Raw(func(driverConn interface{}) error {
					p, err := libc.CString("main")
					if err != nil {
						return err
					}
					c := driverConn.(*conn)
					defer c.free(p)
					got := sqlite3.Xsqlite3_txn_state(c.tls, c.db, p)
					if got != tt.want {
						return fmt.Errorf("in mode %s, got txn state %d, want %d", tt.mode, got, tt.want)
					}
					return nil
				}); err != nil {
					t.Fatalf("Failed to check txn state: %v", err)
				}
			})
		}
	}
}

// https://gitlab.com/cznic/sqlite/-/issues/94
func TestCancelRace(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		os.RemoveAll(tempDir)
	}()

	db, err := sql.Open("sqlite", filepath.Join(tempDir, "testcancelrace.sqlite"))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	tests := []struct {
		name string
		f    func(context.Context, *sql.DB) error
	}{
		{
			"db.ExecContext",
			func(ctx context.Context, d *sql.DB) error {
				_, err := db.ExecContext(ctx, "select 1")
				return err
			},
		},
		{
			"db.QueryContext",
			func(ctx context.Context, d *sql.DB) error {
				_, err := db.QueryContext(ctx, "select 1")
				return err
			},
		},
		{
			"tx.ExecContext",
			func(ctx context.Context, d *sql.DB) error {
				tx, err := db.BeginTx(ctx, &sql.TxOptions{})
				if err != nil {
					return err
				}
				defer tx.Rollback()
				if _, err := tx.ExecContext(ctx, "select 1"); err != nil {
					return err
				}
				return tx.Rollback()
			},
		},
		{
			"tx.QueryContext",
			func(ctx context.Context, d *sql.DB) error {
				tx, err := db.BeginTx(ctx, &sql.TxOptions{})
				if err != nil {
					return err
				}
				defer tx.Rollback()
				if _, err := tx.QueryContext(ctx, "select 1"); err != nil {
					return err
				}
				return tx.Rollback()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// this is a race condition, so it's not guaranteed to fail on any given run,
			// but with a moderate number of iterations it will eventually catch it
			iterations := 100
			for i := 0; i < iterations; i++ {
				// none of these iterations should ever fail, because we never cancel their
				// context until after they complete
				ctx, cancel := context.WithCancel(context.Background())
				if err := tt.f(ctx, db); err != nil {
					t.Fatalf("Failed to run test query on iteration %d: %v", i, err)
				}
				cancel()
			}
		})
	}
}

//go:embed embed.db
var fs embed.FS

//go:embed embed2.db
var fs2 embed.FS

func TestVFS(t *testing.T) {
	fn, f, err := vfs.New(fs)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := f.Close(); err != nil {
			t.Error(err)
		}
	}()

	f2n, f2, err := vfs.New(fs2)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := f2.Close(); err != nil {
			t.Error(err)
		}
	}()

	db, err := sql.Open("sqlite", "file:embed.db?vfs="+fn)
	if err != nil {
		t.Fatal(err)
	}

	defer db.Close()

	db2, err := sql.Open("sqlite", "file:embed2.db?vfs="+f2n)
	if err != nil {
		t.Fatal(err)
	}

	defer db2.Close()

	rows, err := db.Query("select * from t order by i;")
	if err != nil {
		t.Fatal(err)
	}

	var a []int
	for rows.Next() {
		var i, j, k int
		if err := rows.Scan(&i, &j, &k); err != nil {
			t.Fatal(err)
		}

		a = append(a, i, j, k)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	t.Log(a)
	if g, e := fmt.Sprint(a), "[1 2 3 40 50 60]"; g != e {
		t.Fatalf("got %q, expected %q", g, e)
	}

	if rows, err = db2.Query("select * from u order by s;"); err != nil {
		t.Fatal(err)
	}

	var b []string
	for rows.Next() {
		var x, y string
		if err := rows.Scan(&x, &y); err != nil {
			t.Fatal(err)
		}

		b = append(b, x, y)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	t.Log(b)
	if g, e := fmt.Sprint(b), "[123 xyz abc def]"; g != e {
		t.Fatalf("got %q, expected %q", g, e)
	}
}

// y = 2^n, except for n < 0 y = 0.
func exp(n int) int {
	if n < 0 {
		return 0
	}

	return 1 << n
}

func BenchmarkConcurrent(b *testing.B) {
	benchmarkConcurrent(b, "sqlite", []string{"sql", "drv"})
}

func benchmarkConcurrent(b *testing.B, drv string, modes []string) {
	for _, mode := range modes {
		for _, measurement := range []string{"reads", "writes"} {
			for _, writers := range []int{0, 1, 10, 100, 100} {
				for _, readers := range []int{0, 1, 10, 100, 100} {
					if measurement == "reads" && readers == 0 || measurement == "writes" && writers == 0 {
						continue
					}

					tag := fmt.Sprintf("%s %s readers %d writers %d %s", mode, measurement, readers, writers, drv)
					b.Run(tag, func(b *testing.B) { c := &concurrentBenchmark{}; c.run(b, readers, writers, drv, measurement, mode) })
				}
			}
		}
	}
}

// The code for concurrentBenchmark is derived from/heavily inspired by
// original code available at
//
//	https://github.com/kalafut/go-sqlite-bench
//
// # MIT License
//
// # Copyright (c) 2022 Jim Kalafut
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
type concurrentBenchmark struct {
	b     *testing.B
	drv   string
	fn    string
	start chan struct{}
	stop  chan struct{}
	wg    sync.WaitGroup

	reads   int32
	records int32
	writes  int32
}

func (c *concurrentBenchmark) run(b *testing.B, readers, writers int, drv, measurement, mode string) {
	c.b = b
	c.drv = drv
	b.ReportAllocs()
	dir := b.TempDir()
	fn := filepath.Join(dir, "test.db")
	sqlite3.MutexCounters.Disable()
	sqlite3.MutexEnterCallers.Disable()
	c.makeDB(fn)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		c.start = make(chan struct{})
		c.stop = make(chan struct{})
		sqlite3.MutexCounters.Disable()
		sqlite3.MutexEnterCallers.Disable()
		c.makeReaders(readers, mode)
		c.makeWriters(writers, mode)
		sqlite3.MutexCounters.Clear()
		sqlite3.MutexCounters.Enable()
		sqlite3.MutexEnterCallers.Clear()
		//sqlite3.MutexEnterCallers.Enable()
		time.AfterFunc(time.Second, func() { close(c.stop) })
		b.StartTimer()
		close(c.start)
		c.wg.Wait()
	}
	switch measurement {
	case "reads":
		b.ReportMetric(float64(c.reads), "reads/s")
	case "writes":
		b.ReportMetric(float64(c.writes), "writes/s")
	}
	// b.Log(sqlite3.MutexCounters)
	// b.Log(sqlite3.MutexEnterCallers)
}

func (c *concurrentBenchmark) randString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(65 + rand.Intn(26))
	}
	return string(b)
}

func (c *concurrentBenchmark) mustExecSQL(db *sql.DB, sql string) {
	var err error
	for i := 0; i < 100; i++ {
		if _, err = db.Exec(sql); err != nil {
			if c.retry(err) {
				continue
			}

			c.b.Fatalf("%s: %v", sql, err)
		}

		return
	}
	c.b.Fatalf("%s: %v", sql, err)
}

func (c *concurrentBenchmark) mustExecDrv(db driver.Conn, sql string) {
	var err error
	for i := 0; i < 100; i++ {
		if _, err = db.(driver.Execer).Exec(sql, nil); err != nil {
			if c.retry(err) {
				continue
			}

			c.b.Fatalf("%s: %v", sql, err)
		}

		return
	}
	c.b.Fatalf("%s: %v", sql, err)
}

func (c *concurrentBenchmark) makeDB(fn string) {
	const quota = 1e6
	c.fn = fn
	db := c.makeSQLConn()

	defer db.Close()

	c.mustExecSQL(db, "CREATE TABLE foo (id INTEGER NOT NULL PRIMARY KEY, name TEXT)")
	tx, err := db.Begin()
	if err != nil {
		c.b.Fatal(err)
	}

	stmt, err := tx.Prepare("INSERT INTO FOO(name) VALUES($1)")
	if err != nil {
		c.b.Fatal(err)
	}

	for i := int32(0); i < quota; i++ {
		if _, err = stmt.Exec(c.randString(30)); err != nil {
			c.b.Fatal(err)
		}
	}

	if err := tx.Commit(); err != nil {
		c.b.Fatal(err)
	}

	c.records = quota

	// Warm the cache.
	rows, err := db.Query("SELECT * FROM foo")
	if err != nil {
		c.b.Fatal(err)
	}

	for rows.Next() {
		var id int
		var name string
		err = rows.Scan(&id, &name)
		if err != nil {
			c.b.Fatal(err)
		}
	}
}

func (c *concurrentBenchmark) makeSQLConn() *sql.DB {
	db, err := sql.Open(c.drv, c.fn)
	if err != nil {
		c.b.Fatal(err)
	}

	db.SetMaxOpenConns(0)
	c.mustExecSQL(db, "PRAGMA busy_timeout=10000")
	c.mustExecSQL(db, "PRAGMA synchronous=NORMAL")
	c.mustExecSQL(db, "PRAGMA journal_mode=WAL")
	return db
}

func (c *concurrentBenchmark) makeDrvConn() driver.Conn {
	db, err := sql.Open(c.drv, c.fn)
	if err != nil {
		c.b.Fatal(err)
	}

	drv := db.Driver()
	if err := db.Close(); err != nil {
		c.b.Fatal(err)
	}

	conn, err := drv.Open(c.fn)
	if err != nil {
		c.b.Fatal(err)
	}

	c.mustExecDrv(conn, "PRAGMA busy_timeout=10000")
	c.mustExecDrv(conn, "PRAGMA synchronous=NORMAL")
	c.mustExecDrv(conn, "PRAGMA journal_mode=WAL")
	return conn
}

func (c *concurrentBenchmark) retry(err error) bool {
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "lock") || strings.Contains(s, "busy")
}

func (c *concurrentBenchmark) makeReaders(n int, mode string) {
	var wait sync.WaitGroup
	wait.Add(n)
	c.wg.Add(n)
	for i := 0; i < n; i++ {
		switch mode {
		case "sql":
			go func() {
				db := c.makeSQLConn()

				defer func() {
					db.Close()
					c.wg.Done()
				}()

				wait.Done()
				<-c.start

				for i := 1; ; i++ {
					select {
					case <-c.stop:
						return
					default:
					}

					recs := atomic.LoadInt32(&c.records)
					id := recs * int32(i) % recs
					rows, err := db.Query("SELECT * FROM foo WHERE id=$1", id)
					if err != nil {
						if c.retry(err) {
							continue
						}

						c.b.Fatal(err)
					}

					for rows.Next() {
						var id int
						var name string
						err = rows.Scan(&id, &name)
						if err != nil {
							c.b.Fatal(err)
						}
					}
					if err := rows.Close(); err != nil {
						c.b.Fatal(err)
					}

					atomic.AddInt32(&c.reads, 1)
				}

			}()
		case "drv":
			go func() {
				conn := c.makeDrvConn()

				defer func() {
					conn.Close()
					c.wg.Done()
				}()

				q := conn.(driver.Queryer)
				wait.Done()
				<-c.start

				for i := 1; ; i++ {
					select {
					case <-c.stop:
						return
					default:
					}

					recs := atomic.LoadInt32(&c.records)
					id := recs * int32(i) % recs
					rows, err := q.Query("SELECT * FROM foo WHERE id=$1", []driver.Value{int64(id)})
					if err != nil {
						if c.retry(err) {
							continue
						}

						c.b.Fatal(err)
					}

					var dest [2]driver.Value
					for {
						if err := rows.Next(dest[:]); err != nil {
							if err != io.EOF {
								c.b.Fatal(err)
							}
							break
						}
					}

					if err := rows.Close(); err != nil {
						c.b.Fatal(err)
					}

					atomic.AddInt32(&c.reads, 1)
				}

			}()
		default:
			panic(todo(""))
		}
	}
	wait.Wait()
}

func (c *concurrentBenchmark) makeWriters(n int, mode string) {
	var wait sync.WaitGroup
	wait.Add(n)
	c.wg.Add(n)
	for i := 0; i < n; i++ {
		switch mode {
		case "sql":
			go func() {
				db := c.makeSQLConn()

				defer func() {
					db.Close()
					c.wg.Done()
				}()

				wait.Done()
				<-c.start

				for {
					select {
					case <-c.stop:
						return
					default:
					}

					if _, err := db.Exec("INSERT INTO FOO(name) VALUES($1)", c.randString(30)); err != nil {
						if c.retry(err) {
							continue
						}

						c.b.Fatal(err)
					}

					atomic.AddInt32(&c.records, 1)
					atomic.AddInt32(&c.writes, 1)
				}

			}()
		case "drv":
			go func() {
				conn := c.makeDrvConn()

				defer func() {
					conn.Close()
					c.wg.Done()
				}()

				e := conn.(driver.Execer)
				wait.Done()
				<-c.start

				for {
					select {
					case <-c.stop:
						return
					default:
					}

					if _, err := e.Exec("INSERT INTO FOO(name) VALUES($1)", []driver.Value{c.randString(30)}); err != nil {
						if c.retry(err) {
							continue
						}

						c.b.Fatal(err)
					}

					atomic.AddInt32(&c.records, 1)
					atomic.AddInt32(&c.writes, 1)
				}

			}()
		default:
			panic(todo(""))
		}
	}
	wait.Wait()
}
