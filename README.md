[![Tests](https://github.com/glebarez/go-sqlite/actions/workflows/tests.yml/badge.svg)](https://github.com/glebarez/go-sqlite/actions/workflows/tests.yml)<br>
![badge](https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/glebarez/fb4d23f63d866b3e1e58b26d2f5ed01f/raw/badge-sqlite-version.json)

# go-sqlite
This is a pure-Go SQLite driver for Golang's native [database/sql](https://pkg.go.dev/database/sql) package.

This driver is based on pure-Go implementation of SQLite (https://gitlab.com/cznic/sqlite), and has SQLite embedded.

# Usage

## Example

```go
package main

import (
	"database/sql"
	"log"

	_ "github.com/glebarez/go-sqlite"
)

func main() {
	// connect
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatal(err)
	}

	// get SQLite version
	_ := db.QueryRow("select sqlite_version()")
}
```

## Connection string examples
- in-memory SQLite: ```":memory:"```
- on-disk SQLite: ```"path/to/some.db"```
- Foreign-key constraint activation: ```":memory:?_pragma=foreign_keys(1)"```
