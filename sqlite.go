// Copyright 2017 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:generate go run generator.go
//go:generate gofmt -l -s -w .

package sqlite // import "modernc.org/sqlite"

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"time"
	"unsafe"

	"modernc.org/crt/v2"
	"modernc.org/sqlite/internal/bin"
)

var (
	_ driver.Conn    = (*conn)(nil)
	_ driver.Driver  = (*Driver)(nil)
	_ driver.Execer  = (*conn)(nil)
	_ driver.Queryer = (*conn)(nil)
	_ driver.Result  = (*result)(nil)
	_ driver.Rows    = (*rows)(nil)
	_ driver.Stmt    = (*stmt)(nil)
	_ driver.Tx      = (*tx)(nil)
	_ error          = (*Error)(nil)
)

const (
	driverName              = "sqlite"
	ptrSize                 = int(unsafe.Sizeof(uintptr(0)))
	sqliteLockedSharedcache = bin.DSQLITE_LOCKED | (1 << 8)
)

// Error represents sqlite library error code.
type Error struct {
	msg  string
	code int
}

// Error implements error.
func (e *Error) Error() string { return e.msg }

// Code returns the sqlite result code for this error.
func (e *Error) Code() int { return e.code }

var (
	// ErrorCodeString maps Error.Code() to its string representation.
	ErrorCodeString = map[int]string{
		bin.DSQLITE_ABORT:             "Callback routine requested an abort (SQLITE_ABORT)",
		bin.DSQLITE_AUTH:              "Authorization denied (SQLITE_AUTH)",
		bin.DSQLITE_BUSY:              "The database file is locked (SQLITE_BUSY)",
		bin.DSQLITE_CANTOPEN:          "Unable to open the database file (SQLITE_CANTOPEN)",
		bin.DSQLITE_CONSTRAINT:        "Abort due to constraint violation (SQLITE_CONSTRAINT)",
		bin.DSQLITE_CORRUPT:           "The database disk image is malformed (SQLITE_CORRUPT)",
		bin.DSQLITE_DONE:              "sqlite3_step() has finished executing (SQLITE_DONE)",
		bin.DSQLITE_EMPTY:             "Internal use only (SQLITE_EMPTY)",
		bin.DSQLITE_ERROR:             "Generic error (SQLITE_ERROR)",
		bin.DSQLITE_FORMAT:            "Not used (SQLITE_FORMAT)",
		bin.DSQLITE_FULL:              "Insertion failed because database is full (SQLITE_FULL)",
		bin.DSQLITE_INTERNAL:          "Internal logic error in SQLite (SQLITE_INTERNAL)",
		bin.DSQLITE_INTERRUPT:         "Operation terminated by sqlite3_interrupt()(SQLITE_INTERRUPT)",
		bin.DSQLITE_IOERR | (1 << 8):  "(SQLITE_IOERR_READ)",
		bin.DSQLITE_IOERR | (10 << 8): "(SQLITE_IOERR_DELETE)",
		bin.DSQLITE_IOERR | (11 << 8): "(SQLITE_IOERR_BLOCKED)",
		bin.DSQLITE_IOERR | (12 << 8): "(SQLITE_IOERR_NOMEM)",
		bin.DSQLITE_IOERR | (13 << 8): "(SQLITE_IOERR_ACCESS)",
		bin.DSQLITE_IOERR | (14 << 8): "(SQLITE_IOERR_CHECKRESERVEDLOCK)",
		bin.DSQLITE_IOERR | (15 << 8): "(SQLITE_IOERR_LOCK)",
		bin.DSQLITE_IOERR | (16 << 8): "(SQLITE_IOERR_CLOSE)",
		bin.DSQLITE_IOERR | (17 << 8): "(SQLITE_IOERR_DIR_CLOSE)",
		bin.DSQLITE_IOERR | (2 << 8):  "(SQLITE_IOERR_SHORT_READ)",
		bin.DSQLITE_IOERR | (3 << 8):  "(SQLITE_IOERR_WRITE)",
		bin.DSQLITE_IOERR | (4 << 8):  "(SQLITE_IOERR_FSYNC)",
		bin.DSQLITE_IOERR | (5 << 8):  "(SQLITE_IOERR_DIR_FSYNC)",
		bin.DSQLITE_IOERR | (6 << 8):  "(SQLITE_IOERR_TRUNCATE)",
		bin.DSQLITE_IOERR | (7 << 8):  "(SQLITE_IOERR_FSTAT)",
		bin.DSQLITE_IOERR | (8 << 8):  "(SQLITE_IOERR_UNLOCK)",
		bin.DSQLITE_IOERR | (9 << 8):  "(SQLITE_IOERR_RDLOCK)",
		bin.DSQLITE_IOERR:             "Some kind of disk I/O error occurred (SQLITE_IOERR)",
		bin.DSQLITE_LOCKED | (1 << 8): "(SQLITE_LOCKED_SHAREDCACHE)",
		bin.DSQLITE_LOCKED:            "A table in the database is locked (SQLITE_LOCKED)",
		bin.DSQLITE_MISMATCH:          "Data type mismatch (SQLITE_MISMATCH)",
		bin.DSQLITE_MISUSE:            "Library used incorrectly (SQLITE_MISUSE)",
		bin.DSQLITE_NOLFS:             "Uses OS features not supported on host (SQLITE_NOLFS)",
		bin.DSQLITE_NOMEM:             "A malloc() failed (SQLITE_NOMEM)",
		bin.DSQLITE_NOTADB:            "File opened that is not a database file (SQLITE_NOTADB)",
		bin.DSQLITE_NOTFOUND:          "Unknown opcode in sqlite3_file_control() (SQLITE_NOTFOUND)",
		bin.DSQLITE_NOTICE:            "Notifications from sqlite3_log() (SQLITE_NOTICE)",
		bin.DSQLITE_PERM:              "Access permission denied (SQLITE_PERM)",
		bin.DSQLITE_PROTOCOL:          "Database lock protocol error (SQLITE_PROTOCOL)",
		bin.DSQLITE_RANGE:             "2nd parameter to sqlite3_bind out of range (SQLITE_RANGE)",
		bin.DSQLITE_READONLY:          "Attempt to write a readonly database (SQLITE_READONLY)",
		bin.DSQLITE_ROW:               "sqlite3_step() has another row ready (SQLITE_ROW)",
		bin.DSQLITE_SCHEMA:            "The database schema changed (SQLITE_SCHEMA)",
		bin.DSQLITE_TOOBIG:            "String or BLOB exceeds size limit (SQLITE_TOOBIG)",
		bin.DSQLITE_WARNING:           "Warnings from sqlite3_log() (SQLITE_WARNING)",
	}
)

