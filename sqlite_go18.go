// Copyright 2017 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build go1.8

package sqlite // import "modernc.org/sqlite"

import (
	"context"
	"database/sql/driver"
	"errors"
)

// Ping implements driver.Pinger
func (c *conn) Ping(ctx context.Context) error {
	c.Lock()
	defer c.Unlock()

	if c.ppdb == 0 {
		return errors.New("db is closed")
	}

	_, err := c.ExecContext(ctx, "select 1", nil)
	return err
}

// BeginTx implements driver.ConnBeginTx
func (c *conn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	return c.begin(ctx, txOptions{
		Isolation: int(opts.Isolation),
		ReadOnly:  opts.ReadOnly,
	})
}

// PrepareContext implements driver.ConnPrepareContext
func (c *conn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	return c.prepare(ctx, query)
}

// ExecContext implements driver.ExecerContext
func (c *conn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	return c.exec(ctx, query, toNamedValues2(args))
}

// QueryContext implements driver.QueryerContext
func (c *conn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	return c.query(ctx, query, toNamedValues2(args))
}

// ExecContext implements driver.StmtExecContext
func (s *stmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	return s.exec(ctx, toNamedValues2(args))
}

// QueryContext implements driver.StmtQueryContext
func (s *stmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	return s.query(ctx, toNamedValues2(args))
}

// converts []driver.NamedValue to []namedValue
func toNamedValues2(vals []driver.NamedValue) []namedValue {
	args := make([]namedValue, 0, len(vals))
	for _, val := range vals {
		args = append(args, namedValue(val))
	}
	return args
}
