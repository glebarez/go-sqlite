// Copyright 2021 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package libc2 // import "modernc.org/sqlite/internal/libc2"

var CAPI = map[string]struct{}{
	"__libc_cond_broadcast":  {},
	"__libc_cond_destroy":    {},
	"__libc_cond_init":       {},
	"__libc_cond_signal":     {},
	"__libc_cond_wait":       {},
	"__libc_create":          {},
	"__libc_detach":          {},
	"__libc_mutex_destroy":   {},
	"__libc_mutex_init":      {},
	"__libc_mutex_lock":      {},
	"__libc_mutex_trylock":   {},
	"__libc_mutex_unlock":    {},
	"__libc_thr_yield":       {},
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
