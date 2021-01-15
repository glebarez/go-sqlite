// Copyright 2021 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package driver

import (
	"database/sql"
	"fmt"
)

// System Under Test.
type SUT interface {
	CreateTables() error
	InsertCustomer() string
	InsertLineItem() string
	InsertNation() string
	InsertOrders() string
	InsertPart() string
	InsertPartSupp() string
	InsertRegion() string
	InsertSupplier() string
	Name() string
	OpenDB() (*sql.DB, error)
	OpenMem() (SUT, *sql.DB, error)
	Q1() string
	Q2() string
	QProperty() string
	SetWD(path string) error
}

var registered = map[string]SUT{}

func Open(name string) SUT {
	return registered[name]
}

func Register(sut SUT) {
	nm := sut.Name()
	if _, ok := registered[nm]; ok {
		panic(fmt.Errorf("already registered: %s", nm))
	}

	registered[nm] = sut
}

func List() []string {
	r := []string{}
	for k := range registered {
		r = append(r, k)
	}
	return r
}
