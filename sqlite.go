// Copyright 2017 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlite

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"math"
	"os"
	"runtime"
	"sync"
	"time"
	"unsafe"

	"github.com/cznic/crt"
	"github.com/cznic/sqlite/internal/bin"
	"golang.org/x/net/context"
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
)

const (
	driverName  = "sqlite"
	heapReserve = 1 << 20
	heapSize    = 32 << 20
	ptrSize     = 1 << (^uintptr(0)>>32&1 + ^uintptr(0)>>16&1 + ^uintptr(0)>>8&1 + 3) / 8
)

func init() {
	if v := bin.Init(heapSize, heapReserve); v != 0 {
		panic(fmt.Errorf("initialization failed: %v", v))
	}

	sql.Register(driverName, newDrv())
}

func tracer(rx interface{}, format string, args ...interface{}) {
	var b bytes.Buffer
	_, file, line, _ := runtime.Caller(1)
	fmt.Fprintf(&b, "%v:%v: (%[3]T)(%[3]p).", file, line, rx)
	fmt.Fprintf(&b, format, args...)
	fmt.Fprintf(os.Stderr, "%s\n", b.Bytes())
}

type result struct {
	*stmt
	lastInsertId int64
	rowsAffected int
}

func (r *result) String() string {
	return fmt.Sprintf("&%T@%p{stmt: %p, LastInsertId: %v, RowsAffected: %v}", *r, r, r.stmt, r.lastInsertId, r.rowsAffected)
}

func newResult(s *stmt) (_ *result, err error) {
	r := &result{stmt: s}
	if r.rowsAffected, err = r.changes(); err != nil {
		return nil, err
	}

	if r.lastInsertId, err = r.lastInsertRowID(); err != nil {
		return nil, err
	}

	return r, nil
}

// sqlite3_int64 sqlite3_last_insert_rowid(sqlite3*);
func (r *result) lastInsertRowID() (v int64, _ error) {
	return bin.Xsqlite3_last_insert_rowid(r.tls, r.pdb()), nil
}

// int sqlite3_changes(sqlite3*);
func (r *result) changes() (int, error) {
	v := bin.Xsqlite3_changes(r.tls, r.pdb())
	return int(v), nil
}

// LastInsertId returns the database's auto-generated ID after, for example, an
// INSERT into a table with primary key.
func (r *result) LastInsertId() (int64, error) {
	if r == nil {
		return 0, nil
	}

	return r.lastInsertId, nil
}

// RowsAffected returns the number of rows affected by the query.
func (r *result) RowsAffected() (int64, error) {
	if r == nil {
		return 0, nil
	}

	return int64(r.rowsAffected), nil
}

type rows struct {
	*stmt
	columns []string
	rc0     int
	pstmt   unsafe.Pointer
	doStep  bool
}

func (r *rows) String() string {
	return fmt.Sprintf("&%T@%p{stmt: %p, columns: %v, rc0: %v, pstmt: %#x, doStep: %v}", *r, r, r.stmt, r.columns, r.rc0, r.pstmt, r.doStep)
}

func newRows(s *stmt, pstmt unsafe.Pointer, rc0 int) (*rows, error) {
	r := &rows{
		stmt:  s,
		pstmt: pstmt,
		rc0:   rc0,
	}

	n, err := r.columnCount()
	if err != nil {
		return nil, err
	}

	r.columns = make([]string, n)
	for i := range r.columns {
		if r.columns[i], err = r.columnName(i); err != nil {
			return nil, err
		}
	}

	return r, nil
}

// Columns returns the names of the columns. The number of columns of the
// result is inferred from the length of the slice. If a particular column name
// isn't known, an empty string should be returned for that entry.
func (r *rows) Columns() (c []string) {
	if trace {
		defer func() {
			tracer(r, "Columns(): %v", c)
		}()
	}
	return r.columns
}

// Close closes the rows iterator.
func (r *rows) Close() (err error) {
	if trace {
		defer func() {
			tracer(r, "Close(): %v", err)
		}()
	}
	return r.finalize(r.pstmt)
}

