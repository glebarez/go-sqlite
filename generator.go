// Copyright 2017 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	config = []string{
		"-DHAVE_USLEEP",
		"-DLONGDOUBLE_TYPE=double",
		"-DNDEBUG",
		"-DSQLITE_CORE", // testfixture
		"-DSQLITE_DEFAULT_MEMSTATUS=0",
		"-DSQLITE_DEFAULT_PAGE_SIZE=1024", // testfixture, hardcoded. See file_pages in autovacuum.test.
		"-DSQLITE_DEFAULT_WAL_SYNCHRONOUS=1",
		"-DSQLITE_DQS=0",
		"-DSQLITE_ENABLE_BYTECODE_VTAB", // testfixture
		"-DSQLITE_ENABLE_DBPAGE_VTAB",   // testfixture
		"-DSQLITE_ENABLE_DESERIALIZE",   // testfixture
		"-DSQLITE_ENABLE_STMTVTAB",      // testfixture
		"-DSQLITE_ENABLE_UNLOCK_NOTIFY", // Adds sqlite3_unlock_notify().
		"-DSQLITE_HAVE_ZLIB=1",          // testfixture
		"-DSQLITE_LIKE_DOESNT_MATCH_BLOBS",
		"-DSQLITE_MAX_EXPR_DEPTH=0",
		"-DSQLITE_MAX_MMAP_SIZE=8589934592", // testfixture
		"-DSQLITE_MUTEX_APPDEF=1",
		"-DSQLITE_MUTEX_NOOP",
		"-DSQLITE_NO_SYNC=1",                  // testfixture
		"-DSQLITE_OS_UNIX=1",                  // testfixture //TODO adjust for non unix OS
		"-DSQLITE_SERIES_CONSTRAINT_VERIFY=1", // testfixture
		"-DSQLITE_SERVER=1",                   // testfixture
		"-DSQLITE_TEMP_STORE=1",               // testfixture
		"-DSQLITE_TEST",
		"-DSQLITE_THREADSAFE=1",
		"-ccgo-long-double-is-double",
		// "-DSQLITE_OMIT_DECLTYPE", // testfixture
		// "-DSQLITE_OMIT_DEPRECATED", // mptest
		// "-DSQLITE_OMIT_LOAD_EXTENSION", // mptest
		// "-DSQLITE_OMIT_SHARED_CACHE",
		// "-DSQLITE_USE_ALLOCA",
		//TODO "-DHAVE_MALLOC_USABLE_SIZE"
		//TODO 386 "-DSQLITE_MAX_MMAP_SIZE=0", // mmap somehow fails on linux/386
		//TODO- "-DSQLITE_DEBUG", //TODO-
		//TODO- "-DSQLITE_ENABLE_API_ARMOR",     //TODO-
		//TODO- "-DSQLITE_MEMDEBUG", //TODO-
		//TODO- "-ccgo-verify-structs", //TODO-
	}

	downloads = []struct {
		dir, url string
		sz       int
		dev      bool
	}{
		{sqliteDir, "https://www.sqlite.org/2020/sqlite-amalgamation-3320300.zip", 2240, false},
		{sqliteSrcDir, "https://www.sqlite.org/2020/sqlite-src-3320300.zip", 12060, false},
	}

	sqliteDir    = filepath.FromSlash("testdata/sqlite-amalgamation-3320300")
	sqliteSrcDir = filepath.FromSlash("testdata/sqlite-src-3320300")
)

