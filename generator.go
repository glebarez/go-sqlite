// Copyright 2017 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package main

import (
	"archive/zip"
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
		"-DLONGDOUBLE_TYPE=double",
		"-DSQLITE_DEBUG", //TODO-
		"-DSQLITE_DEFAULT_MEMSTATUS=0",
		"-DSQLITE_DEFAULT_WAL_SYNCHRONOUS=1",
		"-DSQLITE_DQS=0",
		"-DSQLITE_LIKE_DOESNT_MATCH_BLOBS",
		"-DSQLITE_MAX_EXPR_DEPTH=0",
		"-DSQLITE_MEMDEBUG", //TODO-
		"-DSQLITE_OMIT_DECLTYPE",
		"-DSQLITE_OMIT_DEPRECATED",
		"-DSQLITE_OMIT_PROGRESS_CALLBACK",
		"-DSQLITE_OMIT_SHARED_CACHE",
		"-DSQLITE_THREADSAFE=0",
		"-DSQLITE_USE_ALLOCA",
	}

	downloads = []struct {
		dir, url string
		sz       int
		dev      bool
	}{
		{sqliteDir, "https://www.sqlite.org/2019/sqlite-amalgamation-3300100.zip", 2400, false},
	}

	sqliteDir = filepath.FromSlash("testdata/sqlite-amalgamation-3300100")
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

						dname := filepath.Join(root, f.Name)
						g, err := os.Create(dname)
						if err != nil {
							return err
						}

						defer g.Close()

						n, err = io.Copy(g, rc)
						return err
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
	out, err := exec.Command(
		"gocc",
		append(
			[]string{
				filepath.Join(sqliteDir, "sqlite3.c"),
				"-o", filepath.FromSlash(fmt.Sprintf("internal/bin/sqlite_%s_%s.go", runtime.GOOS, runtime.GOARCH)),
				"-qbec-pkgname", "bin",
			},
			config...)...,
	).CombinedOutput()
	if err != nil {
		fail("%s\n%s\n", out, err)
	}

	dir, err := ioutil.TempDir("", "go-generate-")
	if err != nil {
		fail("s\n", err)
	}

	defer os.RemoveAll(dir)

	src := "#include \"sqlite3.h\"\nstatic char _;\n"
	fn := filepath.Join(dir, "x.c")
	if err := ioutil.WriteFile(fn, []byte(src), 0660); err != nil {
		fail("s\n", err)
	}

	out, err = exec.Command(
		"gocc",
		append(
			[]string{
				fn,
				"-o", filepath.FromSlash(fmt.Sprintf("internal/bin/h_%s_%s.go", runtime.GOOS, runtime.GOARCH)),
				fmt.Sprintf("-I%s", sqliteDir),
				"-qbec-defines",
				"-qbec-enumconsts",
				"-qbec-import", "<none>",
				"-qbec-pkgname", "bin",
				"-qbec-structs",
			},
			config...)...,
	).CombinedOutput()
	if err != nil {
		fail("%s\n%s\n", out, err)
	}
}
