// Copyright 2017 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlite

import (
	"bytes"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/cznic/virtual"
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
	// -tags virtual.profile
	profileFunctions    = flag.Bool("profile_functions", false, "")
	profileInstructions = flag.Bool("profile_instructions", false, "")
	profileLines        = flag.Bool("profile_lines", false, "")
	profileRate         = flag.Int("profile_rate", 1000, "")
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

func profile(t testing.TB, d time.Duration, w io.Writer, format string, arg ...interface{}) {
	rate := vm.ProfileRate
	if rate == 0 {
		rate = 1
	}
	if len(vm.ProfileFunctions) != 0 {
		fmt.Fprintf(w, format, arg...)
		type u struct {
			virtual.PCInfo
			n int
		}
		var s int64
		var a []u
		var wi int
		for k, v := range vm.ProfileFunctions {
			a = append(a, u{k, v})
			s += int64(v)
			if n := len(k.Name.String()); n > wi {
				wi = n
			}
		}
		sort.Slice(a, func(i, j int) bool {
			if a[i].n < a[j].n {
				return true
			}

			if a[i].n > a[j].n {
				return false
			}

			return a[i].Name < a[j].Name
		})
		fmt.Fprintf(w, "---- Profile functions, %.3f MIPS\n", float64(s)/1e6*float64(rate)*float64(time.Second)/float64(d))
		var c int64
		for i := len(a) - 1; i >= 0; i-- {
			c += int64(a[i].n)
			fmt.Fprintf(
				w,
				"%*v\t%10v%10.2f%%%10.2f%%\n",
				-wi, a[i].Name, a[i].n,
				100*float64(a[i].n)/float64(s),
				100*float64(c)/float64(s),
			)
		}
	}
	if len(vm.ProfileLines) != 0 {
		fmt.Fprintf(w, format, arg...)
		type u struct {
			virtual.PCInfo
			n int
		}
		var s int64
		var a []u
		for k, v := range vm.ProfileLines {
			a = append(a, u{k, v})
			s += int64(v)
		}
		sort.Slice(a, func(i, j int) bool {
			if a[i].n < a[j].n {
				return true
			}

			if a[i].n > a[j].n {
				return false
			}

			if a[i].Name < a[j].Name {
				return true
			}

			if a[i].Name > a[j].Name {
				return false
			}

			return a[i].Line < a[j].Line
		})
		fmt.Fprintf(w, "---- Profile lines, %.3f MIPS\n", float64(s)/1e6*float64(rate)*float64(time.Second)/float64(d))
		var c int64
		for i := len(a) - 1; i >= 0; i-- {
			c += int64(a[i].n)
			fmt.Fprintf(
				w,
				"%v:%v:\t%10v%10.2f%%%10.2f%%\n",
				a[i].Name, a[i].Line, a[i].n,
				100*float64(a[i].n)/float64(s),
				100*float64(c)/float64(s),
			)
		}
	}
	if len(vm.ProfileInstructions) != 0 {
		fmt.Fprintf(w, format, arg...)
		type u struct {
			virtual.Opcode
			n int
		}
		var s int64
		var a []u
		var wi int
		for k, v := range vm.ProfileInstructions {
			a = append(a, u{k, v})
			s += int64(v)
			if n := len(k.String()); n > wi {
				wi = n
			}
		}
		sort.Slice(a, func(i, j int) bool {
			if a[i].n < a[j].n {
				return true
			}

			if a[i].n > a[j].n {
				return false
			}

			return a[i].Opcode < a[j].Opcode
		})
		fmt.Fprintf(w, "---- Profile instructions, %.3f MIPS\n", float64(s)/1e6*float64(rate)*float64(time.Second)/float64(d))
		var c int64
		for i := len(a) - 1; i >= 0; i-- {
			c += int64(a[i].n)
			fmt.Fprintf(
				w,
				"%*s%10v%10.2f%%\t%10.2f%%\n",
				-wi, a[i].Opcode, a[i].n,
				100*float64(a[i].n)/float64(s),
				100*float64(c)/float64(s),
			)
		}
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

	if *profileFunctions {
		vm.ProfileFunctions = map[virtual.PCInfo]int{}
	}
	if *profileLines {
		vm.ProfileLines = map[virtual.PCInfo]int{}
	}
	if *profileInstructions {
		vm.ProfileInstructions = map[virtual.Opcode]int{}
	}
	vm.ProfileRate = int(*profileRate)
	t0 := time.Now()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := s.Exec(int64(i)); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	d := time.Since(t0)
	if _, err := db.Exec(`commit;`); err != nil {
		b.Fatal(err)
	}

	profile(b, d, os.Stderr, "==== BenchmarkInsertMemory b.N %v\n", b.N)
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

	if *profileFunctions {
		vm.ProfileFunctions = map[virtual.PCInfo]int{}
	}
	if *profileLines {
		vm.ProfileLines = map[virtual.PCInfo]int{}
	}
	if *profileInstructions {
		vm.ProfileInstructions = map[virtual.Opcode]int{}
	}
	vm.ProfileRate = int(*profileRate)
	t0 := time.Now()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !r.Next() {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	d := time.Since(t0)
	profile(b, d, os.Stderr, "==== BenchmarkNextMemory b.N %v\n", b.N)
}