func download() {
	tmp, err := ioutil.TempDir("", "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return
	}

	defer os.RemoveAll(tmp)

	for _, v := range downloads {
		dir := filepath.FromSlash(v.dir)
		root := filepath.Dir(v.dir)
		fi, err := os.Stat(dir)
		switch {
		case err == nil:
			if !fi.IsDir() {
				fmt.Fprintf(os.Stderr, "expected %s to be a directory\n", dir)
			}
			continue
		default:
			if !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "%s", err)
				continue
			}
		}

		if err := func() error {
			fmt.Printf("Downloading %v MB from %s\n", float64(v.sz)/1000, v.url)
			resp, err := http.Get(v.url)
			if err != nil {
				return err
			}

			defer resp.Body.Close()

			base := filepath.Base(v.url)
			name := filepath.Join(tmp, base)
			f, err := os.Create(name)
			if err != nil {
				return err
			}

			defer os.Remove(name)

			n, err := io.Copy(f, resp.Body)
			if err != nil {
				return err
			}

			if _, err := f.Seek(0, io.SeekStart); err != nil {
				return err
			}

			switch {
			case strings.HasSuffix(base, ".zip"):
				r, err := zip.NewReader(f, n)
				if err != nil {
					return err
				}

				for _, f := range r.File {
					fi := f.FileInfo()
					if fi.IsDir() {
						if err := os.MkdirAll(filepath.Join(root, f.Name), 0770); err != nil {
							return err
						}

						continue
					}

					if err := func() error {
						rc, err := f.Open()
						if err != nil {
							return err
						}

						defer rc.Close()

						file, err := os.OpenFile(filepath.Join(root, f.Name), os.O_CREATE|os.O_WRONLY, fi.Mode())
						if err != nil {
							return err
						}

						w := bufio.NewWriter(file)
						if _, err = io.Copy(w, rc); err != nil {
							return err
						}

						if err := w.Flush(); err != nil {
							return err
						}

						return file.Close()
					}(); err != nil {
						return err
					}
				}
				return nil
			}
			panic("internal error") //TODOOK
		}(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}

func fail(s string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, s, args...)
	os.Exit(1)
}

func main() {
	download()
	makeSqlite()
	makeMpTest()
	makeSpeedTest()
	makeTestfixture()

	dst := filepath.FromSlash("testdata/tcl")
	if err := os.MkdirAll(dst, 0770); err != nil {
		fail("cannot create %q: %v", dst, err)
	}

	m, err := filepath.Glob(filepath.Join(sqliteSrcDir, "test/*.test"))
	if err != nil {
		fail("cannot glob *.test: %v", err)
	}

	m2, err := filepath.Glob(filepath.Join(sqliteSrcDir, "test/*.tcl"))
	if err != nil {
		fail("cannot glob *.tcl: %v", err)
	}

	m = append(m, m2...)
	for _, v := range m {
		f, err := ioutil.ReadFile(v)
		if err != nil {
			fail("cannot read %v: %v", v, err)
		}

		fn := filepath.Join(dst, filepath.Base(v))
		if err := ioutil.WriteFile(fn, f, 0660); err != nil {
			fail("cannot write %v: %v", fn, err)
		}
	}
}

func configure() {
	wd, err := os.Getwd()
	if err != nil {
		fail("%s", err)
	}

	defer os.Chdir(wd)

	if err := os.Chdir(sqliteSrcDir); err != nil {
		fail("%s", err)
	}

	cmd := newCmd("./configure")
	if err = cmd.Run(); err != nil {
		fail("%s\n", err)
	}

	cmd = newCmd("make", "parse.h", "opcodes.h")
	if err = cmd.Run(); err != nil {
		fail("%s\n", err)
	}
}

func newCmd(bin string, args ...string) *exec.Cmd {
	fmt.Printf("==== newCmd %s\n", bin)
	for _, v := range args {
		fmt.Printf("\t%v\n", v)
	}
	r := exec.Command(bin, args...)
	r.Stdout = os.Stdout
	r.Stderr = os.Stderr
	return r
}

