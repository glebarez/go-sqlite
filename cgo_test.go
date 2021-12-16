// Copyright 2017 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ignore || (cgo && cgotest)
// +build ignore cgo,cgotest

package sqlite // import "modernc.org/sqlite"
import (
	"database/sql"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// https://gitlab.com/cznic/sqlite/-/issues/65
func TestIssue65CGo(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		os.RemoveAll(tempDir)
	}()

	db, err := sql.Open("sqlite3", filepath.Join(tempDir, "testissue65.sqlite"))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	testIssue65(t, db, false)
}