// Next is called to populate the next row of data into the provided slice. The
// provided slice will be the same size as the Columns() are wide.
//
// Next should return io.EOF when there are no more rows.
func (r *rows) Next(dest []driver.Value) (err error) {
	if trace {
		defer func() {
			tracer(r, "Next(%v): %v", dest, err)
		}()
	}
	rc := r.rc0
	if r.doStep {
		if rc, err = r.step(r.pstmt); err != nil {
			return err
		}
	}

	r.doStep = true

	switch rc {
	case bin.XSQLITE_ROW:
		if g, e := len(dest), len(r.columns); g != e {
			return fmt.Errorf("Next(): have %v destination values, expected %v", g, e)
		}

		for i := range dest {
			ct, err := r.columnType(i)
			if err != nil {
				return err
			}

			switch ct {
			case bin.XSQLITE_INTEGER:
				v, err := r.columnInt64(i)
				if err != nil {
					return err
				}

				dest[i] = v
			case bin.XSQLITE_FLOAT:
				v, err := r.columnDouble(i)
				if err != nil {
					return err
				}

				dest[i] = v
			case bin.XSQLITE_TEXT:
				v, err := r.columnText(i)
				if err != nil {
					return err
				}

				dest[i] = v
			case bin.XSQLITE_BLOB:
				v, err := r.columnBlob(i)
				if err != nil {
					return err
				}

				dest[i] = v
			case bin.XSQLITE_NULL:
				dest[i] = nil
			default:
				panic("internal error")
			}
		}
		return nil
	case bin.XSQLITE_DONE:
		return io.EOF
	default:
		return r.errstr(int32(rc))
	}
}

// int sqlite3_column_bytes(sqlite3_stmt*, int iCol);
func (r *rows) columnBytes(iCol int) (_ int, err error) {
	v := bin.Xsqlite3_column_bytes(r.tls, r.pstmt, int32(iCol))
	return int(v), nil
}

// const void *sqlite3_column_blob(sqlite3_stmt*, int iCol);
func (r *rows) columnBlob(iCol int) (v []byte, err error) {
	p := bin.Xsqlite3_column_blob(r.tls, r.pstmt, int32(iCol))
	len, err := r.columnBytes(iCol)
	if err != nil {
		return nil, err
	}

	return crt.GoBytesLen((*int8)(p), len), nil
}

// const unsigned char *sqlite3_column_text(sqlite3_stmt*, int iCol);
func (r *rows) columnText(iCol int) (v string, err error) {
	p := bin.Xsqlite3_column_text(r.tls, r.pstmt, int32(iCol))
	len, err := r.columnBytes(iCol)
	if err != nil {
		return "", err
	}

	return crt.GoStringLen((*int8)(unsafe.Pointer(p)), len), nil
}

// double sqlite3_column_double(sqlite3_stmt*, int iCol);
func (r *rows) columnDouble(iCol int) (v float64, err error) {
	v = bin.Xsqlite3_column_double(r.tls, r.pstmt, int32(iCol))
	return v, nil
}

// sqlite3_int64 sqlite3_column_int64(sqlite3_stmt*, int iCol);
func (r *rows) columnInt64(iCol int) (v int64, err error) {
	v = bin.Xsqlite3_column_int64(r.tls, r.pstmt, int32(iCol))
	return v, nil
}

// int sqlite3_column_type(sqlite3_stmt*, int iCol);
func (r *rows) columnType(iCol int) (_ int, err error) {
	v := bin.Xsqlite3_column_type(r.tls, r.pstmt, int32(iCol))
	return int(v), nil
}

// int sqlite3_column_count(sqlite3_stmt *pStmt);
func (r *rows) columnCount() (_ int, err error) {
	v := bin.Xsqlite3_column_count(r.tls, r.pstmt)
	return int(v), nil
}

// const char *sqlite3_column_name(sqlite3_stmt*, int N);
func (r *rows) columnName(n int) (string, error) {
	p := bin.Xsqlite3_column_name(r.tls, r.pstmt, int32(n))
	return crt.GoString(p), nil
}

type stmt struct {
	*conn
	allocs []unsafe.Pointer
	psql   *int8
	ppstmt *unsafe.Pointer
	pzTail **int8
}

