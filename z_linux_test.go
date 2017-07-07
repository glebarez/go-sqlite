// Copyright 2017 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlite

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

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

	nm := filepath.FromSlash("./mptest")
	for _, v := range m {
		os.Remove("db")
		out, err := exec.Command(nm, "db", v).CombinedOutput()
		t.Logf("%s", out)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestThread1(t *testing.T) {
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

	out, err := exec.Command("./threadtest1", "10").CombinedOutput()
	t.Logf("%s", out)
	if err != nil {
		t.Fatal(err)
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

func TestThread3(t *testing.T) {
	t.Log("TODO")
	return //TODO-

	//TODO sqlite3.c:142510: sqlite3_wal_hook(db, sqlite3WalDefaultHook, SQLITE_INT_TO_PTR(nFrame)); -> fatal error: bad pointer in write barrier
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

	s := []string{"build", "-o", "threadtest3"}
	if *memTrace {
		s = append(s, "-tags", "memory.trace", "-race")
	}
	if out, err := exec.Command("go", append(s, "github.com/cznic/sqlite/internal/threadtest3")...).CombinedOutput(); err != nil {
		t.Fatalf("go build mptest: %s\n%s", err, out)
	}

	for _, opts := range [][]string{
		{"walthread1"},
		{"walthread2"},
		{"walthread3"},
		{"walthread4"},
		{"walthread5"},
		{"cgt_pager_1"},
		{"dynamic_triggers"},
		{"checkpoint_starvation_1"},
		{"checkpoint_starvation_2"},
		{"create_drop_index_1"},
		{"lookaside1"},
		{"vacuum1"},
		{"stress1"},
		{"stress2"},
	} {
		out, err := exec.Command("./threadtest3", opts...).CombinedOutput()
		t.Logf("%v\n%s", opts, out)
		if err != nil {
			t.Fatal(err)
		}

		if bytes.Contains(out, []byte("fault address")) ||
			bytes.Contains(out, []byte("data race")) ||
			bytes.Contains(out, []byte("RACE")) {
			t.Fatal("fault")
		}
	}
}

func TestThread4(t *testing.T) {
	cases := 0
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

	s := []string{"build", "-o", "threadtest4"}
	if *memTrace {
		s = append(s, "-tags", "memory.trace", "-race")
	}
	if out, err := exec.Command("go", append(s, "github.com/cznic/sqlite/internal/threadtest4")...).CombinedOutput(); err != nil {
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
		out, err := exec.Command("./threadtest4", append(opts, "5")...).CombinedOutput()
		t.Logf("%v\n%s", opts, out)
		if err != nil {
			t.Fatal(err)
		}

		if bytes.Contains(out, []byte("fault address")) ||
			bytes.Contains(out, []byte("data race")) ||
			bytes.Contains(out, []byte("RACE")) {
			t.Fatalf("case %v: fault", cases)
		}
		cases++
	}
	t.Logf("cases: %v", cases)
}