func init() {
	tls := crt.NewTLS()
	if bin.Xsqlite3_threadsafe(tls) == 0 {
		panic(fmt.Errorf("sqlite: thread safety configuration error"))
	}

	varArgs := crt.Xmalloc(tls, crt.Intptr(ptrSize))
	if varArgs == 0 {
		panic(fmt.Errorf("cannot allocate memory"))
	}

	*(*uintptr)(unsafe.Pointer(uintptr(varArgs))) = uintptr(unsafe.Pointer(&mutexMethods))
	// int sqlite3_config(int, ...);
	if rc := bin.Xsqlite3_config(tls, bin.DSQLITE_CONFIG_MUTEX, uintptr(varArgs)); rc != bin.DSQLITE_OK {
		p := bin.Xsqlite3_errstr(tls, rc)
		str := crt.GoString(p)
		panic(fmt.Errorf("sqlite: failed to configure mutex methods: %v", str))
	}

	crt.Xfree(tls, varArgs)
	sql.Register(driverName, newDriver())
}

type result struct {
	lastInsertID int64
	rowsAffected int
}

func newResult(c *conn) (_ *result, err error) {
	r := &result{}
	if r.rowsAffected, err = c.changes(); err != nil {
		return nil, err
	}

	if r.lastInsertID, err = c.lastInsertRowID(); err != nil {
		return nil, err
	}

	return r, nil
}

// LastInsertId returns the database's auto-generated ID after, for example, an
// INSERT into a table with primary key.
func (r *result) LastInsertId() (int64, error) {
	if r == nil {
		return 0, nil
	}

	return r.lastInsertID, nil
}

// RowsAffected returns the number of rows affected by the query.
func (r *result) RowsAffected() (int64, error) {
	if r == nil {
		return 0, nil
	}

	return int64(r.rowsAffected), nil
}

type rows struct {
	allocs  []crt.Intptr
	c       *conn
	columns []string
	pstmt   crt.Intptr

	doStep bool
}

func newRows(c *conn, pstmt crt.Intptr, allocs []crt.Intptr) (r *rows, err error) {
	defer func() {
		if err != nil {
			c.finalize(pstmt)
			r = nil
		}
	}()

	r = &rows{c: c, pstmt: pstmt, allocs: allocs}
	n, err := c.columnCount(pstmt)
	if err != nil {
		return nil, err
	}

	r.columns = make([]string, n)
	for i := range r.columns {
		if r.columns[i], err = r.c.columnName(pstmt, i); err != nil {
			return nil, err
		}
	}

	return r, nil
}

