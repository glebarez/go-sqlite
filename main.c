// Copyright 2017 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

#include <sqlite3.h>

static void use(int, ...)
{
}

int main(int argc, char **argv)
{
	use(0, sqlite3_exec, sqlite3_enable_load_extension);
}
