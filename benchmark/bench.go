// Copyright 2021 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package benchmark

/*
this file contains benchmarks inspired by https://www.sqlite.org/speed.html
*/

import (
	"database/sql"
	"fmt"
	"math/rand"
	"testing"
)

// corresponds to Test 1 from https://www.sqlite.org/speed.html
func benchInsert(b *testing.B, db *sql.DB) {
	// create test table
	createTestTable(db)

	// measure from here
	b.ResetTimer()

	fillTestTable(db, b.N)
}

// corresponds to Test 2 from https://www.sqlite.org/speed.html
func benchInsertInTransaction(b *testing.B, db *sql.DB) {
	// create test table
	createTestTable(db)

	// measure from here
	b.ResetTimer()

	fillTestTableInTx(db, b.N)
}

// corresponds to Test 3 from https://www.sqlite.org/speed.html
func benchInsertIntoIndexed(b *testing.B, db *sql.DB) {
	// create test table with indexed column
	createTestTable(db, `c`)

	b.ResetTimer()

	fillTestTableInTx(db, b.N)
}

// corresponds to Test 4 from https://www.sqlite.org/speed.html
func benchSelectWithoutIndex(b *testing.B, db *sql.DB) {
	// create test table
	createTestTable(db)

	// fill test table
	fillTestTableInTx(db, testTableRowCount)

	// prepare statement for selection
	stmt, err := db.Prepare(fmt.Sprintf(`SELECT count(*), avg(b) FROM %s WHERE b>=? AND b<?`, testTableName))
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	b.ResetTimer()

	runInTransaction(db, func() {
		// exec many selects
		// SELECT count(*), avg(b) FROM t2 WHERE b>=0 AND b<1000;
		// SELECT count(*), avg(b) FROM t2 WHERE b>=100 AND b<1100;
		// SELECT count(*), avg(b) FROM t2 WHERE b>=200 AND b<1200;
		// ...
		for i := 0; i < b.N; i++ {
			b := (i * 100) % maxGeneratedNum
			if _, err := stmt.Exec(b, b+1000); err != nil {
				panic(err)
			}
		}
	})
}

// corresponds to Test 5 from https://www.sqlite.org/speed.html
func benchSelectOnStringComparison(b *testing.B, db *sql.DB) {
	// create test table
	createTestTable(db)

	// fill test table
	fillTestTableInTx(db, testTableRowCount)

	// prepare statement for selection
	stmt, err := db.Prepare(fmt.Sprintf(`SELECT count(*), avg(b) FROM %s WHERE c LIKE ?`, testTableName))
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	b.ResetTimer()

	runInTransaction(db, func() {
		// exec many selects
		// SELECT count(*), avg(b) FROM t2 WHERE c LIKE '%one%';
		// SELECT count(*), avg(b) FROM t2 WHERE c LIKE '%two%';
		// ...
		// SELECT count(*), avg(b) FROM t2 WHERE c LIKE '%ninety nine%';
		for i := 1; i <= b.N; i++ {
			// e.g. following will produce "... WHERE c LIKE '%seventy eight%' "
			likeCond := pronounceNum(uint32(i) % 100)
			if _, err := stmt.Exec(`%` + likeCond + `%`); err != nil {
				panic(err)
			}
		}
	})
}

// corresponds to Test 6 from https://www.sqlite.org/speed.html
func benchCreateIndex(b *testing.B, db *sql.DB) {
	// create test table
	createTestTable(db)

	// fill test table
	fillTestTableInTx(db, testTableRowCount)

	var (
		nameIdxA       = "iA"
		nameIdxB       = "iB"
		createIndStmtA = fmt.Sprintf(`CREATE INDEX %s ON %s(a)`, nameIdxA, testTableName)
		createIndStmtB = fmt.Sprintf(`CREATE INDEX %s ON %s(c)`, nameIdxB, testTableName)
		dropIndStmtA   = fmt.Sprintf(`DROP INDEX IF EXISTS %s`, nameIdxA)
		dropIndStmtB   = fmt.Sprintf(`DROP INDEX IF EXISTS %s`, nameIdxB)
	)

	b.ResetTimer()
	for i := 1; i <= b.N; i++ {
		// drop indexes from previous iteration (if any)
		b.StopTimer()
		mustExec(db, dropIndStmtA, dropIndStmtB)
		b.StartTimer()

		// create indexes
		runInTransaction(db, func() {
			mustExec(db, createIndStmtA, createIndStmtB)
		})
	}
}

// corresponds to Test 7 from https://www.sqlite.org/speed.html
func benchSelectWithIndex(b *testing.B, db *sql.DB) {
	// create test table with indexed field
	createTestTable(db, `b`)

	// fill test table
	fillTestTableInTx(db, testTableRowCount)

	// prepare statement for selection
	stmt, err := db.Prepare(fmt.Sprintf(`SELECT count(*), avg(b) FROM %s WHERE b>=? AND b<?`, testTableName))
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	b.ResetTimer()

	runInTransaction(db, func() {
		// exec many selects
		// SELECT count(*), avg(b) FROM t2 WHERE b>=0 AND b<100;
		// SELECT count(*), avg(b) FROM t2 WHERE b>=100 AND b<200;
		// SELECT count(*), avg(b) FROM t2 WHERE b>=200 AND b<300;
		for i := 0; i < b.N; i++ {
			b := (i * 100) % maxGeneratedNum
			if _, err := stmt.Exec(b, b+100); err != nil {
				panic(err)
			}

		}
	})

}

