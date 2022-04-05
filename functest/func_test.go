// Copyright 2022 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package functest // modernc.org/sqlite/functest

import (
	"bytes"
	"crypto/md5"
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	sqlite3 "github.com/glebarez/go-sqlite"
)

func init() {
	sqlite3.MustRegisterDeterministicScalarFunction(
		"test_int64",
		0,
		func(ctx *sqlite3.FunctionContext, args []driver.Value) (driver.Value, error) {
			return int64(42), nil
		},
	)

	sqlite3.MustRegisterDeterministicScalarFunction(
		"test_float64",
		0,
		func(ctx *sqlite3.FunctionContext, args []driver.Value) (driver.Value, error) {
			return float64(1e-2), nil
		},
	)

	sqlite3.MustRegisterDeterministicScalarFunction(
		"test_null",
		0,
		func(ctx *sqlite3.FunctionContext, args []driver.Value) (driver.Value, error) {
			return nil, nil
		},
	)

	sqlite3.MustRegisterDeterministicScalarFunction(
		"test_error",
		0,
		func(ctx *sqlite3.FunctionContext, args []driver.Value) (driver.Value, error) {
			return nil, errors.New("boom")
		},
	)

	sqlite3.MustRegisterDeterministicScalarFunction(
		"test_empty_byte_slice",
		0,
		func(ctx *sqlite3.FunctionContext, args []driver.Value) (driver.Value, error) {
			return []byte{}, nil
		},
	)

	sqlite3.MustRegisterDeterministicScalarFunction(
		"test_nonempty_byte_slice",
		0,
		func(ctx *sqlite3.FunctionContext, args []driver.Value) (driver.Value, error) {
			return []byte("abcdefg"), nil
		},
	)

	sqlite3.MustRegisterDeterministicScalarFunction(
		"test_empty_string",
		0,
		func(ctx *sqlite3.FunctionContext, args []driver.Value) (driver.Value, error) {
			return "", nil
		},
	)

	sqlite3.MustRegisterDeterministicScalarFunction(
		"test_nonempty_string",
		0,
		func(ctx *sqlite3.FunctionContext, args []driver.Value) (driver.Value, error) {
			return "abcdefg", nil
		},
	)

	sqlite3.MustRegisterDeterministicScalarFunction(
		"yesterday",
		1,
		func(ctx *sqlite3.FunctionContext, args []driver.Value) (driver.Value, error) {
			var arg time.Time
			switch argTyped := args[0].(type) {
			case int64:
				arg = time.Unix(argTyped, 0)
			default:
				fmt.Println(argTyped)
				return nil, fmt.Errorf("expected argument to be int64, got: %T", argTyped)
			}
			return arg.Add(-24 * time.Hour), nil
		},
	)

	sqlite3.MustRegisterDeterministicScalarFunction(
		"md5",
		1,
		func(ctx *sqlite3.FunctionContext, args []driver.Value) (driver.Value, error) {
			var arg *bytes.Buffer
			switch argTyped := args[0].(type) {
			case string:
				arg = bytes.NewBuffer([]byte(argTyped))
			case []byte:
				arg = bytes.NewBuffer(argTyped)
			default:
				return nil, fmt.Errorf("expected argument to be a string, got: %T", argTyped)
			}
			w := md5.New()
			if _, err := arg.WriteTo(w); err != nil {
				return nil, fmt.Errorf("unable to compute md5 checksum: %s", err)
			}
			return hex.EncodeToString(w.Sum(nil)), nil
		},
	)
}

