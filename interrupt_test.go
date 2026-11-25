// Copyright 2017 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlite // import "modernc.org/sqlite"

import (
	"context"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

// TestInterruptOnDoneSerializesWithClose makes the deferred function returned
// by interruptOnDone run while the cancellation watcher is still between
// claiming the cancellation and interrupting the connection. The pool closes a
// connection as soon as its statement returns, and Close writes c.db under the
// connection lock while the watcher reads it without the lock. Unless the
// deferred function waits for the watcher, the race detector reports the read
// against Close's write.
func TestInterruptOnDoneSerializesWithClose(t *testing.T) {
	c, err := newConn(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	orig := c.db
	defer c.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// The watcher reads c.db and then blocks in interrupt on this lock.
	c.Lock()
	var done int32
	stop := interruptOnDone(ctx, c, &done)
	deadline := time.Now().Add(5 * time.Second)
	for atomic.LoadInt32(&done) == 0 {
		if time.Now().After(deadline) {
			c.Unlock()
			t.Fatal("cancellation watcher did not claim the cancellation")
		}
		runtime.Gosched()
	}

	returned := make(chan struct{})
	go func() {
		stop()
		close(returned)
	}()

	select {
	case <-returned:
		// The statement returned while the watcher was still running: write
		// c.db as Close does once the statement returns.
		c.db = 0
		c.Unlock()
	case <-time.After(200 * time.Millisecond):
		// The deferred function is waiting for the watcher, which cannot finish
		// until this test releases the lock. Let it finish and require it back.
		c.Unlock()
		select {
		case <-returned:
		case <-time.After(5 * time.Second):
			t.Fatal("deferred function did not return after the watcher finished")
		}
		// The statement is done. Writing c.db as Close does cannot race with the
		// watcher's read if the deferred function waited for it.
		c.db = 0
		c.db = orig
	}
}
