package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func main() {
	if err := main1(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main1() error {
	dir, err := ioutil.TempDir("", "test-")
	if err != nil {
		return err
	}

	defer os.RemoveAll(dir)

	fn := filepath.Join(dir, "db")

	db, err := sql.Open("sqlite", fn)
	if err != nil {
		return err
	}

	if _, err := db.Exec("create table if not exists t(i);"); err != nil {
		return err
	}

	if err := db.Close(); err != nil {
		return err
	}

	fi, err := os.Stat(fn)
	if err != nil {
		return err
	}

	fmt.Printf("%s size: %v\n", fn, fi.Size())
	return nil
}
