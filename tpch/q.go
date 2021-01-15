// Copyright 2032 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"database/sql"
	"fmt"
	"math"
	"time"

	"modernc.org/sqlite/tpch/driver"
)

func cpDB(sut driver.SUT, src, dest *sql.DB) error {
	for _, v := range []struct {
		t, i string
		int
	}{
		{"customer", sut.InsertCustomer(), 8},
		{"lineitem", sut.InsertLineItem(), 16},
		{"nation", sut.InsertNation(), 4},
		{"orders", sut.InsertOrders(), 9},
		{"part", sut.InsertPart(), 9},
		{"partsupp", sut.InsertPartSupp(), 5},
		{"region", sut.InsertRegion(), 3},
		{"supplier", sut.InsertSupplier(), 7},
	} {
		if err := cpTable(src, dest, v.t, v.i, v.int); err != nil {
			return err
		}
	}
	return nil
}

func cpTable(src, dest *sql.DB, tn, ins string, ncols int) (err error) {
	rows, err := src.Query(fmt.Sprintf("select * from %s", tn))
	if err != nil {
		return err
	}

	defer func() {
		if e := rows.Close(); e != nil && err == nil {
			err = e
		}
	}()

	var tx *sql.Tx
	var stmt *sql.Stmt
	row := make([]interface{}, ncols)
	data := make([]interface{}, ncols)
	for i := range data {
		data[i] = &row[i]
	}
	for i := 0; rows.Next(); i++ {
		if i%1000 == 0 {
			if i != 0 {
				if err = tx.Commit(); err != nil {
					return err
				}
			}

			if tx, err = dest.Begin(); err != nil {
				return err
			}

			if stmt, err = tx.Prepare(ins); err != nil {
				return err
			}
		}

		if err = rows.Scan(data...); err != nil {
			return err
		}

		for i, v := range row {
			switch x := v.(type) {
			case time.Time:
			case int64:
			case []byte:
				row[i] = string(x)
			default:
				panic("TODO")
			}
		}
		if _, err = stmt.Exec(row...); err != nil {
			return err
		}

	}
	if err = rows.Err(); err != nil {
		return err
	}

	return tx.Commit()
}

func run(sut driver.SUT, mem bool, n, sf int, verbose bool) (err error) {
	pth := pthForSUT(sut, sf)
	if err := sut.SetWD(pth); err != nil {
		return err
	}

	db, err := sut.OpenDB()
	if err != nil {
		return err
	}

	defer func(db *sql.DB) {
		if cerr := db.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}(db)

	if mem {
		msut, mdb, err := sut.OpenMem()
		if err != nil {
			return err
		}

		if err = msut.CreateTables(); err != nil {
			return err
		}

		if err = cpDB(sut, db, mdb); err != nil {
			return err
		}

		sut, db = msut, mdb
	}

	rng := newRng(0, math.MaxInt64)
	rng.r.Seed(time.Now().UnixNano())
	t0 := time.Now()

	defer func() {
		fmt.Println(time.Since(t0))
	}()

	switch n {
	case 1:
		return exec(db, 10, sut.Q1(), verbose, rng.randomValue(60, 120))
	case 2:
		return exec(db, 8, sut.Q2(), verbose, rng.randomValue(1, 50), rng.types(), rng.regions())
	default:
		return fmt.Errorf("No query/test #%d", n)
	}
}

func exec(db *sql.DB, ncols int, q string, verbose bool, arg ...interface{}) error {
	rec := make([]interface{}, ncols)
	data := make([]interface{}, ncols)
	for i := range data {
		data[i] = &rec[i]
	}

	rows, err := db.Query(q, arg...)
	if err != nil {
		return err
	}

	defer func() {
		if e := rows.Close(); e != nil && err == nil {
			err = e
		}
	}()

	for rows.Next() {
		if !verbose {
			continue
		}

		if err = rows.Scan(data...); err != nil {
			return err
		}

		for i, v := range rec {
			if b, ok := v.([]byte); ok {
				rec[i] = string(b)
			}
		}

		fmt.Println(rec)
	}
	return rows.Err()
}
