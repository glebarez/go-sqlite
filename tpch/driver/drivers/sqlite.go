// Copyright 2021 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package drivers

import (
	"database/sql"
	"path/filepath"

	_ "modernc.org/sqlite"
	"modernc.org/sqlite/tpch/driver"
)

func init() {
	driver.Register(newSQLite())
}

var _ driver.SUT = (*sqlite)(nil)

type sqlite struct {
	*sqlite3
}

func newSQLite() *sqlite {
	return &sqlite{newSQLite3()}
}

func (b *sqlite) Name() string { return "sqlite" }

func (b *sqlite) OpenDB() (*sql.DB, error) {
	pth := filepath.Join(b.wd, "sqlite.db")
	db, err := sql.Open(b.Name(), pth)
	if err != nil {
		return nil, err
	}

	b.db = db
	return db, nil
}

func (b *sqlite) OpenMem() (driver.SUT, *sql.DB, error) {
	db, err := sql.Open(b.Name(), "file::memory:")
	if err != nil {
		return nil, nil, err
	}

	return &sqlite{&sqlite3{db: db}}, db, nil
}