func (s *stmt) String() string {
	return fmt.Sprintf("&%T@%p{conn: %p, alloc %v, psql: %#x, ppstmt: %#x, pzTail: %#x}", *s, s, s.conn, s.allocs, s.psql, s.ppstmt, s.pzTail)
}

func newStmt(c *conn, sql string) (*stmt, error) {
	s := &stmt{conn: c}
	psql, err := s.cString(sql)
	if err != nil {
		return nil, err
	}

	s.psql = psql
	ppstmt, err := s.malloc(ptrSize)
	if err != nil {
		s.free(unsafe.Pointer(psql))
		return nil, err
	}

	s.ppstmt = (*unsafe.Pointer)(ppstmt)
	pzTail, err := s.malloc(ptrSize)
	if err != nil {
		s.free(unsafe.Pointer(psql))
		s.free(ppstmt)
		return nil, err
	}

	s.pzTail = (**int8)(pzTail)
	return s, nil
}

// Close closes the statement.
//
// As of Go 1.1, a Stmt will not be closed if it's in use by any queries.
func (s *stmt) Close() (err error) {
	if trace {
		defer func() {
			tracer(s, "Close(): %v", err)
		}()
	}
	if s.psql != nil {
		err = s.free(unsafe.Pointer(s.psql))
		s.psql = nil
	}
	if s.ppstmt != nil {
		if err2 := s.free(unsafe.Pointer(s.ppstmt)); err2 != nil && err == nil {
			err = err2
		}
		s.ppstmt = nil
	}
	if s.pzTail != nil {
		if err2 := s.free(unsafe.Pointer(s.pzTail)); err2 != nil && err == nil {
			err = err2
		}
		s.pzTail = nil
	}
	for _, v := range s.allocs {
		if err2 := s.free(v); err2 != nil && err == nil {
			err = err2
		}
	}
	s.allocs = nil
	return err
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
	if trace {
		defer func() {
			tracer(s, "NumInput(): %v", n)
		}()
	}
	return -1
}

// Exec executes a query that doesn't return rows, such as an INSERT or UPDATE.
func (s *stmt) Exec(args []driver.Value) (driver.Result, error) {
	return s.exec(context.Background(), toNamedValues(args))
}

func (s *stmt) exec(ctx context.Context, args []namedValue) (r driver.Result, err error) {
	if trace {
		defer func(args []namedValue) {
			tracer(s, "Exec(%v): (%v, %v)", args, r, err)
		}(args)
	}

	var pstmt unsafe.Pointer

	donech := make(chan struct{})
	defer close(donech)
	go func() {
		select {
		case <-ctx.Done():
			if pstmt != nil {
				s.interrupt(s.pdb())
			}
		case <-donech:
		}
	}()

	for psql := s.psql; *psql != 0; psql = *s.pzTail {
		if err := s.prepareV2(psql); err != nil {
			return nil, err
		}

		pstmt = *s.ppstmt
		if pstmt == nil {
			continue
		}

		n, err := s.bindParameterCount(pstmt)
		if err != nil {
			return nil, err
		}

		if n != 0 {
			if err = s.bind(pstmt, n, args); err != nil {
				return nil, err
			}
		}

		rc, err := s.step(pstmt)
		if err != nil {
			s.finalize(pstmt)
			return nil, err
		}

		switch rc & 0xff {
		case bin.XSQLITE_DONE, bin.XSQLITE_ROW:
			if err := s.finalize(pstmt); err != nil {
				return nil, err
			}
		default:
			err = s.errstr(int32(rc))
			s.finalize(pstmt)
			return nil, err
		}
	}
	return newResult(s)
}

func (s *stmt) Query(args []driver.Value) (driver.Rows, error) {
	return s.query(context.Background(), toNamedValues(args))
}