// Close closes the rows iterator.
func (r *rows) Close() (err error) {
	for _, v := range r.allocs {
		r.c.free(v)
	}
	r.allocs = nil
	return r.c.finalize(r.pstmt)
}

// Columns returns the names of the columns. The number of columns of the
// result is inferred from the length of the slice. If a particular column name
// isn't known, an empty string should be returned for that entry.
func (r *rows) Columns() (c []string) {
	return r.columns
}

// Next is called to populate the next row of data into the provided slice. The
// provided slice will be the same size as the Columns() are wide.
//
// Next should return io.EOF when there are no more rows.
func (r *rows) Next(dest []driver.Value) (err error) {
	rc := bin.DSQLITE_ROW
	if r.doStep {
		if rc, err = r.c.step(r.pstmt); err != nil {
			return err
		}
	}

	r.doStep = true
	switch rc {
	case bin.DSQLITE_ROW:
		if g, e := len(dest), len(r.columns); g != e {
			return fmt.Errorf("sqlite: Next: have %v destination values, expected %v", g, e)
		}

		for i := range dest {
			ct, err := r.c.columnType(r.pstmt, i)
			if err != nil {
				return err
			}

			switch ct {
			case bin.DSQLITE_INTEGER:
				v, err := r.c.columnInt64(r.pstmt, i)
				if err != nil {
					return err
				}

				dest[i] = v
			case bin.DSQLITE_FLOAT:
				v, err := r.c.columnDouble(r.pstmt, i)
				if err != nil {
					return err
				}

				dest[i] = v
			case bin.DSQLITE_TEXT:
				v, err := r.c.columnText(r.pstmt, i)
				if err != nil {
					return err
				}

				dest[i] = v
			case bin.DSQLITE_BLOB:
				v, err := r.c.columnBlob(r.pstmt, i)
				if err != nil {
					return err
				}

				dest[i] = v
			case bin.DSQLITE_NULL:
				dest[i] = nil
			default:
				return fmt.Errorf("internal error: rc %d", rc)
			}
		}
		return nil
	case bin.DSQLITE_DONE:
		return io.EOF
	default:
		return r.c.errstr(int32(rc))
	}
}

type stmt struct {
	c    *conn
	psql crt.Intptr
}

func newStmt(c *conn, sql string) (*stmt, error) {
	p, err := crt.CString(sql)
	if err != nil {
		return nil, err
	}

	return &stmt{c: c, psql: p}, nil
}

// Close closes the statement.
//
// As of Go 1.1, a Stmt will not be closed if it's in use by any queries.
func (s *stmt) Close() (err error) {
	s.c.free(s.psql)
	s.psql = 0
	return nil
}

// Exec executes a query that doesn't return rows, such as an INSERT or UPDATE.
//
//
// Deprecated: Drivers should implement StmtExecContext instead (or
// additionally).
func (s *stmt) Exec(args []driver.Value) (driver.Result, error) { //TODO StmtExecContext
	return s.exec(context.Background(), toNamedValues(args))
}

// toNamedValues converts []driver.Value to []driver.NamedValue
func toNamedValues(vals []driver.Value) (r []driver.NamedValue) {
	r = make([]driver.NamedValue, len(vals))
	for i, val := range vals {
		r[i] = driver.NamedValue{Value: val, Ordinal: i + 1}
	}
	return r
}

func (s *stmt) exec(ctx context.Context, args []driver.NamedValue) (r driver.Result, err error) {
	var pstmt crt.Intptr

	donech := make(chan struct{})

	go func() {
		select {
		case <-ctx.Done():
			if pstmt != 0 {
				s.c.interrupt(s.c.db)
			}
		case <-donech:
		}
	}()

	defer func() {
		pstmt = 0
		close(donech)
	}()

	for psql := s.psql; *(*byte)(unsafe.Pointer(uintptr(psql))) != 0; {
		if pstmt, err = s.c.prepareV2(&psql); err != nil {
			return nil, err
		}

		if pstmt == 0 {
			continue
		}

		if err := func() (err error) {
			defer func() {
				if e := s.c.finalize(pstmt); e != nil && err == nil {
					err = e
				}
			}()

			n, err := s.c.bindParameterCount(pstmt)
			if err != nil {
				return err
			}

			if n != 0 {
				allocs, err := s.c.bind(pstmt, n, args)
				if err != nil {
					return err
				}

				if len(allocs) != 0 {
					defer func() {
						for _, v := range allocs {
							s.c.free(v)
						}
					}()
				}
			}

			rc, err := s.c.step(pstmt)
			if err != nil {
				return err
			}

			switch rc & 0xff {
			case bin.DSQLITE_DONE, bin.DSQLITE_ROW:
				// nop
			default:
				return s.c.errstr(int32(rc))
			}

			return nil
		}(); err != nil {
			return nil, err
		}
	}
	return newResult(s.c)
}