func TestRegisteredFunctions(t *testing.T) {
	withDB := func(test func(db *sql.DB)) {
		db, err := sql.Open("sqlite", "file::memory:")
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer db.Close()

		test(db)
	}

	t.Run("int64", func(tt *testing.T) {
		withDB(func(db *sql.DB) {
			row := db.QueryRow("select test_int64()")

			var a int
			if err := row.Scan(&a); err != nil {
				tt.Fatal(err)
			}
			if g, e := a, 42; g != e {
				tt.Fatal(g, e)
			}

		})
	})

	t.Run("float64", func(tt *testing.T) {
		withDB(func(db *sql.DB) {
			row := db.QueryRow("select test_float64()")

			var a float64
			if err := row.Scan(&a); err != nil {
				tt.Fatal(err)
			}
			if g, e := a, 1e-2; g != e {
				tt.Fatal(g, e)
			}

		})
	})

	t.Run("error", func(tt *testing.T) {
		withDB(func(db *sql.DB) {
			err := db.QueryRow("select test_error()").Scan()
			if err == nil {
				tt.Fatal("expected error, got none")
			}
			if !strings.Contains(err.Error(), "boom") {
				tt.Fatal(err)
			}
		})
	})

	t.Run("empty_byte_slice", func(tt *testing.T) {
		withDB(func(db *sql.DB) {
			row := db.QueryRow("select test_empty_byte_slice()")

			var a []byte
			if err := row.Scan(&a); err != nil {
				tt.Fatal(err)
			}
			if len(a) > 0 {
				tt.Fatal("expected empty byte slice")
			}
		})
	})

	t.Run("nonempty_byte_slice", func(tt *testing.T) {
		withDB(func(db *sql.DB) {
			row := db.QueryRow("select test_nonempty_byte_slice()")

			var a []byte
			if err := row.Scan(&a); err != nil {
				tt.Fatal(err)
			}
			if g, e := a, []byte("abcdefg"); !bytes.Equal(g, e) {
				tt.Fatal(string(g), string(e))
			}
		})
	})

	t.Run("empty_string", func(tt *testing.T) {
		withDB(func(db *sql.DB) {
			row := db.QueryRow("select test_empty_string()")

			var a string
			if err := row.Scan(&a); err != nil {
				tt.Fatal(err)
			}
			if len(a) > 0 {
				tt.Fatal("expected empty string")
			}
		})
	})

	t.Run("nonempty_string", func(tt *testing.T) {
		withDB(func(db *sql.DB) {
			row := db.QueryRow("select test_nonempty_string()")

			var a string
			if err := row.Scan(&a); err != nil {
				tt.Fatal(err)
			}
			if g, e := a, "abcdefg"; g != e {
				tt.Fatal(g, e)
			}
		})
	})

	t.Run("null", func(tt *testing.T) {
		withDB(func(db *sql.DB) {
			row := db.QueryRow("select test_null()")

			var a interface{}
			if err := row.Scan(&a); err != nil {
				tt.Fatal(err)
			}
			if a != nil {
				tt.Fatal("expected nil")
			}
		})
	})

	t.Run("dates", func(tt *testing.T) {
		withDB(func(db *sql.DB) {
			row := db.QueryRow("select yesterday(unixepoch('2018-11-01'))")

			var a int64
			if err := row.Scan(&a); err != nil {
				tt.Fatal(err)
			}
			if g, e := time.Unix(a, 0), time.Date(2018, time.October, 31, 0, 0, 0, 0, time.UTC); !g.Equal(e) {
				tt.Fatal(g, e)
			}
		})
	})

	t.Run("md5", func(tt *testing.T) {
		withDB(func(db *sql.DB) {
			row := db.QueryRow("select md5('abcdefg')")

			var a string
			if err := row.Scan(&a); err != nil {
				tt.Fatal(err)
			}
			if g, e := a, "7ac66c0f148de9519b8bd264312c4d64"; g != e {
				tt.Fatal(g, e)
			}
		})
	})

	t.Run("md5 with blob input", func(tt *testing.T) {
		withDB(func(db *sql.DB) {
			if _, err := db.Exec("create table t(b blob); insert into t values (?)", []byte("abcdefg")); err != nil {
				tt.Fatal(err)
			}
			row := db.QueryRow("select md5(b) from t")

			var a []byte
			if err := row.Scan(&a); err != nil {
				tt.Fatal(err)
			}
			if g, e := a, []byte("7ac66c0f148de9519b8bd264312c4d64"); !bytes.Equal(g, e) {
				tt.Fatal(string(g), string(e))
			}
		})
	})
}
