// Copyright 2017 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package sqlite is an in-process implementation of a self-contained,
// serverless, zero-configuration, transactional SQL database engine. (Work In Progress)
//
// Changelog
//
// 2020-07-26 v1.4.0-beta1:
//
// The project has reached beta status while supporting linux/amd64 only at the
// moment. The 'extraquick' Tcl testsuite reports
//
//	630 errors out of 200177 tests on  Linux 64-bit little-endian
//
// and some memory leaks
//
//	Unfreed memory: 698816 bytes in 322 allocations
//
// 2019-12-28 v1.2.0-alpha.3: Third alpha fixes issue #19.
//
// It also bumps the minor version as the repository was wrongly already tagged
// with v1.1.0 before.  Even though the tag was deleted there are proxies that
// cached that tag. Thanks /u/garaktailor for detecting the problem and
// suggesting this solution.
//
// 2019-12-26 v1.1.0-alpha.2: Second alpha release adds support for accessing a
// database concurrently by multiple goroutines and/or processes. v1.1.0 is now
// considered feature-complete. Next planed release should be a beta with a
// proper test suite.
//
// 2019-12-18 v1.1.0-alpha.1: First alpha release using the new cc/v3, gocc,
// qbe toolchain. Some primitive tests pass on linux_{amd64,386}. Not yet safe
// for concurrent access by multiple goroutines. Next alpha release is planed
// to arrive before the end of this year.
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
// Supported platforms and architectures
//
// 	linux	amd64
//
// Planned platforms and architectures
//
// 	linux	386
// 	linux	arm
// 	linux	arm64
// 	windows	386
// 	windows	amd64
//
// Sqlite documentation
//
// See https://sqlite.org/docs.html
package sqlite // import "modernc.org/sqlite"
