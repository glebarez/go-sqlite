// Copyright 2017 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package sqlite is an in-process implementation of a self-contained,
// serverless, zero-configuration, transactional SQL database engine. (Work In Progress)
//
// Changelog
//
// 2019-12-18 v1.1.0 First alpha release using the new cc/v3, gocc, qbe
// toolchain. Some primitive tests pass on linux_{amd64,386}. Not yet safe for
// concurrent access by multiple goroutines. Next alpha release is planed to
// arrive before the end of this year.
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
// Supported platforms and architectures
//
// 	linux	386
// 	linux	amd64
//
// Planned platforms and architectures
// 	linux	arm
// 	linux	arm64
// 	windows	386
// 	windows	amd64
//
//
// Sqlite documentation
//
// See https://sqlite.org/docs.html
package sqlite // import "modernc.org/sqlite"
