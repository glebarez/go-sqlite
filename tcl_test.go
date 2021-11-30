// Copyright 2020 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlite // import "modernc.org/sqlite"

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"modernc.org/tcl"
)

var (
	oMaxError = flag.Uint("maxerror", 0, "argument of -maxerror passed to the Tcl test suite")
	oStart    = flag.String("start", "", "argument of -start passed to the Tcl test suite (-start=[$permutation:]$testfile)")
	oTclSuite = flag.String("suite", "full", "Tcl test suite [test-file] to run")
	oVerbose  = flag.String("verbose", "0", "argument of -verbose passed to the Tcl test suite, must be set to a boolean (0, 1) or to \"file\"")
)

func TestTclTest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	blacklist := map[string]struct{}{}
	switch runtime.GOARCH {
	case "386", "arm":
		// # This test causes thrashing on machines with smaller amounts of
		// # memory.  Make sure the host has at least 8GB available before running
		// # this test.
		blacklist["bigsort.test"] = struct{}{}
	case "s390x":
		blacklist["sysfault.test"] = struct{}{} //TODO
	}
	switch runtime.GOOS {
	case "freebsd":
		if err := setMaxOpenFiles(1024); err != nil { // Avoid misc7.test hanging for a long time.
			t.Fatal(err)
		}
	case "windows":
		// See https://gitlab.com/cznic/sqlite/-/issues/23#note_599920077 for details.
		blacklist["symlink2.test"] = struct{}{}
		blacklist["zipfile.test"] = struct{}{} //TODO
	}
	tclTests := "testdata/3.37.0/tcl/*"
	switch fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH) {
	case
		"linux/s390x",
		"netbsd/amd64",
		"windows/386",
		"windows/amd64":

		tclTests = "testdata/3.36.0/tcl/*"
	}
	m, err := filepath.Glob(filepath.FromSlash(tclTests))
	if err != nil {
		t.Fatal(err)
	}

	dir, err := ioutil.TempDir("", "sqlite-test-")
	if err != nil {
		t.Fatal(err)
	}

	defer os.RemoveAll(dir)

	bin := "testfixture"
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	testfixture := filepath.Join(dir, bin)
	args0 := []string{"build", "-o", testfixture}
	tags := "-tags=libc.nofsync"
	if s := *oXTags; s != "" {
		tags += "," + s
	}
	args0 = append(args0, tags, "modernc.org/sqlite/internal/testfixture")
	cmd := exec.Command("go", args0...)
	if out, err := cmd.CombinedOutput(); err != nil {
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

	for _, v := range m {
		if _, ok := blacklist[filepath.Base(v)]; ok {
			continue
		}

		s := filepath.Join(wd, v)
		d := filepath.Join(dir, filepath.Base(v))
		f, err := ioutil.ReadFile(s)
		if err != nil {
			t.Fatal(err)
		}

		if err := ioutil.WriteFile(d, f, 0660); err != nil {
			t.Fatal(err)
		}
	}

	library := filepath.Join(dir, "library")
	if err := tcl.Library(library); err != nil {
		t.Fatal(err)
	}

	os.Setenv("TCL_LIBRARY", library)
	var args []string
	switch s := *oTclSuite; s {
	case "":
		args = []string{"all.test"}
	default:
		a := strings.Split(s, " ")
		args = append([]string{"permutations.test"}, a...)
	}
	if *oVerbose != "" {
		args = append(args, fmt.Sprintf("-verbose=%s", *oVerbose))
	}
	if *oMaxError != 0 {
		args = append(args, fmt.Sprintf("-maxerror=%d", *oMaxError))
	}
	if *oStart != "" {
		args = append(args, fmt.Sprintf("-start=%s", *oStart))
	}
	os.Setenv("PATH", fmt.Sprintf("%s%c%s", dir, os.PathListSeparator, os.Getenv("PATH")))
	cmd = exec.Command(bin, args...)
	var w echoWriter
	cmd.Stdout = &w
	cmd.Stderr = &w
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	if b := w.w.Bytes(); bytes.Contains(b, []byte("while executing")) {
		t.Fatal("silent/unreported error detected in output")
	}
}

type echoWriter struct {
	w bytes.Buffer
}

func (w *echoWriter) Write(b []byte) (int, error) {
	os.Stdout.Write(b)
	return w.w.Write(b)
}