// NumInput returns the number of placeholder parameters.
//
// If NumInput returns >= 0, the sql package will sanity check argument counts
// from callers and return errors to the caller before the statement's Exec or
// Query methods are called.
//
// NumInput may also return -1, if the driver doesn't know its number of
// placeholders. In that case, the sql package will not sanity check Exec or
// Query argument counts.
func (s *stmt) NumInput() (n int) {
	return -1
}

// Query executes a query that may return rows, such as a
// SELECT.
//
// Deprecated: Drivers should implement StmtQueryContext instead (or
// additionally).
func (s *stmt) Query(args []driver.Value) (driver.Rows, error) { //TODO StmtQueryContext
	return s.query(context.Background(), toNamedValues(args))
}

func (s *stmt) query(ctx context.Context, args []driver.NamedValue) (r driver.Rows, err error) {
	var pstmt crt.Intptr

	donech := make(chan struct{})

	go func() {
		select {
		case <-ctx.Done():
			if pstmt != 0 {
				s.c.interrupt(s.c.db)
			}
		case <-donech:
		}
	}()

	defer func() {
		pstmt = 0
		close(donech)
	}()

	var allocs []crt.Intptr
	for psql := s.psql; *(*byte)(unsafe.Pointer(uintptr(psql))) != 0; {
		if pstmt, err = s.c.prepareV2(&psql); err != nil {
			return nil, err
		}

		if pstmt == 0 {
			continue
		}

		if err = func() (err error) {
			defer func() {
				if e := s.c.finalize(pstmt); e != nil && err == nil {
					err = e
				}
			}()

			n, err := s.c.bindParameterCount(pstmt)
			if err != nil {
				return err
			}

			if n != 0 {
				a, err := s.c.bind(pstmt, n, args)
				if err != nil {
					return err
				}

				if len(a) != 0 {
					allocs = append(allocs, a...)
				}
			}

			rc, err := s.c.step(pstmt)
			if err != nil {
				return err
			}

			switch rc & 0xff {
			case bin.DSQLITE_ROW:
				if r, err = newRows(s.c, pstmt, allocs); err != nil {
					return err
				}

				pstmt = 0
				return nil
			case bin.DSQLITE_DONE:
				// nop
			default:
				return s.c.errstr(int32(rc))
			}

			return nil
		}(); err != nil {
			return nil, err
		}
	}
	if r != nil {
		return r, nil
	}

	panic("TODO")
}

type tx struct {
	c *conn
}

func newTx(c *conn) (*tx, error) {
	r := &tx{c: c}
	if err := r.exec(context.Background(), "begin"); err != nil {
		return nil, err
	}

	return r, nil
}

// Commit implements driver.Tx.
func (t *tx) Commit() (err error) {
	return t.exec(context.Background(), "commit")
}

// Rollback implements driver.Tx.
func (t *tx) Rollback() (err error) {
	return t.exec(context.Background(), "rollback")
}

func (t *tx) exec(ctx context.Context, sql string) (err error) {
	psql, err := crt.CString(sql)
	if err != nil {
		return err
	}

	defer t.c.free(psql)

	//TODO use t.conn.ExecContext() instead
	donech := make(chan struct{})

	defer close(donech)

	go func() {
		select {
		case <-ctx.Done():
			t.c.interrupt(t.c.db)
		case <-donech:
		}
	}()

	if rc := bin.Xsqlite3_exec(t.c.tls, t.c.db, psql, 0, 0, 0); rc != bin.DSQLITE_OK {
		return t.c.errstr(rc)
	}

	return nil
}

type conn struct {
	db  crt.Intptr // *bin.Xsqlite3
	tls *crt.TLS
}

func newConn(name string) (*conn, error) {
	c := &conn{tls: crt.NewTLS()}
	db, err := c.openV2(
		name,
		bin.DSQLITE_OPEN_READWRITE|bin.DSQLITE_OPEN_CREATE|
			bin.DSQLITE_OPEN_FULLMUTEX|
			bin.DSQLITE_OPEN_URI,
	)
	if err != nil {
		return nil, err
	}

	c.db = db
	if err = c.extendedResultCodes(true); err != nil {
		return nil, err
	}

	return c, nil
}

