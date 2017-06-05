// Copyright 2017 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// SQLite Is Public Domain

// +build ignore

#define minAlloc (2<<5)

#include <sqlite3.h>
#include <stdlib.h>

int main(int argc, char **argv)
{
	init(-1);
}

int init(int heapSize)
{
	void *heap = malloc(heapSize);
	if (heap == 0) {
		return 1;
	}

	int rc = sqlite3_config(SQLITE_CONFIG_HEAP, heap, heapSize, minAlloc);
	if (rc) {
		return 2;
	}

	rc = sqlite3_threadsafe();
	if (!rc) {
		return 3;
	}

	return 0;
}