func makeTestfixture() {
	dir := filepath.FromSlash(fmt.Sprintf("internal/testfixture"))
	configure()
	cmd := newCmd(
		"ccgo",
		append(
			[]string{
				"-DSQLITE_OMIT_LOAD_EXTENSION",
				"-DTCLSH_INIT_PROC=sqlite3TestInit",
				"-I/usr/include/tcl8.6",
				"-ccgo-export-defines", "",
				"-ccgo-export-fields", "F",
				"-ccgo-pkgname", "testfixture",
				"-l", "modernc.org/tcl/lib,modernc.org/sqlite/internal/crt2,modernc.org/sqlite/lib",
				"-o", filepath.Join(dir, fmt.Sprintf("testfixture_%s_%s.go", runtime.GOOS, runtime.GOARCH)),
				//TODO- "-ccgo-watch-instrumentation", //TODO-
				filepath.Join(sqliteSrcDir, "ext", "expert", "sqlite3expert.c"),
				filepath.Join(sqliteSrcDir, "ext", "expert", "test_expert.c"),
				filepath.Join(sqliteSrcDir, "ext", "fts5", "fts5_tcl.c"),
				filepath.Join(sqliteSrcDir, "ext", "misc", "amatch.c"),
				filepath.Join(sqliteSrcDir, "ext", "misc", "carray.c"),
				filepath.Join(sqliteSrcDir, "ext", "misc", "closure.c"),
				filepath.Join(sqliteSrcDir, "ext", "misc", "csv.c"),
				filepath.Join(sqliteSrcDir, "ext", "misc", "eval.c"),
				filepath.Join(sqliteSrcDir, "ext", "misc", "explain.c"),
				filepath.Join(sqliteSrcDir, "ext", "misc", "fileio.c"),
				filepath.Join(sqliteSrcDir, "ext", "misc", "fuzzer.c"),
				filepath.Join(sqliteSrcDir, "ext", "misc", "ieee754.c"),
				filepath.Join(sqliteSrcDir, "ext", "misc", "mmapwarm.c"),
				filepath.Join(sqliteSrcDir, "ext", "misc", "nextchar.c"),
				filepath.Join(sqliteSrcDir, "ext", "misc", "normalize.c"),
				filepath.Join(sqliteSrcDir, "ext", "misc", "percentile.c"),
				filepath.Join(sqliteSrcDir, "ext", "misc", "prefixes.c"),
				filepath.Join(sqliteSrcDir, "ext", "misc", "regexp.c"),
				filepath.Join(sqliteSrcDir, "ext", "misc", "remember.c"),
				filepath.Join(sqliteSrcDir, "ext", "misc", "series.c"),
				filepath.Join(sqliteSrcDir, "ext", "misc", "spellfix.c"),
				filepath.Join(sqliteSrcDir, "ext", "misc", "totype.c"),
				filepath.Join(sqliteSrcDir, "ext", "misc", "unionvtab.c"),
				filepath.Join(sqliteSrcDir, "ext", "misc", "wholenumber.c"),
				filepath.Join(sqliteSrcDir, "ext", "misc", "zipfile.c"),
				filepath.Join(sqliteSrcDir, "ext", "rbu", "sqlite3rbu.c"),
				filepath.Join(sqliteSrcDir, "ext", "rbu", "test_rbu.c"),
				filepath.Join(sqliteSrcDir, "src", "tclsqlite.c"),
				filepath.Join(sqliteSrcDir, "src", "test1.c"),
				filepath.Join(sqliteSrcDir, "src", "test2.c"),
				filepath.Join(sqliteSrcDir, "src", "test3.c"),
				filepath.Join(sqliteSrcDir, "src", "test4.c"),
				filepath.Join(sqliteSrcDir, "src", "test5.c"),
				filepath.Join(sqliteSrcDir, "src", "test6.c"),
				filepath.Join(sqliteSrcDir, "src", "test7.c"),
				filepath.Join(sqliteSrcDir, "src", "test8.c"),
				filepath.Join(sqliteSrcDir, "src", "test9.c"),
				filepath.Join(sqliteSrcDir, "src", "test_async.c"),
				filepath.Join(sqliteSrcDir, "src", "test_autoext.c"),
				filepath.Join(sqliteSrcDir, "src", "test_backup.c"),
				filepath.Join(sqliteSrcDir, "src", "test_bestindex.c"),
				filepath.Join(sqliteSrcDir, "src", "test_blob.c"),
				filepath.Join(sqliteSrcDir, "src", "test_btree.c"),
				filepath.Join(sqliteSrcDir, "src", "test_config.c"),
				filepath.Join(sqliteSrcDir, "src", "test_delete.c"),
				filepath.Join(sqliteSrcDir, "src", "test_demovfs.c"),
				filepath.Join(sqliteSrcDir, "src", "test_devsym.c"),
				filepath.Join(sqliteSrcDir, "src", "test_fs.c"),
				filepath.Join(sqliteSrcDir, "src", "test_func.c"),
				filepath.Join(sqliteSrcDir, "src", "test_hexio.c"),
				filepath.Join(sqliteSrcDir, "src", "test_init.c"),
				filepath.Join(sqliteSrcDir, "src", "test_intarray.c"),
				filepath.Join(sqliteSrcDir, "src", "test_journal.c"),
				filepath.Join(sqliteSrcDir, "src", "test_malloc.c"),
				filepath.Join(sqliteSrcDir, "src", "test_md5.c"),
				filepath.Join(sqliteSrcDir, "src", "test_multiplex.c"),
				filepath.Join(sqliteSrcDir, "src", "test_mutex.c"),
				filepath.Join(sqliteSrcDir, "src", "test_onefile.c"),
				filepath.Join(sqliteSrcDir, "src", "test_osinst.c"),
				filepath.Join(sqliteSrcDir, "src", "test_pcache.c"),
				filepath.Join(sqliteSrcDir, "src", "test_quota.c"),
				filepath.Join(sqliteSrcDir, "src", "test_rtree.c"),
				filepath.Join(sqliteSrcDir, "src", "test_schema.c"),
				filepath.Join(sqliteSrcDir, "src", "test_server.c"),
				filepath.Join(sqliteSrcDir, "src", "test_superlock.c"),
				filepath.Join(sqliteSrcDir, "src", "test_syscall.c"),
				filepath.Join(sqliteSrcDir, "src", "test_tclsh.c"),
				filepath.Join(sqliteSrcDir, "src", "test_tclvar.c"),
				filepath.Join(sqliteSrcDir, "src", "test_thread.c"),
				filepath.Join(sqliteSrcDir, "src", "test_vdbecov.c"),
				filepath.Join(sqliteSrcDir, "src", "test_vfs.c"),
				filepath.Join(sqliteSrcDir, "src", "test_window.c"),
				fmt.Sprintf("-I%s", sqliteDir),
				fmt.Sprintf("-I%s", sqliteSrcDir),
			},
			config...)...,
	)
	if err := cmd.Run(); err != nil {
		fail("%s\n", err)
	}
	os.Remove(filepath.Join(dir, fmt.Sprintf("capi_%s_%s.go", runtime.GOOS, runtime.GOARCH)))
}