// const void *sqlite3_column_blob(sqlite3_stmt*, int iCol);
func (c *conn) columnBlob(pstmt crt.Intptr, iCol int) (v []byte, err error) {
	p := bin.Xsqlite3_column_blob(c.tls, pstmt, int32(iCol))
	len, err := c.columnBytes(pstmt, iCol)
	if err != nil {
		return nil, err
	}

	if p == 0 || len == 0 {
		return nil, nil
	}

	v = make([]byte, len)
	copy(v, (*crt.RawMem)(unsafe.Pointer(uintptr(p)))[:len])
	return v, nil
}

// int sqlite3_column_bytes(sqlite3_stmt*, int iCol);
func (c *conn) columnBytes(pstmt crt.Intptr, iCol int) (_ int, err error) {
	v := bin.Xsqlite3_column_bytes(c.tls, pstmt, int32(iCol))
	return int(v), nil
}

// const unsigned char *sqlite3_column_text(sqlite3_stmt*, int iCol);
func (c *conn) columnText(pstmt crt.Intptr, iCol int) (v string, err error) {
	p := bin.Xsqlite3_column_text(c.tls, pstmt, int32(iCol))
	len, err := c.columnBytes(pstmt, iCol)
	if err != nil {
		return "", err
	}

	if p == 0 || len == 0 {
		return "", nil
	}

	b := make([]byte, len)
	copy(b, (*crt.RawMem)(unsafe.Pointer(uintptr(p)))[:len])
	return string(b), nil
}

// double sqlite3_column_double(sqlite3_stmt*, int iCol);
func (c *conn) columnDouble(pstmt crt.Intptr, iCol int) (v float64, err error) {
	v = bin.Xsqlite3_column_double(c.tls, pstmt, int32(iCol))
	return v, nil
}

// sqlite3_int64 sqlite3_column_int64(sqlite3_stmt*, int iCol);
func (c *conn) columnInt64(pstmt crt.Intptr, iCol int) (v int64, err error) {
	v = bin.Xsqlite3_column_int64(c.tls, pstmt, int32(iCol))
	return v, nil
}

// int sqlite3_column_type(sqlite3_stmt*, int iCol);
func (c *conn) columnType(pstmt crt.Intptr, iCol int) (_ int, err error) {
	v := bin.Xsqlite3_column_type(c.tls, pstmt, int32(iCol))
	return int(v), nil
}

// const char *sqlite3_column_name(sqlite3_stmt*, int N);
func (c *conn) columnName(pstmt crt.Intptr, n int) (string, error) {
	p := bin.Xsqlite3_column_name(c.tls, pstmt, int32(n))
	return crt.GoString(p), nil
}

// int sqlite3_column_count(sqlite3_stmt *pStmt);
func (c *conn) columnCount(pstmt crt.Intptr) (_ int, err error) {
	v := bin.Xsqlite3_column_count(c.tls, pstmt)
	return int(v), nil
}

// sqlite3_int64 sqlite3_last_insert_rowid(sqlite3*);
func (c *conn) lastInsertRowID() (v int64, _ error) {
	return bin.Xsqlite3_last_insert_rowid(c.tls, c.db), nil
}

// int sqlite3_changes(sqlite3*);
func (c *conn) changes() (int, error) {
	v := bin.Xsqlite3_changes(c.tls, c.db)
	return int(v), nil
}

// int sqlite3_step(sqlite3_stmt*);
func (c *conn) step(pstmt crt.Intptr) (int, error) {
	for {
		switch rc := bin.Xsqlite3_step(c.tls, pstmt); rc {
		case sqliteLockedSharedcache, bin.DSQLITE_BUSY:
			if err := c.retry(pstmt); err != nil {
				return bin.DSQLITE_LOCKED, err
			}
		default:
			return int(rc), nil
		}
	}
}