func (s *stmt) query(ctx context.Context, args []namedValue) (r driver.Rows, err error) {
	if trace {
		defer func(args []namedValue) {
			tracer(s, "Query(%v): (%v, %v)", args, r, err)
		}(args)
	}

	var pstmt, rowStmt unsafe.Pointer
	var rc0 int

	donech := make(chan struct{})
	defer close(donech)
	go func() {
		select {
		case <-ctx.Done():
			if pstmt != nil {
				s.interrupt(s.pdb())
			}
		case <-donech:
		}
	}()

	for psql := s.psql; *psql != 0; psql = *s.pzTail {
		if err := s.prepareV2(psql); err != nil {
			return nil, err
		}

		pstmt = *s.ppstmt
		if pstmt == nil {
			continue
		}

		n, err := s.bindParameterCount(pstmt)
		if err != nil {
			return nil, err
		}

		if n != 0 {
			if err = s.bind(pstmt, n, args); err != nil {
				return nil, err
			}
		}

		rc, err := s.step(pstmt)
		if err != nil {
			s.finalize(pstmt)
			return nil, err
		}

		switch rc {
		case bin.XSQLITE_ROW:
			if rowStmt != nil {
				if err := s.finalize(pstmt); err != nil {
					return nil, err
				}

				return nil, fmt.Errorf("query contains multiple select statements")
			}

			rowStmt = pstmt
			rc0 = rc
		case bin.XSQLITE_DONE:
			if rowStmt == nil {
				rc0 = rc
			}
		default:
			err = s.errstr(int32(rc))
			s.finalize(pstmt)
			return nil, err
		}
	}
	return newRows(s, rowStmt, rc0)
}

// int sqlite3_bind_double(sqlite3_stmt*, int, double);
func (s *stmt) bindDouble(pstmt unsafe.Pointer, idx1 int, value float64) (err error) {
	if rc := bin.Xsqlite3_bind_double(s.tls, pstmt, int32(idx1), value); rc != 0 {
		return s.errstr(rc)
	}

	return nil
}

// int sqlite3_bind_int(sqlite3_stmt*, int, int);
func (s *stmt) bindInt(pstmt unsafe.Pointer, idx1, value int) (err error) {
	if rc := bin.Xsqlite3_bind_int(s.tls, pstmt, int32(idx1), int32(value)); rc != bin.XSQLITE_OK {
		return s.errstr(rc)
	}

	return nil
}

// int sqlite3_bind_int64(sqlite3_stmt*, int, sqlite3_int64);
func (s *stmt) bindInt64(pstmt unsafe.Pointer, idx1 int, value int64) (err error) {
	if rc := bin.Xsqlite3_bind_int64(s.tls, pstmt, int32(idx1), value); rc != bin.XSQLITE_OK {
		return s.errstr(rc)
	}

	return nil
}

// int sqlite3_bind_blob(sqlite3_stmt*, int, const void*, int n, void(*)(void*));
func (s *stmt) bindBlob(pstmt unsafe.Pointer, idx1 int, value []byte) (err error) {
	p, err := s.malloc(len(value))
	if err != nil {
		return err
	}

	s.allocs = append(s.allocs, p)
	crt.CopyBytes(p, value, false)
	if rc := bin.Xsqlite3_bind_blob(s.tls, pstmt, int32(idx1), p, int32(len(value)), nil); rc != bin.XSQLITE_OK {
		return s.errstr(rc)
	}

	return nil
}

// int sqlite3_bind_text(sqlite3_stmt*,int,const char*,int,void(*)(void*));
func (s *stmt) bindText(pstmt unsafe.Pointer, idx1 int, value string) (err error) {
	p, err := s.cString(value)
	if err != nil {
		return err
	}

	s.allocs = append(s.allocs, unsafe.Pointer(p))
	if rc := bin.Xsqlite3_bind_text(s.tls, pstmt, int32(idx1), p, int32(len(value)), nil); rc != bin.XSQLITE_OK {
		return s.errstr(rc)
	}

	return nil
}

