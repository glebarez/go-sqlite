// Copyright 2017 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//+build cgo,cgobench

package sqlite // import "modernc.org/sqlite"

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

const gcoDriver = "sqlite3"

func prepareDatabase() string {
	//if this fails you should probably clean your folders
	for i := 0; ; i++ {
		path := fmt.Sprintf("%dbench.db", i)
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			return path
		}
	}
}

var drivers = []string{
	driverName,
	gcoDriver,
}

var inMemory = []bool{
	true,
	false,
}

func makename(inMemory bool, driver string) string {
	name := driver
	if inMemory {
		name += "InMemory"
	} else {
		name += "OnDisk"
	}
	return name
}

func reading1Memory(b *testing.B, drivername, file string) {
	db, err := sql.Open(drivername, file)
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
		b.Fatal(err)
	}

	defer r.Close()
	dst := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !r.Next() {
			b.Fatal(r.Err())
		}
		err = r.Scan(&dst)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	if *oRecsPerSec {
		b.SetBytes(1e6)
	}
}

func BenchmarkReading1(b *testing.B) {
	for _, memory := range inMemory {
		filename := "file::memory:"
		if !memory {
			filename = prepareDatabase()
		}
		for _, driver := range drivers {
			b.Run(makename(memory, driver), func(b *testing.B) {
				reading1Memory(b, driver, filename)
				if !memory {
					err := os.Remove(filename)
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}