func (c *conn) retry(pstmt crt.Intptr) error {
	mu := mutexAlloc(c.tls, bin.DSQLITE_MUTEX_FAST)
	(*mutex)(unsafe.Pointer(uintptr(mu))).enter(c.tls.ID) // Block
	rc := bin.Xsqlite3_unlock_notify(
		c.tls,
		c.db,
		*(*crt.Intptr)(unsafe.Pointer(&struct {
			f func(*crt.TLS, crt.Intptr, int32)
		}{unlockNotify})),
		mu,
	)
	if rc == bin.DSQLITE_LOCKED { // Deadlock, see https://www.sqlite.org/c3ref/unlock_notify.html
		(*mutex)(unsafe.Pointer(uintptr(mu))).leave() // Clear
		mutexFree(c.tls, mu)
		return c.errstr(rc)
	}

	(*mutex)(unsafe.Pointer(uintptr(mu))).enter(c.tls.ID) // Wait
	(*mutex)(unsafe.Pointer(uintptr(mu))).leave()         // Clear
	mutexFree(c.tls, mu)
	if pstmt != 0 {
		bin.Xsqlite3_reset(c.tls, pstmt)
	}
	return nil
}

func unlockNotify(t *crt.TLS, ppArg crt.Intptr, nArg int32) {
	for i := int32(0); i < nArg; i++ {
		mu := *(*crt.Intptr)(unsafe.Pointer(uintptr(ppArg)))
		(*mutex)(unsafe.Pointer(uintptr(mu))).leave() // Signal
		ppArg += crt.Intptr(ptrSize)
	}
}

func (c *conn) bind(pstmt crt.Intptr, n int, args []driver.NamedValue) (allocs []crt.Intptr, err error) {
	defer func() {
		if err == nil {
			return
		}

		for _, v := range allocs {
			c.free(v)
		}
		allocs = nil
	}()

	for i := 1; i <= n; i++ {
		var p crt.Intptr
		name, err := c.bindParameterName(pstmt, i)
		if err != nil {
			return allocs, err
		}

		var v driver.NamedValue
		for _, v = range args {
			if name != "" {
				// sqlite supports '$', '@' and ':' prefixes for string
				// identifiers and '?' for numeric, so we cannot
				// combine different prefixes with the same name
				// because `database/sql` requires variable names
				// to start with a letter
				if name[1:] == v.Name[:] {
					break
				}
			} else {
				if v.Ordinal == i {
					break
				}
			}
		}

		if v.Ordinal == 0 {
			if name != "" {
				return allocs, fmt.Errorf("missing named argument %q", name[1:])
			}

			return allocs, fmt.Errorf("missing argument with %d index", i)
		}

		switch x := v.Value.(type) {
		case int64:
			if err := c.bindInt64(pstmt, i, x); err != nil {
				return allocs, err
			}
		case float64:
			if err := c.bindDouble(pstmt, i, x); err != nil {
				return allocs, err
			}
		case bool:
			v := 0
			if x {
				v = 1
			}
			if err := c.bindInt(pstmt, i, v); err != nil {
				return allocs, err
			}
		case []byte:
			if p, err = c.bindBlob(pstmt, i, x); err != nil {
				return allocs, err
			}
		case string:
			if p, err = c.bindText(pstmt, i, x); err != nil {
				return allocs, err
			}
		case time.Time:
			if p, err = c.bindText(pstmt, i, x.String()); err != nil {
				return allocs, err
			}
		default:
			return allocs, fmt.Errorf("sqlite: invalid driver.Value type %T", x)
		}
		if p != 0 {
			allocs = append(allocs, p)
		}
	}
	return allocs, nil
}

// int sqlite3_bind_text(sqlite3_stmt*,int,const char*,int,void(*)(void*));
func (c *conn) bindText(pstmt crt.Intptr, idx1 int, value string) (crt.Intptr, error) {
	p, err := crt.CString(value)
	if err != nil {
		return 0, err
	}

	if rc := bin.Xsqlite3_bind_text(c.tls, pstmt, int32(idx1), p, int32(len(value)), 0); rc != bin.DSQLITE_OK {
		c.free(p)
		return 0, c.errstr(rc)
	}

	return p, nil
}

// int sqlite3_bind_blob(sqlite3_stmt*, int, const void*, int n, void(*)(void*));
func (c *conn) bindBlob(pstmt crt.Intptr, idx1 int, value []byte) (crt.Intptr, error) {
	p, err := c.malloc(len(value))
	if err != nil {
		return 0, err
	}

	copy((*crt.RawMem)(unsafe.Pointer(uintptr(p)))[:len(value)], value)
	if rc := bin.Xsqlite3_bind_blob(c.tls, pstmt, int32(idx1), p, int32(len(value)), 0); rc != bin.DSQLITE_OK {
		c.free(p)
		return 0, c.errstr(rc)
	}

	return p, nil
}

// int sqlite3_bind_int(sqlite3_stmt*, int, int);
func (c *conn) bindInt(pstmt crt.Intptr, idx1, value int) (err error) {
	if rc := bin.Xsqlite3_bind_int(c.tls, pstmt, int32(idx1), int32(value)); rc != bin.DSQLITE_OK {
		return c.errstr(rc)
	}

	return nil
}