func (s *stmt) bind(pstmt unsafe.Pointer, n int, args []namedValue) error {
	for i := 1; i <= n; i++ {
		name, err := s.bindParameterName(pstmt, i)
		if err != nil {
			return err
		}

		var v namedValue
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
				return fmt.Errorf("missing named argument %q", name[1:])
			}

			return fmt.Errorf("missing argument with %d index", i)
		}

		switch x := v.Value.(type) {
		case int64:
			if err := s.bindInt64(pstmt, i, x); err != nil {
				return err
			}
		case float64:
			if err := s.bindDouble(pstmt, i, x); err != nil {
				return err
			}
		case bool:
			v := 0
			if x {
				v = 1
			}
			if err := s.bindInt(pstmt, i, v); err != nil {
				return err
			}
		case []byte:
			if err := s.bindBlob(pstmt, i, x); err != nil {
				return err
			}
		case string:
			if err := s.bindText(pstmt, i, x); err != nil {
				return err
			}
		case time.Time:
			if err := s.bindText(pstmt, i, x.String()); err != nil {
				return err
			}
		default:
			return fmt.Errorf("invalid driver.Value type %T", x)
		}
	}
	return nil
}

// int sqlite3_bind_parameter_count(sqlite3_stmt*);
func (s *stmt) bindParameterCount(pstmt unsafe.Pointer) (_ int, err error) {
	r := bin.Xsqlite3_bind_parameter_count(s.tls, pstmt)
	return int(r), nil
}

// const char *sqlite3_bind_parameter_name(sqlite3_stmt*, int);
func (s *stmt) bindParameterName(pstmt unsafe.Pointer, i int) (string, error) {
	p := bin.Xsqlite3_bind_parameter_name(s.tls, pstmt, int32(i))
	return crt.GoString(p), nil
}

// int sqlite3_finalize(sqlite3_stmt *pStmt);
func (s *stmt) finalize(pstmt unsafe.Pointer) error {
	if rc := bin.Xsqlite3_finalize(s.tls, pstmt); rc != bin.XSQLITE_OK {
		return s.errstr(rc)
	}

	return nil
}

// int sqlite3_step(sqlite3_stmt*);
func (s *stmt) step(pstmt unsafe.Pointer) (int, error) {
	r := bin.Xsqlite3_step(s.tls, pstmt)
	return int(r), nil
}

// int sqlite3_prepare_v2(
//   sqlite3 *db,            /* Database handle */
//   const char *zSql,       /* SQL statement, UTF-8 encoded */
//   int nByte,              /* Maximum length of zSql in bytes. */
//   sqlite3_stmt **ppStmt,  /* OUT: Statement handle */
//   const char **pzTail     /* OUT: Pointer to unused portion of zSql */
// );
func (s *stmt) prepareV2(zSql *int8) error {
	if rc := bin.Xsqlite3_prepare_v2(s.tls, s.pdb(), zSql, -1, s.ppstmt, s.pzTail); rc != bin.XSQLITE_OK {
		return s.errstr(rc)
	}

	return nil
}

type tx struct {
	*conn
}

func (t *tx) String() string { return fmt.Sprintf("&%T@%p{conn: %p}", *t, t, t.conn) }

func newTx(c *conn) (*tx, error) {
	t := &tx{conn: c}
	if err := t.exec(context.Background(), "begin"); err != nil {
		return nil, err
	}

	return t, nil
}

// Commit implements driver.Tx.
func (t *tx) Commit() (err error) {
	if trace {
		defer func() {
			tracer(t, "Commit(): %v", err)
		}()
	}
	return t.exec(context.Background(), "commit")
}

// Rollback implements driver.Tx.
func (t *tx) Rollback() (err error) {
	if trace {
		defer func() {
			tracer(t, "Rollback(): %v", err)
		}()
	}
	return t.exec(context.Background(), "rollback")
}

// int sqlite3_exec(
//   sqlite3*,                                  /* An open database */
//   const char *sql,                           /* SQL to be evaluated */
//   int (*callback)(void*,int,char**,char**),  /* Callback function */
//   void *,                                    /* 1st argument to callback */
//   char **errmsg                              /* Error msg written here */
// );
func (t *tx) exec(ctx context.Context, sql string) (err error) {
	psql, err := t.cString(sql)
	if err != nil {
		return err
	}

	defer t.free(unsafe.Pointer(psql))

	// TODO: use t.conn.ExecContext() instead
	donech := make(chan struct{})
	defer close(donech)
	go func() {
		select {
		case <-ctx.Done():
			t.interrupt(t.pdb())
		case <-donech:
		}
	}()

	if rc := bin.Xsqlite3_exec(t.tls, t.pdb(), psql, nil, nil, nil); rc != bin.XSQLITE_OK {
		return t.errstr(rc)
	}

	return nil
}

