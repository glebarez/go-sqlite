// Copyright 2017 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//TODO Pinger (tags go1.8)

//go:generate go run generator.go

// Package sqlite is an in-process implementation of a self-contained,
// serverless, zero-configuration, transactional SQL database engine. (Work In Progress)
//
// Connecting to a database
//
// To access a Sqlite database do something like
//
//	import (
//		"database/sql"
//
//		_ "github.com/cznic/sqlite"
//	)
//
//	...
//
//
//	db, err := sql.Open("sqlite", dsnURI)
//
//	...
//
//
// Do not use in production
//
// This is an experimental, pre-alpha, technology preview package. Performance
// is not (yet) a priority. When this virtual machine approach, hopefully,
// reaches a reasonable level of completeness and correctness, the plan is to
// eventually mechanically translate the IR form, produced by
// http://github.com/cznic/ccir, to Go. Unreadable Go, presumably.
//
// Supported platforms and architectures
//
// See http://github.com/cznic/ccir. To add a newly supported os/arch
// combination to this package try running 'go generate'.
//
// Sqlite documentation
//
// See https://sqlite.org/docs.html
package sqlite

import (
	"bytes"
	"compress/gzip"
	"database/sql"
	"database/sql/driver"
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime"
	"sync"
	"time"
	"unsafe"

	"github.com/cznic/internal/buffer"
	"github.com/cznic/ir"
	"github.com/cznic/mathutil"
	"github.com/cznic/sqlite/internal/bin"
	"github.com/cznic/virtual"
	"github.com/cznic/xc"
)

var (
	_ driver.Conn    = (*conn)(nil)
	_ driver.Driver  = (*sqlite)(nil)
	_ driver.Execer  = (*conn)(nil)
	_ driver.Queryer = (*conn)(nil)
	_ driver.Result  = (*result)(nil)
	_ driver.Rows    = (*rows)(nil)
	_ driver.Stmt    = (*stmt)(nil)
	_ driver.Tx      = (*tx)(nil)
	_ io.Writer      = debugWriter{}
)

const (
	driverName      = "sqlite"
	ptrSize         = mathutil.UintPtrBits / 8
	vmHeapReserve   = 1 << 20
	vmHeapSize      = 32 << 20
	vmMainStackSize = 1 << 16
	vmStackSize     = 1 << 18
)

var (
	binary virtual.Binary
	dict   = xc.Dict
	null   = virtual.Ptr(0)
	vm     *virtual.Machine

	// FFI
	bindBlob            int
	bindDouble          int
	bindInt             int
	bindInt64           int
	bindParameterCount  int
	bindText            int
	changes             int
	closeV2             int
	columnBlob          int
	columnBytes         int
	columnCount         int
	columnDouble        int
	columnInt64         int
	columnName          int
	columnText          int
	columnType          int
	errmsg              int
	errstr              int
	exec                int
	extendedResultCodes int
	finalize            int
	free                int
	lastInsertRowID     int
	maloc               int
	openV2              int
	prepareV2           int
	step                int
)

func init() {
	b0 := bytes.NewBufferString(bin.Data)
	decomp, err := gzip.NewReader(b0)
	if err != nil {
		panic(err)
	}

	var b1 bytes.Buffer
	chunk := make([]byte, 1<<15)
	for {
		n, err := decomp.Read(chunk)
		b1.Write(chunk[:n])
		if err != nil {
			if err != io.EOF {
				panic(err)
			}

			break
		}
	}
	dec := gob.NewDecoder(&b1)
	if err := dec.Decode(&binary); err != nil {
		panic(err)
	}

	for _, v := range []struct {
		*int
		string
	}{
		{&bindBlob, "sqlite3_bind_blob"},
		{&bindDouble, "sqlite3_bind_double"},
		{&bindInt, "sqlite3_bind_int"},
		{&bindInt64, "sqlite3_bind_int64"},
		{&bindParameterCount, "sqlite3_bind_parameter_count"},
		{&bindText, "sqlite3_bind_text"},
		{&changes, "sqlite3_changes"},
		{&closeV2, "sqlite3_close_v2"},
		{&columnBlob, "sqlite3_column_blob"},
		{&columnBytes, "sqlite3_column_bytes"},
		{&columnCount, "sqlite3_column_count"},
		{&columnDouble, "sqlite3_column_double"},
		{&columnInt64, "sqlite3_column_int64"},
		{&columnName, "sqlite3_column_name"},
		{&columnText, "sqlite3_column_text"},
		{&columnType, "sqlite3_column_type"},
		{&errmsg, "sqlite3_errmsg"},
		{&errstr, "sqlite3_errstr"},
		{&exec, "sqlite3_exec"},
		{&extendedResultCodes, "sqlite3_extended_result_codes"},
		{&finalize, "sqlite3_finalize"},
		{&free, "sqlite3_free"},
		{&lastInsertRowID, "sqlite3_last_insert_rowid"},
		{&maloc, "sqlite3_malloc"},
		{&openV2, "sqlite3_open_v2"},
		{&prepareV2, "sqlite3_prepare_v2"},
		{&step, "sqlite3_step"},
	} {
		var ok bool
		if *v.int, ok = binary.Sym[ir.NameID(dict.SID(v.string))]; !ok {
			panic(fmt.Errorf("missing symbol: %v", v.string))
		}
	}

	sql.Register(driverName, newDrv())
}