func makeSpeedTest() {
	cmd := newCmd(
		"ccgo",
		append(
			[]string{
				"-o", filepath.FromSlash(fmt.Sprintf("speedtest1/main_%s_%s.go", runtime.GOOS, runtime.GOARCH)),
				filepath.Join(sqliteSrcDir, "test", "speedtest1.c"),
				fmt.Sprintf("-I%s", sqliteDir),
				"-l", "modernc.org/sqlite/lib",
			},
			config...)...,
	)
	if err := cmd.Run(); err != nil {
		fail("%s\n", err)
	}
}

func makeMpTest() {
	cmd := newCmd(
		"ccgo",
		append(
			[]string{
				"-o", filepath.FromSlash(fmt.Sprintf("internal/mptest/main_%s_%s.go", runtime.GOOS, runtime.GOARCH)),
				filepath.Join(sqliteSrcDir, "mptest", "mptest.c"),
				fmt.Sprintf("-I%s", sqliteDir),
				"-l", "modernc.org/sqlite/lib",
			},
			config...)...,
	)
	if err := cmd.Run(); err != nil {
		fail("%s\n", err)
	}
}

func makeSqlite() {
	cmd := newCmd(
		"ccgo",
		append(
			[]string{
				"-DSQLITE_PRIVATE=",
				"-ccgo-export-defines", "",
				"-ccgo-export-externs", "X",
				"-ccgo-export-fields", "F",
				"-ccgo-export-typedefs", "",
				"-ccgo-pkgname", "sqlite3",
				"-o", filepath.FromSlash(fmt.Sprintf("lib/sqlite_%s_%s.go", runtime.GOOS, runtime.GOARCH)),
				//TODO "-ccgo-volatile", "sqlite3_io_error_pending,sqlite3_open_file_count,sqlite3_pager_readdb_count,sqlite3_search_count,sqlite3_sort_count",
				filepath.Join(sqliteDir, "sqlite3.c"),
			},
			config...)...,
	)
	if err := cmd.Run(); err != nil {
		fail("%s\n", err)
	}
}
