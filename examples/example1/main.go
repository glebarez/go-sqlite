package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/glebarez/go-sqlite"
)

func main() {
	if err := main1(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main1() error {
	dir, err := os.MkdirTemp("", "test-")
	if err != nil {
		return err
	}

	defer os.RemoveAll(dir)

	fn := filepath.Join(dir, "db")

	db, err := sql.Open("sqlite", fn)
	if err != nil {
		return err
	}

	if _, err = db.Exec(`
drop table if exists t;
create table t(i);
insert into t values(42), (314);
`); err != nil {
		return err
	}

	rows, err := db.Query("select 3*i from t order by i;")
	if err != nil {
		return err
	}

	for rows.Next() {
		var i int
		if err = rows.Scan(&i); err != nil {
			return err
		}

		fmt.Println(i)
	}

	if err = rows.Err(); err != nil {
		return err
	}

	if err = db.Close(); err != nil {
		return err
	}

	fi, err := os.Stat(fn)
	if err != nil {
		return err
	}

	fmt.Printf("%s size: %v\n", fn, fi.Size())
	return nil
}