func tracer(rx interface{}, format string, args ...interface{}) {
	var b buffer.Bytes
	_, file, line, _ := runtime.Caller(1)
	fmt.Fprintf(&b, "%v:%v: (%[3]T)(%[3]p).", file, line, rx)
	fmt.Fprintf(&b, format, args...)
	fmt.Fprintf(os.Stderr, "%s\n", b.Bytes())
	b.Close()
}

func readI8(p uintptr) int8     { return *(*int8)(unsafe.Pointer(p)) }
func readPtr(p uintptr) uintptr { return *(*uintptr)(unsafe.Pointer(p)) }

type debugWriter struct{}

func (debugWriter) Write(b []byte) (int, error) { return os.Stderr.Write(b) }

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
	_, err := r.FFI1(
		lastInsertRowID,
		virtual.Int64Result{&v},
		virtual.Ptr(r.pdb()),
	)
	return v, err
}

// int sqlite3_changes(sqlite3*);
func (r *result) changes() (int, error) {
	var v int32
	_, err := r.FFI1(
		changes,
		virtual.Int32Result{&v},
		virtual.Ptr(r.pdb()),
	)
	return int(v), err
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
	pstmt   uintptr
	doStep  bool
}

func (r *rows) String() string {
	return fmt.Sprintf("&%T@%p{stmt: %p, columns: %v, rc0: %v, pstmt: %#x, doStep: %v}", *r, r, r.stmt, r.columns, r.rc0, r.pstmt, r.doStep)
}

func newRows(s *stmt, pstmt uintptr, rc0 int) (*rows, error) {
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
	var v int32
	if _, err = r.FFI1(
		columnBytes,
		virtual.Int32Result{&v},
		virtual.Ptr(r.pstmt), virtual.Int32(iCol),
	); err != nil {
		return 0, err
	}

	return int(v), err
}

// const void *sqlite3_column_blob(sqlite3_stmt*, int iCol);
func (r *rows) columnBlob(iCol int) (v []byte, err error) {
	var p uintptr
	if _, err = r.FFI1(
		columnBlob,
		virtual.PtrResult{&p},
		virtual.Ptr(r.pstmt), virtual.Int32(iCol),
	); err != nil {
		return nil, err
	}

	len, err := r.columnBytes(iCol)
	if err != nil {
		return nil, err
	}

	return virtual.GoBytesLen(p, len), nil
}

// const unsigned char *sqlite3_column_text(sqlite3_stmt*, int iCol);
func (r *rows) columnText(iCol int) (v string, err error) {
	var p uintptr
	if _, err = r.FFI1(
		columnText,
		virtual.PtrResult{&p},
		virtual.Ptr(r.pstmt), virtual.Int32(iCol),
	); err != nil {
		return "", err
	}

	len, err := r.columnBytes(iCol)
	if err != nil {
		return "", err
	}

	return virtual.GoStringLen(p, len), nil
}

// double sqlite3_column_double(sqlite3_stmt*, int iCol);
func (r *rows) columnDouble(iCol int) (v float64, err error) {
	_, err = r.FFI1(
		columnDouble,
		virtual.Float64Result{&v},
		virtual.Ptr(r.pstmt), virtual.Int32(iCol),
	)
	return v, err
}

