// Copyright 2021 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// TPC BENCHMARK TM H
// (Decision Support)
// Standard Specification
// Revision 2.17.1
//
// Transaction Processing Performance Council (TPC)
// Presidio of San Francisco
// Building 572B Ruger St. (surface)
// P.O. Box 29920 (mail)
// San Francisco, CA 94129-0920
// Voice:415-561-6272
// Fax:415-561-6120
// Email: webmaster@tpc.org
// © 1993 - 2014 Transaction Processing Performance Council

package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"modernc.org/sqlite/tpch/driver"
	_ "modernc.org/sqlite/tpch/driver/drivers"
)

// 4.1.3.1 Scale factors used for the test database must be chosen from the set
// of fixed scale factors defined as follows:
//
// 	1, 10, 30, 100, 300, 1000, 3000, 10000, 30000, 100000
//
// The database size is defined with reference to scale factor 1 (i.e., SF = 1;
// approximately 1GB as per Clause 4.2.5), the minimum required size for a test
// database. Therefore, the following series of database sizes corresponds to
// the series of scale factors and must be used in the metric names QphH@Size
// and Price-per-QphH@Size (see Clause 5.4), as well as in the executive
// summary statement (see Appendix E):
//
//	1GB, 10GB, 30GB, 100GB, 300GB, 1000GB, 3000GB, 10000GB, 30000GB, 100000GB
//
//	Where GB stands for gigabyte, defined to be 2^30 bytes.
//
// Comment 1: Although the minimum size of the test database for a valid
// performance test is 1GB (i.e., SF = 1), a test database of 3GB (i.e., SF =
// 3) is not permitted. This requirement is intended to encourage comparability
// of results at the low end and to ensure a substantial actual difference in
// test database sizes.
//
// Comment 2: The maximum size of the test database for a valid performance
// test is currently set at 100000 (i.e., SF = 100,000). The TPC recognizes
// that additional benchmark development work is necessary to allow TPC-H to
// scale beyond that limit.

func main() {
	log.SetFlags(0)

	dbgen := flag.Bool("dbgen", false, "Generate test DB. (Several GB)")
	list := flag.Bool("list", false, "List registered drivers")
	maxrecs := flag.Int("recs", -1, "Limit table recs. Use specs if < 0.")
	mem := flag.Bool("mem", false, "Run test with DB in mem, if SUT supports that.")
	pseudotext := flag.Bool("pseudotext", false, "generate testdata/pseudotext (300MB).")
	q := flag.Int("q", 0, "Query to run, if > 0. Valid values in [1, 2].")
	sf := flag.Int("sf", 1, "Scale factor.")
	sutName := flag.String("sut", "", "System Under Test name.")
	verbose := flag.Bool("v", false, "Verbose.")

	flag.Parse()
	maxRecs = *maxrecs
	switch *sf {
	case 1, 10, 30, 100, 300, 1000, 3000, 10000, 30000, 100000:
		// nop
	default:
		log.Fatalf("Invalid -sf value: %v", *sf)
	}

	var sut driver.SUT
	nm := strings.TrimSpace(*sutName)
	if nm == "" && !*pseudotext && !*list {
		log.Fatal("Missing SUT name")
	}

	if nm != "" {
		if sut = driver.Open(nm); sut == nil {
			log.Fatalf("SUT not registered: %s", nm)
		}
	}

	var err error
	switch {
	case *list:
		fmt.Println(driver.List())
	case *pseudotext:
		err = genPseudotext()
	case *dbgen:
		err = dbGen(sut, *sf)
	case *q > 0:
		err = run(sut, *mem, *q, *sf, *verbose)
	}

	if err != nil {
		log.Fatal(err)
	}
}
