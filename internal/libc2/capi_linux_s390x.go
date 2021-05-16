// Copyright 2020 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package libc2 // import "modernc.org/sqlite/internal/libc2"

var CAPI = map[string]struct{}{
	"pthread_cond_broadcast": {},
	"pthread_cond_destroy":   {},
	"pthread_cond_init":      {},
	"pthread_cond_signal":    {},
	"pthread_cond_wait":      {},
	"pthread_create":         {},
	"pthread_detach":         {},
	"pthread_mutex_destroy":  {},
	"pthread_mutex_init":     {},
	"pthread_mutex_lock":     {},
	"pthread_mutex_trylock":  {},
	"pthread_mutex_unlock":   {},
	"sched_yield":            {},
}
