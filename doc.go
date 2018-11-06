// Copyright 2017 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package sqlite is an in-process implementation of a self-contained,
// serverless, zero-configuration, transactional SQL database engine. (Work In Progress)
//
// Changelog
//
// 2017-06-10 Windows/Intel no more uses the VM (thanks Steffen Butzer).
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
//		_ "modernc.org/sqlite"
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
// This is an experimental, pre-alpha, technology preview package.
//
// The alpha release is due when the C runtime support of SQLite in cznic/crt
// will be complete.
//
// Supported platforms and architectures
//
// See http://modernc.org/ccir. To add a newly supported os/arch
// combination to this package try running 'go generate'.
//
// Sqlite documentation
//
// See https://sqlite.org/docs.html
package sqlite // import "modernc.org/sqlite"
