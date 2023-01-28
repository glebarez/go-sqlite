// Copyright 2021 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package benchmark

import (
	"database/sql"
	"fmt"
	"math/rand"
	"path"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

const (
	// maximum for randomly generated number
	maxGeneratedNum = 1000000

	// default row count for pre-filled test table
	testTableRowCount = 100000

	// default name for test table
	testTableName = "t1"
)

var (
	matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

// mustExec executes SQL statements and panic if error occurs
func mustExec(db *sql.DB, statements ...string) {
	for _, s := range statements {
		if _, err := db.Exec(s); err != nil {
			panic(fmt.Sprintf("%s: %v", s, err))
		}
	}
}

// createTestTableWithName creates new DB table
// with following DDL:
//
// CREATE TABLE <tableName>(a INTEGER, b INTEGER, c VARCHAR(100)).
//
// Additionaly, indicies are created for columns whose names are passed in indexedColumns.
func createTestTableWithName(db *sql.DB, tableName string, indexedColumns ...string) {
	// define table create statements
	statements := []string{
		fmt.Sprintf(`DROP TABLE IF EXISTS %s`, tableName),
		fmt.Sprintf(`CREATE TABLE %s(a INTEGER, b INTEGER, c VARCHAR(100))`, tableName),
	}

	// add index creating statements
	for i, indexedColumn := range indexedColumns {
		statements = append(statements, fmt.Sprintf(`CREATE INDEX i%d ON %s(%s)`, i+1, tableName, indexedColumn))
	}

	// execute table creation
	mustExec(db, statements...)
}

// createTestTable is a wrapper for createTestTableWithName with default table name
func createTestTable(db *sql.DB, indexedColumns ...string) {
	createTestTableWithName(db, testTableName, indexedColumns...)
}

// runInTransaction executes f() inside BEGIN/COMMIT block
func runInTransaction(db *sql.DB, f func()) {
	if _, err := db.Exec(`BEGIN`); err != nil {
		panic(err)
	}
	f()
	if _, err := db.Exec(`COMMIT`); err != nil {
		panic(err)
	}
}

// fillTestTable inserts <rowCount> rows into test table of default name (testTableName).
// the values of columns are as follows:
//
// a - sequence number starting from 1
//
// b - random number between 0 and maxGeneratedNum
//
// c - text with english prononciation of b
//
// for example, SQL statements will be similiar to following:
//
// INSERT INTO t1 VALUES(1,13153,'thirteen thousand one hundred fifty three');
//
// INSERT INTO t1 VALUES(2,75560,'seventy five thousand five hundred sixty');
func fillTestTable(db *sql.DB, rowCount int) {
	// prepare statement for insertion of rows
	stmt, err := db.Prepare(fmt.Sprintf("INSERT INTO %s VALUES(?,?,?)", testTableName))
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	// insert rows
	for i := 0; i < rowCount; i++ {
		// generate random number
		num := rand.Int31n(maxGeneratedNum)

		// get number as words
		numAsWords := pronounceNum(uint32(num))

		// insert row
		if _, err := stmt.Exec(i+1, num, numAsWords); err != nil {
			panic(err)
		}
	}
}

// fillTestTableInTx calls fillTestTable inside a transaction
func fillTestTableInTx(db *sql.DB, rowCount int) {
	runInTransaction(db, func() {
		fillTestTable(db, rowCount)
	})
}

// createDB creates a new instance of sql.DB with specified SQLite driver name.
// if inMemory = false, then database file is created in random temporary directory via tb.TempDir()
func createDB(tb testing.TB, inMemory bool, driverName string) *sql.DB {
	var dsn string
	if inMemory {
		dsn = ":memory:"
	} else {
		dsn = path.Join(tb.TempDir(), "test.db")
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		panic(err)
	}
	db.SetMaxOpenConns(1)

	// when in on-disk mode - set synchronous = OFF
	// this turns off fsync() sys call at every record inserted out of transaction scope
	// thus we don't bother HDD/SSD too often during specific bechmarks
	if !inMemory {
		// disable sync
		_, err = db.Exec(`PRAGMA synchronous = OFF`)
		if err != nil {
			tb.Fatal(err)
		}
	}
	return db
}

// getFuncName gets function name as string, without package prefix
func getFuncName(i interface{}) string {
	// get function name as "package.function"
	fn := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()

	// return last component
	comps := strings.Split(fn, ".")
	return comps[len(comps)-1]
}

// pronounceNum generates english pronounciation for a given number
func pronounceNum(n uint32) string {
	switch {
	case n == 0:
		return `zero`
	case n < 10:
		return []string{`one`, `two`, `three`, `four`, `five`, `six`, `seven`, `eight`, `nine`}[n-1]
	case n < 20:
		return []string{`ten`, `eleven`, `twelve`, `thirteen`, `fourteen`, `fifteen`, `sixteen`, `seventeen`, `eighteen`, `nineteen`}[n-10]
	case n < 100:
		p := []string{`twenty`, `thirty`, `forty`, `fifty`, `sixty`, `seventy`, `eighty`, `ninety`}[n/10-2]
		if n%10 == 0 {
			return p
		}
		return fmt.Sprint(p, " ", pronounceNum(n%10))
	default:
		divisors := []struct {
			num  uint32
			name string
		}{
			{num: 1000000000, name: `1e9`},
			{num: 1000000, name: `million`},
			{num: 1000, name: `thousand`},
			{num: 100, name: `hundred`},
		}

		for _, div := range divisors {
			if n >= div.num {
				p := fmt.Sprint(pronounceNum(n/div.num), " ", div.name)
				if n%div.num == 0 {
					return p
				}
				return fmt.Sprint(p, " ", pronounceNum(n%div.num))
			}
		}
	}
	panic("must have returned already")
}

// AvgVal provides average value with value contributions on-the-fly
type avgVal struct {
	// the current average value
	val float64

	// number of contribuitons for involved in current average
	numContributions int
}

func (a *avgVal) contribFloat(v float64) {
	nContrib := float64(a.numContributions)
	a.val = (a.val*nContrib + v) / (nContrib + 1.)
	a.numContributions++
}

func (a *avgVal) contribInt(v int64) {
	a.contribFloat(float64(v))
}
