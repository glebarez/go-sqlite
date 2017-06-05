// Copyright 2017 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package sqlite is an in-process implementation of a self-contained,
// serverless, zero-configuration, transactional SQL database engine. (Work In Progress)
//
// Changelog
//
// 2017-06-05 Linux/Intel no more uses the VM (cznic/virtual).
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
// Even though the translation to Go is now done for Linux/Intel, the package
// status is still as described above, it's just closer to the alpha release in
// this respect. The alpha release is due when the C runtime support of SQLite
// in cznic/crt will be complete.
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