type conn struct {
	*Driver
	ppdb **bin.Xsqlite3
	tls  *crt.TLS
}

func (c *conn) String() string {
	return fmt.Sprintf("&%T@%p{sqlite: %p, Thread: %p, ppdb: %#x}", *c, c, c.Driver, c.tls, c.ppdb)
}

func newConn(s *Driver, name string) (_ *conn, err error) {
	c := &conn{Driver: s}

	defer func() {
		if err != nil {
			c.close()
		}
	}()

	c.Lock()

	defer c.Unlock()

	c.tls = crt.NewTLS()
	if err = c.openV2(
		name,
		bin.XSQLITE_OPEN_READWRITE|bin.XSQLITE_OPEN_CREATE|
			bin.XSQLITE_OPEN_FULLMUTEX|
			bin.XSQLITE_OPEN_URI,
	); err != nil {
		return nil, err
	}

	if err = c.extendedResultCodes(true); err != nil {
		return nil, err
	}

	return c, nil
}

// Prepare returns a prepared statement, bound to this connection.
func (c *conn) Prepare(query string) (s driver.Stmt, err error) {
	return c.prepare(context.Background(), query)
}

func (c *conn) prepare(ctx context.Context, query string) (s driver.Stmt, err error) {
	if trace {
		defer func() {
			tracer(c, "Prepare(%s): (%v, %v)", query, s, err)
		}()
	}
	return newStmt(c, query)
}

// Close invalidates and potentially stops any current prepared statements and
// transactions, marking this connection as no longer in use.
//
// Because the sql package maintains a free pool of connections and only calls
// Close when there's a surplus of idle connections, it shouldn't be necessary
// for drivers to do their own connection caching.
func (c *conn) Close() (err error) {
	if trace {
		defer func() {
			tracer(c, "Close(): %v", err)
		}()
	}
	return c.close()
}

// Begin starts a transaction.
func (c *conn) Begin() (driver.Tx, error) {
	return c.begin(context.Background(), txOptions{})
}

// copy of driver.TxOptions
type txOptions struct {
	Isolation int // driver.IsolationLevel
	ReadOnly  bool
}

func (c *conn) begin(ctx context.Context, opts txOptions) (t driver.Tx, err error) {
	if trace {
		defer func() {
			tracer(c, "BeginTx(): (%v, %v)", t, err)
		}()
	}
	return newTx(c)
}

// Execer is an optional interface that may be implemented by a Conn.
//
// If a Conn does not implement Execer, the sql package's DB.Exec will first
// prepare a query, execute the statement, and then close the statement.
//
// Exec may return ErrSkip.
func (c *conn) Exec(query string, args []driver.Value) (driver.Result, error) {
	return c.exec(context.Background(), query, toNamedValues(args))
}

