// Copyright 2021 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package benchmark

import (
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"path"
	"testing"
)

var (
	// in dryRun we just generate random values to quickly see how information is plotted
	dryRun bool

	// whethe to use dark palette when plotting results
	darkPalette bool
)

func TestMain(m *testing.M) {
	flag.BoolVar(&dryRun, "dry", false, "just generate random values to quickly see how information is plotted")
	flag.BoolVar(&darkPalette, "dark", false, "use dark palette when plotting")
	flag.Parse()
	os.Exit(m.Run())
}

func TestBenchmarkAndPlot(t *testing.T) {
	// choose palette for plottin
	var palette = LightPalette
	if darkPalette {
		palette = DarkPalette
	}

	for _, benchFunc := range allBenchmarksOfNRows {
		for _, isMemoryDB := range inMemory {

			// create graph
			graph := &GraphCompareOfNRows{
				title:      fmt.Sprintf("%s | In-Memory: %v", getFuncName(benchFunc), isMemoryDB),
				rowCountsE: rowCountsE,
				palette:    palette,
			}

			// drivers
			for _, driver := range drivers {
				// this slice accumulates values as float64, for later plotting
				var (
					seriesValues []float64
					rowsPerSec   float64
				)

				// number of rows in table
				for _, e := range rowCountsE {
					if dryRun {
						// in dryRun mode we just generate random value to quickly see how information is plotted
						rowsPerSec = rand.Float64() * 200000
					} else {
						// create DB
						db := createDB(t, isMemoryDB, driver)

						// run benchmark
						result := testing.Benchmark(func(b *testing.B) {
							benchFunc(b, db, int(math.Pow10(e)))
						})

						// close DB
						db.Close()

						// calculate rows/sec
						rowsPerSec = math.Pow10(e) * float64(result.N) / result.T.Seconds()
					}

					// print result to console (FYI)
					benchName := fmt.Sprintf("%s_%s", getFuncName(benchFunc), makeName(isMemoryDB, driver, e))
					fmt.Println(benchName, "\t", fmt.Sprintf("%10.0f", rowsPerSec), "rows/sec")

					// add corresponding value to series
					seriesValues = append(seriesValues, rowsPerSec)
				}

				// add series to graph
				var seriesName string
				if driver == "sqlite3" {
					seriesName = "CGo"
				} else {
					seriesName = "Go"
				}
				graph.AddSeries(seriesName, seriesValues)
			}

			// render graph into file
			outputFilename := path.Join("out", fmt.Sprintf("%s_memory_%v.png", getFuncName(benchFunc), isMemoryDB))
			if err := graph.Render(outputFilename); err != nil {
				log.Fatal(err)
			}
			log.Printf("plot written into %s\n", outputFilename)
		}
	}
}