// int sqlite3_bind_double(sqlite3_stmt*, int, double);
func (c *conn) bindDouble(pstmt crt.Intptr, idx1 int, value float64) (err error) {
	if rc := bin.Xsqlite3_bind_double(c.tls, pstmt, int32(idx1), value); rc != 0 {
		return c.errstr(rc)
	}

	return nil
}

// int sqlite3_bind_int64(sqlite3_stmt*, int, sqlite3_int64);
func (c *conn) bindInt64(pstmt crt.Intptr, idx1 int, value int64) (err error) {
	if rc := bin.Xsqlite3_bind_int64(c.tls, pstmt, int32(idx1), value); rc != bin.DSQLITE_OK {
		return c.errstr(rc)
	}

	return nil
}

// const char *sqlite3_bind_parameter_name(sqlite3_stmt*, int);
func (c *conn) bindParameterName(pstmt crt.Intptr, i int) (string, error) {
	p := bin.Xsqlite3_bind_parameter_name(c.tls, pstmt, int32(i))
	return crt.GoString(p), nil
}

// int sqlite3_bind_parameter_count(sqlite3_stmt*);
func (c *conn) bindParameterCount(pstmt crt.Intptr) (_ int, err error) {
	r := bin.Xsqlite3_bind_parameter_count(c.tls, pstmt)
	return int(r), nil
}

// int sqlite3_finalize(sqlite3_stmt *pStmt);
func (c *conn) finalize(pstmt crt.Intptr) error {
	if rc := bin.Xsqlite3_finalize(c.tls, pstmt); rc != bin.DSQLITE_OK {
		return c.errstr(rc)
	}

	return nil
}

// int sqlite3_prepare_v2(
//   sqlite3 *db,            /* Database handle */
//   const char *zSql,       /* SQL statement, UTF-8 encoded */
//   int nByte,              /* Maximum length of zSql in bytes. */
//   sqlite3_stmt **ppStmt,  /* OUT: Statement handle */
//   const char **pzTail     /* OUT: Pointer to unused portion of zSql */
// );
func (c *conn) prepareV2(zSql *crt.Intptr) (pstmt crt.Intptr, err error) {
	var ppstmt, pptail crt.Intptr

	defer func() {
		c.free(ppstmt)
		c.free(pptail)
	}()

	if ppstmt, err = c.malloc(ptrSize); err != nil {
		return 0, err
	}

	if pptail, err = c.malloc(ptrSize); err != nil {
		return 0, err
	}

	for {
		switch rc := bin.Xsqlite3_prepare_v2(c.tls, c.db, *zSql, -1, ppstmt, pptail); rc {
		case bin.DSQLITE_OK:
			*zSql = *(*crt.Intptr)(unsafe.Pointer(uintptr(pptail)))
			return *(*crt.Intptr)(unsafe.Pointer(uintptr(ppstmt))), nil
		case sqliteLockedSharedcache, bin.DSQLITE_BUSY:
			if err := c.retry(0); err != nil {
				return 0, err
			}
		default:
			return 0, c.errstr(rc)
		}
	}
}

// void sqlite3_interrupt(sqlite3*);
func (c *conn) interrupt(pdb crt.Intptr) (err error) {
	bin.Xsqlite3_interrupt(c.tls, pdb)
	return nil
}

// int sqlite3_extended_result_codes(sqlite3*, int onoff);
func (c *conn) extendedResultCodes(on bool) error {
	if rc := bin.Xsqlite3_extended_result_codes(c.tls, c.db, crt.Bool32(on)); rc != bin.DSQLITE_OK {
		return c.errstr(rc)
	}

	return nil
}

// int sqlite3_open_v2(
//   const char *filename,   /* Database filename (UTF-8) */
//   sqlite3 **ppDb,         /* OUT: SQLite db handle */
//   int flags,              /* Flags */
//   const char *zVfs        /* Name of VFS module to use */
// );
func (c *conn) openV2(name string, flags int32) (crt.Intptr, error) {
	var p, s crt.Intptr

	defer func() {
		if p != 0 {
			c.free(p)
		}
		if s != 0 {
			c.free(s)
		}
	}()

	p, err := c.malloc(ptrSize)
	if err != nil {
		return 0, err
	}

	if s, err = crt.CString(name); err != nil {
		return 0, err
	}

	if rc := bin.Xsqlite3_open_v2(c.tls, s, p, flags, 0); rc != bin.DSQLITE_OK {
		return 0, c.errstr(rc)
	}

	return *(*crt.Intptr)(unsafe.Pointer(uintptr(p))), nil
}