func (c *conn) exec(ctx context.Context, query string, args []namedValue) (r driver.Result, err error) {
	if trace {
		defer func() {
			tracer(c, "ExecContext(%s, %v): (%v, %v)", query, args, r, err)
		}()
	}

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

// copy of driver.NameValue
type namedValue struct {
	Name    string
	Ordinal int
	Value   driver.Value
}

// toNamedValues converts []driver.Value to []namedValue
func toNamedValues(vals []driver.Value) []namedValue {
	args := make([]namedValue, 0, len(vals))
	for i, val := range vals {
		args = append(args, namedValue{Value: val, Ordinal: i + 1})
	}
	return args
}

// Queryer is an optional interface that may be implemented by a Conn.
//
// If a Conn does not implement Queryer, the sql package's DB.Query will first
// prepare a query, execute the statement, and then close the statement.
//
// Query may return ErrSkip.
func (c *conn) Query(query string, args []driver.Value) (driver.Rows, error) {
	return c.query(context.Background(), query, toNamedValues(args))
}

func (c *conn) query(ctx context.Context, query string, args []namedValue) (r driver.Rows, err error) {
	if trace {
		defer func() {
			tracer(c, "Query(%s, %v): (%v, %v)", query, args, r, err)
		}()
	}
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

func (c *conn) pdb() *bin.Xsqlite3 { return *c.ppdb }

// int sqlite3_extended_result_codes(sqlite3*, int onoff);
func (c *conn) extendedResultCodes(on bool) (err error) {
	var v int32
	if on {
		v = 1
	}
	if rc := bin.Xsqlite3_extended_result_codes(c.tls, c.pdb(), v); rc != bin.XSQLITE_OK {
		return c.errstr(rc)
	}

	return nil
}

// void *sqlite3_malloc(int);
func (c *conn) malloc(n int) (r unsafe.Pointer, err error) {
	if n > math.MaxInt32 {
		panic("internal error")
	}

	r = bin.Xsqlite3_malloc(c.tls, int32(n))
	if r == nil {
		return nil, fmt.Errorf("malloc(%v) failed", n)
	}

	return r, nil
}

func (c *conn) cString(s string) (*int8, error) {
	n := len(s)
	p, err := c.malloc(n + 1)
	if err != nil {
		return nil, err
	}

	crt.CopyString(p, s, true)
	return (*int8)(p), nil
}

// int sqlite3_open_v2(
//   const char *filename,   /* Database filename (UTF-8) */
//   sqlite3 **ppDb,         /* OUT: SQLite db handle */
//   int flags,              /* Flags */
//   const char *zVfs        /* Name of VFS module to use */
// );
func (c *conn) openV2(name string, flags int32) error {
	filename, err := c.cString(name)
	if err != nil {
		return err
	}

	defer c.free(unsafe.Pointer(filename))

	ppdb, err := c.malloc(ptrSize)
	if err != nil {
		return err
	}

	c.ppdb = (**bin.Xsqlite3)(ppdb)
	if rc := bin.Xsqlite3_open_v2(c.tls, filename, c.ppdb, flags, nil); rc != bin.XSQLITE_OK {
		return c.errstr(rc)
	}

	return nil
}

// const char *sqlite3_errstr(int);
func (c *conn) errstr(rc int32) (err error) {
	p := bin.Xsqlite3_errstr(c.tls, rc)
	str := crt.GoString(p)
	p = bin.Xsqlite3_errmsg(c.tls, c.pdb())

	switch msg := crt.GoString(p); {
	case msg == str:
		return fmt.Errorf("%s (%v)", str, rc)
	default:
		return fmt.Errorf("%s: %s (%v)", str, msg, rc)
	}
}

// int sqlite3_close_v2(sqlite3*);
func (c *conn) closeV2() (err error) {
	if rc := bin.Xsqlite3_close_v2(c.tls, c.pdb()); rc != bin.XSQLITE_OK {
		return c.errstr(rc)
	}

	err = c.free(unsafe.Pointer(c.ppdb))
	c.ppdb = nil
	return err
}

// void sqlite3_free(void*);
func (c *conn) free(p unsafe.Pointer) (err error) {
	bin.Xsqlite3_free(c.tls, p)
	return nil
}

// void sqlite3_interrupt(sqlite3*);
func (c *conn) interrupt(pdb *bin.Xsqlite3) (err error) {
	bin.Xsqlite3_interrupt(c.tls, pdb)
	return nil
}

func (c *conn) close() (err error) {
	c.Lock()

	defer c.Unlock()

	if c.ppdb != nil {
		err = c.closeV2()
	}
	return err
}

type Driver struct {
	sync.Mutex
}

func newDrv() *Driver { return &Driver{} }

// Open returns a new connection to the database.  The name is a string in a
// driver-specific format.
//
// Open may return a cached connection (one previously closed), but doing so is
// unnecessary; the sql package maintains a pool of idle connections for
// efficient re-use.
//
// The returned connection is only used by one goroutine at a time.
func (s *Driver) Open(name string) (c driver.Conn, err error) {
	if trace {
		defer func() {
			tracer(s, "Open(%s): (%v, %v)", name, c, err)
		}()
	}
	return newConn(s, name)
}
