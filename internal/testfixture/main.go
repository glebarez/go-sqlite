// Copyright 2020 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package testfixture

import (
	"fmt"
	"os"
	"unsafe"

	"modernc.org/crt/v3"
)

func Main() {
	crt.Watch(fmt.Sprint(os.Args))
	tls := crt.NewTLS()
	argv := crt.Xcalloc(tls, crt.Size_t(len(os.Args)+1), crt.Size_t(unsafe.Sizeof(uintptr(0))))
	a := []uintptr{argv}
	p := argv
	for _, v := range os.Args {
		s := crt.Xcalloc(tls, crt.Size_t(len(v)+1), 1)
		a = append(a, s)
		copy((*(*[1 << 20]byte)(unsafe.Pointer(s)))[:], v)
		*(*uintptr)(unsafe.Pointer(p)) = s
		p += unsafe.Sizeof(uintptr(0))
	}
	crt.SetEnviron(os.Environ())
	rc := main(tls, int32(len(os.Args)), argv)
	for _, p := range a {
		crt.Xfree(tls, p)
	}
	crt.Xexit(tls, rc)
}
