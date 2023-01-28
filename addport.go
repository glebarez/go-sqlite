// Copyright 2022 The Sqlite Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func fail(rc int, msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg, args...)
	os.Exit(rc)
}

func main() {
	if len(os.Args) != 3 {
		fail(1, "expected 2 args: pattern and replacement\n")
	}

	pattern := os.Args[1]
	replacement := os.Args[2]
	if err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		dir, file := filepath.Split(path)
		if x := strings.Index(file, pattern); x >= 0 {
			// pattern      freebsd
			// replacement  netbsd
			// file         libc_freebsd_amd64.go
			// replaced     libc_netbsd_amd64.go
			//              01234567890123456789
			//                        1
			// x            5
			file = file[:x] + replacement + file[x+len(pattern):]
			dst := filepath.Join(dir, file)
			b, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("reading %s: %v", path, err)
			}

			if err := os.WriteFile(dst, b, 0640); err != nil {
				return fmt.Errorf("writing %s: %v", dst, err)
			}
			fmt.Printf("%s -> %s\n", path, dst)
		}

		return nil
	}); err != nil {
		fail(1, "%s", err)
	}
}