// sqlite3_int64 sqlite3_column_int64(sqlite3_stmt*, int iCol);
func (r *rows) columnInt64(iCol int) (v int64, err error) {
	_, err = r.FFI1(
		columnInt64,
		virtual.Int64Result{&v},
		virtual.Ptr(r.pstmt), virtual.Int32(iCol),
	)
	return v, err
}

// int sqlite3_column_type(sqlite3_stmt*, int iCol);
func (r *rows) columnType(iCol int) (_ int, err error) {
	var v int32
	_, err = r.FFI1(
		columnType,
		virtual.Int32Result{&v},
		virtual.Ptr(r.pstmt), virtual.Int32(iCol),
	)
	return int(v), err
}

// int sqlite3_column_count(sqlite3_stmt *pStmt);
func (r *rows) columnCount() (_ int, err error) {
	var v int32
	_, err = r.FFI1(
		columnCount,
		virtual.Int32Result{&v},
		virtual.Ptr(r.pstmt),
	)
	return int(v), err
}

// const char *sqlite3_column_name(sqlite3_stmt*, int N);
func (r *rows) columnName(n int) (string, error) {
	var p uintptr
	if _, err := r.FFI1(
		columnName,
		virtual.PtrResult{&p},
		virtual.Ptr(r.pstmt), virtual.Int32(n),
	); err != nil {
		return "", err
	}

	return virtual.GoString(p), nil
}