// corresponds to Test 8 from https://www.sqlite.org/speed.html
func benchUpdateWithoutIndex(b *testing.B, db *sql.DB) {
	// create test table
	createTestTable(db)

	// fill test table
	fillTestTableInTx(db, testTableRowCount)

	// prepare statement
	stmt, err := db.Prepare(fmt.Sprintf(`UPDATE %s SET b=b*2 WHERE a>=? AND a<?`, testTableName))
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	b.ResetTimer()

	runInTransaction(db, func() {
		// exec many
		// UPDATE t1 SET b=b*2 WHERE a>=0 AND a<10;
		// UPDATE t1 SET b=b*2 WHERE a>=10 AND a<20;
		// UPDATE t1 SET b=b*2 WHERE a>=20 AND a<30;
		for i := 0; i < b.N; i++ {
			a := (i * 10) % testTableRowCount
			if _, err := stmt.Exec(a, a+10); err != nil {
				panic(err)
			}
		}
	})

}

// corresponds to Test 9 from https://www.sqlite.org/speed.html
func benchUpdateWithIndex(b *testing.B, db *sql.DB) {
	// create test table
	createTestTable(db, `a`)

	// fill test table
	fillTestTableInTx(db, testTableRowCount)

	// prepare statement
	stmt, err := db.Prepare(fmt.Sprintf(`UPDATE %s SET b=? WHERE a=?`, testTableName))
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	b.ResetTimer()

	runInTransaction(db, func() {
		// exec many
		// UPDATE t2 SET b=468026 WHERE a=1;
		// UPDATE t2 SET b=121928 WHERE a=2;
		// ...
		for i := 0; i < b.N; i++ {
			if _, err := stmt.Exec(
				rand.Uint32(),         // b = ?
				i%testTableRowCount+1, // WHERE a=?
			); err != nil {
				panic(err)
			}
		}
	})
}

// corresponds to Test 10 from https://www.sqlite.org/speed.html
func benchUpdateTextWithIndex(b *testing.B, db *sql.DB) {
	// create test table
	createTestTable(db, `a`)

	// fill test table
	fillTestTableInTx(db, testTableRowCount)

	// prepare statement
	stmt, err := db.Prepare(fmt.Sprintf(`UPDATE %s SET c=? WHERE a=?`, testTableName))
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	b.ResetTimer()

	runInTransaction(db, func() {
		// exec many
		// UPDATE t2 SET c='one hundred forty eight thousand three hundred eighty two' WHERE a=1;
		// UPDATE t2 SET c='three hundred sixty six thousand five hundred two' WHERE a=2;
		for i := 0; i < b.N; i++ {
			// generate new random number-as-words for c
			if _, err := stmt.Exec(
				pronounceNum(uint32(rand.Int31n(maxGeneratedNum))), // SET c=?
				i%testTableRowCount+1,                              // WHERE a=?
			); err != nil {
				panic(err)
			}
		}
	})
}

// corresponds to Test 11 from https://www.sqlite.org/speed.html
func benchInsertFromSelect(b *testing.B, db *sql.DB) {
	// create source table
	createTestTable(db)
	fillTestTableInTx(db, testTableRowCount)

	// create name for target table
	targetTableName := fmt.Sprintf("%s_copy", testTableName)

	// need to create table here, otherwise prepared statement will give error
	createTestTableWithName(db, targetTableName)

	// prepare statement
	stmt, err := db.Prepare(fmt.Sprintf(`INSERT INTO %s SELECT b,a,c FROM %s`, targetTableName, testTableName))
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// drop/create target table
		b.StopTimer()
		createTestTableWithName(db, targetTableName)
		b.StartTimer()

		runInTransaction(db, func() {
			if _, err := stmt.Exec(); err != nil {
				panic(err)
			}
		})
	}
}

// corresponds to Test 12 from https://www.sqlite.org/speed.html
func benchDeleteWithoutIndex(b *testing.B, db *sql.DB) {
	// create test table
	createTestTable(db)

	// prepare statement for deletion
	stmt, err := db.Prepare(fmt.Sprintf(`DELETE FROM %s WHERE c LIKE '%%fifty%%'`, testTableName))
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// drop/create target table
		b.StopTimer()
		fillTestTableInTx(db, testTableRowCount)
		b.StartTimer()

		runInTransaction(db, func() {
			if _, err := stmt.Exec(); err != nil {
				panic(err)
			}
		})
	}
}

// corresponds to Test 13 from https://www.sqlite.org/speed.html
func benchDeleteWithIndex(b *testing.B, db *sql.DB) {
	// create test table with indexed column
	createTestTable(db, `a`)

	// prepare statement for BIG deletion (nearly half of table rows)
	stmt, err := db.Prepare(fmt.Sprintf(`DELETE FROM %s WHERE a>%d AND a<%d`, testTableName, 10, testTableRowCount/2))
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// drop/create target table
		b.StopTimer()
		fillTestTableInTx(db, testTableRowCount)
		b.StartTimer()

		runInTransaction(db, func() {
			if _, err := stmt.Exec(); err != nil {
				panic(err)
			}
		})
	}
}

// corresponds to Test 16 from https://www.sqlite.org/speed.html
func benchDropTable(b *testing.B, db *sql.DB) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		createTestTable(db)
		fillTestTableInTx(db, testTableRowCount)
		b.StartTimer()

		if _, err := db.Exec(fmt.Sprintf("DROP TABLE %s", testTableName)); err != nil {
			panic(err)
		}
	}
}
