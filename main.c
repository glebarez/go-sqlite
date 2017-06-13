// Copyright 2017 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

#include <sqlite3.h>

int main(int argc, char **argv)
{
	// Prevent the linker from optimizing out everything.
	int (*f) (int, ...) = sqlite3_config;
}