type stmt struct {
	*conn
	allocs []uintptr
	psql   uintptr
	ppstmt uintptr
	pzTail uintptr
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
		s.free(psql)
		return nil, err
	}

	s.ppstmt = ppstmt
	pzTail, err := s.malloc(ptrSize)
	if err != nil {
		s.free(psql)
		s.free(ppstmt)
		return nil, err
	}

	s.pzTail = pzTail
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
	if s.psql != 0 {
		err = s.free(s.psql)
		s.psql = 0
	}
	if s.ppstmt != 0 {
		if err2 := s.free(s.ppstmt); err2 != nil && err == nil {
			err = err2
		}
		s.ppstmt = 0
	}
	if s.pzTail != 0 {
		if err2 := s.free(s.pzTail); err2 != nil && err == nil {
			err = err2
		}
		s.pzTail = 0
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
//
// Deprecated: Drivers should implement StmtExecContext instead (or
// additionally).
func (s *stmt) Exec(args []driver.Value) (r driver.Result, err error) {
	if trace {
		defer func(args []driver.Value) {
			tracer(s, "Exec(%v): (%v, %v)", args, r, err)
		}(args)
	}
	for psql := s.psql; readI8(psql) != 0; psql = readPtr(s.pzTail) {
		if err := s.prepareV2(psql); err != nil {
			return nil, err
		}

		pstmt := readPtr(s.ppstmt)
		if pstmt == 0 {
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

			args = args[n:]
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

// Query executes a query that may return rows, such as a SELECT.
//
// Deprecated: Drivers should implement StmtQueryContext instead (or
// additionally).
func (s *stmt) Query(args []driver.Value) (r driver.Rows, err error) {
	if trace {
		defer func(args []driver.Value) {
			tracer(s, "Query(%v): (%v, %v)", args, r, err)
		}(args)
	}
	var rowStmt uintptr
	var rc0 int
	for psql := s.psql; readI8(psql) != 0; psql = readPtr(s.pzTail) {
		if err := s.prepareV2(psql); err != nil {
			return nil, err
		}

		pstmt := readPtr(s.ppstmt)
		if pstmt == 0 {
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

			args = args[n:]
		}
		rc, err := s.step(pstmt)
		if err != nil {
			s.finalize(pstmt)
			return nil, err
		}

		switch rc {
		case bin.XSQLITE_ROW:
			if rowStmt != 0 {
				if err := s.finalize(pstmt); err != nil {
					return nil, err
				}

				return nil, fmt.Errorf("query contains multiple select statements")
			}

			rowStmt = pstmt
			rc0 = rc
		case bin.XSQLITE_DONE:
			if rowStmt == 0 {
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
func (s *stmt) bindDouble(pstmt uintptr, idx1 int, value float64) (err error) {
	var rc int32
	if _, err = s.FFI1(
		bindDouble,
		virtual.Int32Result{&rc},
		virtual.Ptr(pstmt), virtual.Int32(int32(idx1)), virtual.Float64(value),
	); err != nil {
		return err
	}

	if rc != bin.XSQLITE_OK {
		return s.errstr(rc)
	}

	return nil
}

// int sqlite3_bind_int(sqlite3_stmt*, int, int);
func (s *stmt) bindInt(pstmt uintptr, idx1, value int) (err error) {
	var rc int32
	if _, err = s.FFI1(
		bindInt,
		virtual.Int32Result{&rc},
		virtual.Ptr(pstmt), virtual.Int32(int32(idx1)), virtual.Int32(int32(value)),
	); err != nil {
		return err
	}

	if rc != bin.XSQLITE_OK {
		return s.errstr(rc)
	}

	return nil
}

// int sqlite3_bind_int64(sqlite3_stmt*, int, sqlite3_int64);
func (s *stmt) bindInt64(pstmt uintptr, idx1 int, value int64) (err error) {
	var rc int32
	if _, err = s.FFI1(
		bindInt64,
		virtual.Int32Result{&rc},
		virtual.Ptr(pstmt), virtual.Int32(int32(idx1)), virtual.Int64(value),
	); err != nil {
		return err
	}

	if rc != bin.XSQLITE_OK {
		return s.errstr(rc)
	}

	return nil
}

// int sqlite3_bind_blob(sqlite3_stmt*, int, const void*, int n, void(*)(void*));
func (s *stmt) bindBlob(pstmt uintptr, idx1 int, value []byte) (err error) {
	p, err := s.malloc(len(value))
	if err != nil {
		return err
	}

	s.allocs = append(s.allocs, p)
	virtual.CopyBytes(p, value, false)
	var rc int32
	if _, err = s.FFI1(
		bindBlob,
		virtual.Int32Result{&rc},
		virtual.Ptr(pstmt), virtual.Int32(int32(idx1)), virtual.Ptr(p), virtual.Int32(int32(len(value))), null,
	); err != nil {
		return err
	}

	if rc != bin.XSQLITE_OK {
		return s.errstr(rc)
	}

	return nil
}

// int sqlite3_bind_text(sqlite3_stmt*,int,const char*,int,void(*)(void*));
func (s *stmt) bindText(pstmt uintptr, idx1 int, value string) (err error) {
	p, err := s.cString(value)
	if err != nil {
		return err
	}

	s.allocs = append(s.allocs, p)
	var rc int32
	if _, err = s.FFI1(
		bindText,
		virtual.Int32Result{&rc},
		virtual.Ptr(pstmt), virtual.Int32(int32(idx1)), virtual.Ptr(p), virtual.Int32(int32(len(value))), null,
	); err != nil {
		return err
	}

	if rc != bin.XSQLITE_OK {
		return s.errstr(rc)
	}

	return nil
}

func (s *stmt) bind(pstmt uintptr, n int, args []driver.Value) error {
	if len(args) < n {
		return fmt.Errorf("missing arguments: got %v, expected %v", len(args), n)
	}

	for i, v := range args[:n] {
		switch x := v.(type) {
		case int64:
			if err := s.bindInt64(pstmt, i+1, x); err != nil {
				return err
			}
		case float64:
			if err := s.bindDouble(pstmt, i+1, x); err != nil {
				return err
			}
		case bool:
			v := 0
			if x {
				v = 1
			}
			if err := s.bindInt(pstmt, i+1, v); err != nil {
				return err
			}
		case []byte:
			if err := s.bindBlob(pstmt, i+1, x); err != nil {
				return err
			}
		case string:
			if err := s.bindText(pstmt, i+1, x); err != nil {
				return err
			}
		case time.Time:
			if err := s.bindText(pstmt, i+1, x.String()); err != nil {
				return err
			}
		default:
			return fmt.Errorf("invalid driver.Value type %T", x)
		}
	}
	return nil
}

// int sqlite3_bind_parameter_count(sqlite3_stmt*);
func (s *stmt) bindParameterCount(pstmt uintptr) (_ int, err error) {
	var r int32
	_, err = s.FFI1(
		bindParameterCount,
		virtual.Int32Result{&r},
		virtual.Ptr(pstmt),
	)
	return int(r), err
}

// int sqlite3_finalize(sqlite3_stmt *pStmt);
func (s *stmt) finalize(pstmt uintptr) error {
	var rc int32
	if _, err := s.FFI1(
		finalize,
		virtual.Int32Result{&rc},
		virtual.Ptr(pstmt),
	); err != nil {
		return err
	}

	if rc != bin.XSQLITE_OK {
		return s.errstr(rc)
	}

	return nil
}

// int sqlite3_step(sqlite3_stmt*);
func (s *stmt) step(pstmt uintptr) (int, error) {
	var rc int32
	_, err := s.FFI1(
		step,
		virtual.Int32Result{&rc},
		virtual.Ptr(pstmt),
	)
	return int(rc), err
}

// int sqlite3_prepare_v2(
//   sqlite3 *db,            /* Database handle */
//   const char *zSql,       /* SQL statement, UTF-8 encoded */
//   int nByte,              /* Maximum length of zSql in bytes. */
//   sqlite3_stmt **ppStmt,  /* OUT: Statement handle */
//   const char **pzTail     /* OUT: Pointer to unused portion of zSql */
// );
func (s *stmt) prepareV2(zSql uintptr) error {
	var rc int32
	if _, err := s.FFI1(
		prepareV2,
		virtual.Int32Result{&rc},
		virtual.Ptr(s.pdb()), virtual.Ptr(zSql), virtual.Int32(-1), virtual.Ptr(s.ppstmt), virtual.Ptr(s.pzTail),
	); err != nil {
		return err
	}

	if rc != bin.XSQLITE_OK {
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
	if err := t.exec("begin"); err != nil {
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
	return t.exec("commit")
}

// Rollback implements driver.Tx.
func (t *tx) Rollback() (err error) {
	if trace {
		defer func() {
			tracer(t, "Rollback(): %v", err)
		}()
	}
	return t.exec("rollback")
}

// int sqlite3_exec(
//   sqlite3*,                                  /* An open database */
//   const char *sql,                           /* SQL to be evaluated */
//   int (*callback)(void*,int,char**,char**),  /* Callback function */
//   void *,                                    /* 1st argument to callback */
//   char **errmsg                              /* Error msg written here */
// );
func (t *tx) exec(sql string) (err error) {
	psql, err := t.cString(sql)
	if err != nil {
		return err
	}

	defer t.free(psql)

	var rc int32
	if _, err = t.FFI1(
		exec,
		virtual.Int32Result{&rc},
		virtual.Ptr(t.pdb()), virtual.Ptr(psql), null, null, null,
	); err != nil {
		return err
	}

	if rc != bin.XSQLITE_OK {
		return t.errstr(rc)
	}

	return nil
}

type conn struct {
	*sqlite
	*virtual.Thread
	ppdb uintptr
}

func (c *conn) String() string {
	return fmt.Sprintf("&%T@%p{sqlite: %p, Thread: %p, ppdb: %#x}", *c, c, c.sqlite, c.Thread, c.ppdb)
}

func newConn(s *sqlite, name string) (_ *conn, err error) {
	c := &conn{sqlite: s}

	defer func() {
		if err != nil {
			c.close()
		}
	}()

	c.Lock()

	defer c.Unlock()

	c.conns++
	if c.conns == 1 {
		stderr := ioutil.Discard
		if trace {
			stderr = debugWriter{}
		}
		m, status, err2 := virtual.New(&binary, []string{"", fmt.Sprint(vmHeapSize - vmHeapReserve)}, nil, nil, stderr, vmHeapSize, vmMainStackSize, "")
		if status != 0 || err2 != nil {
			return nil, fmt.Errorf("virtual.New: %v, %v", status, err2)
		}

		vm = m
	}
	c.Thread, err = vm.NewThread(vmStackSize)
	if err != nil {
		return nil, err
	}

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

// Begin starts and returns a new transaction.
//
// Deprecated: Drivers should implement ConnBeginTx instead (or additionally).
func (c *conn) Begin() (t driver.Tx, err error) {
	if trace {
		defer func() {
			tracer(c, "Begin(): (%v, %v)", t, err)
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
//
// Deprecated: Drivers should implement ExecerContext instead (or
// additionally).
func (c *conn) Exec(query string, args []driver.Value) (r driver.Result, err error) {
	if trace {
		defer func() {
			tracer(c, "Exec(%s, %v): (%v, %v)", query, args, r, err)
		}()
	}
	s, err := c.Prepare(query)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err2 := s.Close(); err2 != nil && err == nil {
			err = err2
		}
	}()

	return s.Exec(args)
}

// Queryer is an optional interface that may be implemented by a Conn.
//
// If a Conn does not implement Queryer, the sql package's DB.Query will first
// prepare a query, execute the statement, and then close the statement.
//
// Query may return ErrSkip.
//
// Deprecated: Drivers should implement QueryerContext instead (or
// additionally).
func (c *conn) Query(query string, args []driver.Value) (r driver.Rows, err error) {
	if trace {
		defer func() {
			tracer(c, "Query(%s, %v): (%v, %v)", query, args, r, err)
		}()
	}
	s, err := c.Prepare(query)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err2 := s.Close(); err2 != nil && err == nil {
			err = err2
		}
	}()

	return s.Query(args)
}

func (c *conn) pdb() uintptr { return readPtr(c.ppdb) }

// int sqlite3_extended_result_codes(sqlite3*, int onoff);
func (c *conn) extendedResultCodes(on bool) (err error) {
	var v, rc int32
	if on {
		v = 1
	}
	if _, err = c.FFI1(
		extendedResultCodes,
		virtual.Int32Result{&rc},
		virtual.Ptr(c.pdb()), virtual.Int32(v),
	); err != nil {
		return err
	}

	if rc != bin.XSQLITE_OK {
		return c.errstr(rc)
	}

	return nil
}

// void *sqlite3_malloc(int);
func (c *conn) malloc(n int) (r uintptr, err error) {
	_, err = c.FFI1(
		maloc,
		virtual.PtrResult{&r},
		virtual.Int32(int32(n)),
	)
	return r, err
}

func (c *conn) cString(s string) (p uintptr, err error) {
	n := len(s)
	if p, err = c.malloc(n + 1); err != nil {
		return 0, err
	}

	virtual.CopyString(p, s, true)
	return p, nil
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

	defer c.free(filename)

	ppdb, err := c.malloc(ptrSize)
	if err != nil {
		return err
	}

	c.ppdb = ppdb
	var rc int32
	if _, err = c.FFI1(
		openV2,
		virtual.Int32Result{&rc},
		virtual.Ptr(filename), virtual.Ptr(ppdb), virtual.Int32(flags), null,
	); err != nil {
		return err
	}

	if rc != bin.XSQLITE_OK {
		return c.errstr(rc)
	}

	return nil
}

// const char *sqlite3_errstr(int);
func (c *conn) errstr(rc int32) (err error) {
	var p uintptr
	if _, err = c.FFI1(
		errstr,
		virtual.PtrResult{&p},
		virtual.Int32(rc),
	); err != nil {
		return err
	}

	str := virtual.GoString(p)
	if _, err = c.FFI1(
		errmsg,
		virtual.PtrResult{&p},
		virtual.Ptr(c.pdb()),
	); err != nil {
		return err
	}

	switch msg := virtual.GoString(p); {
	case msg == str:
		return fmt.Errorf("%s (%v)", str, rc)
	default:
		return fmt.Errorf("%s: %s (%v)", str, msg, rc)
	}
}

// int sqlite3_close_v2(sqlite3*);
func (c *conn) closeV2() (err error) {
	var rc int32
	if _, err = c.FFI1(
		closeV2,
		virtual.Int32Result{&rc},
		virtual.Ptr(c.pdb()),
	); err != nil {
		return err
	}

	if rc != bin.XSQLITE_OK {
		return c.errstr(rc)
	}

	err = c.free(c.ppdb)
	c.ppdb = 0
	return err
}

// void sqlite3_free(void*);
func (c *conn) free(p uintptr) (err error) {
	_, err = c.FFI0(
		free,
		virtual.Ptr(p),
	)
	return err
}

func (c *conn) close() (err error) {
	c.Lock()

	defer func() {
		c.conns--
		if c.conns == 0 {
			if err2 := vm.Close(); err2 != nil && err == nil {
				err = err2
			}
			vm = nil
		}
		c.Unlock()
	}()

	if c.ppdb != 0 {
		err = c.closeV2()
	}
	return err
}

type sqlite struct {
	conns int
	sync.Mutex
}

func newDrv() *sqlite { return &sqlite{} }

// Open returns a new connection to the database.  The name is a string in a
// driver-specific format.
//
// Open may return a cached connection (one previously closed), but doing so is
// unnecessary; the sql package maintains a pool of idle connections for
// efficient re-use.
//
// The returned connection is only used by one goroutine at a time.
func (s *sqlite) Open(name string) (c driver.Conn, err error) {
	if trace {
		defer func() {
			tracer(s, "Open(%s): (%v, %v)", name, c, err)
		}()
	}
	return newConn(s, name)
}