func (c *conn) malloc(n int) (crt.Intptr, error) {
	if p := crt.Xmalloc(c.tls, crt.Intptr(n)); p != 0 {
		return p, nil
	}

	return 0, fmt.Errorf("sqlite: cannot allocate %d bytes of memory", n)
}

func (c *conn) free(p crt.Intptr) {
	if p != 0 {
		crt.Xfree(c.tls, p)
	}
}

// const char *sqlite3_errstr(int);
func (c *conn) errstr(rc int32) error {
	p := bin.Xsqlite3_errstr(c.tls, rc)
	str := crt.GoString(p)
	p = bin.Xsqlite3_errmsg(c.tls, c.db)
	switch msg := crt.GoString(p); {
	case msg == str:
		return &Error{msg: fmt.Sprintf("%s (%v)", str, rc), code: int(rc)}
	default:
		return &Error{msg: fmt.Sprintf("%s: %s (%v)", str, msg, rc), code: int(rc)}
	}
}

// Begin starts a transaction.
//
// Deprecated: Drivers should implement ConnBeginTx instead (or additionally).
func (c *conn) Begin() (driver.Tx, error) {
	return c.begin(context.Background(), driver.TxOptions{})
}

func (c *conn) begin(ctx context.Context, opts driver.TxOptions) (t driver.Tx, err error) {
	return newTx(c)
}

// Close invalidates and potentially stops any current prepared statements and
// transactions, marking this connection as no longer in use.
//
// Because the sql package maintains a free pool of connections and only calls
// Close when there's a surplus of idle connections, it shouldn't be necessary
// for drivers to do their own connection caching.
func (c *conn) Close() error {
	if c.db != 0 {
		if err := c.closeV2(c.db); err != nil {
			return err
		}

		c.db = 0
	}
	return nil
}

// int sqlite3_close_v2(sqlite3*);
func (c *conn) closeV2(db crt.Intptr) error {
	if rc := bin.Xsqlite3_close_v2(c.tls, db); rc != bin.DSQLITE_OK {
		return c.errstr(rc)
	}

	return nil
}

// Execer is an optional interface that may be implemented by a Conn.
//
// If a Conn does not implement Execer, the sql package's DB.Exec will first
// prepare a query, execute the statement, and then close the statement.
//
// Exec may return ErrSkip.
//
// Deprecated: Drivers should implement ExecerContext instead.
func (c *conn) Exec(query string, args []driver.Value) (driver.Result, error) {
	return c.exec(context.Background(), query, toNamedValues(args))
}

func (c *conn) exec(ctx context.Context, query string, args []driver.NamedValue) (r driver.Result, err error) {
	s, err := c.prepare(ctx, query)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err2 := s.Close(); err2 != nil && err == nil {
			err = err2
		}
	}()

	return s.(*stmt).exec(ctx, args)
}

// Prepare returns a prepared statement, bound to this connection.
func (c *conn) Prepare(query string) (driver.Stmt, error) {
	return c.prepare(context.Background(), query)
}

func (c *conn) prepare(ctx context.Context, query string) (s driver.Stmt, err error) {
	//TODO use ctx
	return newStmt(c, query)
}

// Queryer is an optional interface that may be implemented by a Conn.
//
// If a Conn does not implement Queryer, the sql package's DB.Query will first
// prepare a query, execute the statement, and then close the statement.
//
// Query may return ErrSkip.
//
// Deprecated: Drivers should implement QueryerContext instead.
func (c *conn) Query(query string, args []driver.Value) (driver.Rows, error) {
	return c.query(context.Background(), query, toNamedValues(args))
}

func (c *conn) query(ctx context.Context, query string, args []driver.NamedValue) (r driver.Rows, err error) {
	s, err := c.prepare(ctx, query)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err2 := s.Close(); err2 != nil && err == nil {
			err = err2
		}
	}()

	return s.(*stmt).query(ctx, args)
}

// Driver implements database/sql/driver.Driver.
type Driver struct{}

func newDriver() *Driver { return &Driver{} }

// Open returns a new connection to the database.  The name is a string in a
// driver-specific format.
//
// Open may return a cached connection (one previously closed), but doing so is
// unnecessary; the sql package maintains a pool of idle connections for
// efficient re-use.
//
// The returned connection is only used by one goroutine at a time.
func (d *Driver) Open(name string) (driver.Conn, error) {
	return newConn(name)
}
