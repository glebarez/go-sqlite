## Benchmarks
The benchmarking is conducted against CGo implementation of SQLite driver (https://github.com/mattn/go-sqlite3).

Benchmark tests are inspired by and closely repeat those described in https://www.sqlite.org/speed.html.

## Doing benchmarks
Benchmarks are run by custom runner and invoked with
```console
go test -v .
```
Additional command line arguments:

| flag | type | default | description                                                                                     |
| ---- | ---- | ------- | ----------------------------------------------------------------------------------------------- |
| -mem | bool | false   | if set, use in-memory SQLite                                                                    |
| -rep | uint | 1       | run each benchmark multiple times and average the results. this may provide more stable results |


## Current results
```console
root@vmi369792:~/sqldev/go-sqlite/benchmark# go test -v . -count=1
=== RUN   Test_BenchmarkSQLite

goos:   linux
goarch: amd64
cpu:    Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz
repeat: 1 time(s)
in-memory SQLite: true

bench_create_index                  |  3.23x | CGo: 159.612 ms/op | Pure-Go: 515.857 ms/op
bench_select_on_string_comparison   |  3.66x | CGo:  30.391 ms/op | Pure-Go: 111.339 ms/op
bench_select_with_index             |  3.85x | CGo:   0.009 ms/op | Pure-Go:   0.035 ms/op
bench_select_without_index          |  2.37x | CGo:  13.812 ms/op | Pure-Go:  32.796 ms/op
bench_insert                        |  1.82x | CGo:   0.014 ms/op | Pure-Go:   0.026 ms/op
bench_insert_in_transaction         |  1.32x | CGo:   0.013 ms/op | Pure-Go:   0.017 ms/op
bench_insert_into_indexed           |  1.68x | CGo:   0.015 ms/op | Pure-Go:   0.024 ms/op
bench_insert_from_select            |  2.24x | CGo:  46.112 ms/op | Pure-Go: 103.090 ms/op
bench_update_text_with_index        |  2.75x | CGo:   0.010 ms/op | Pure-Go:   0.027 ms/op
bench_update_with_index             |  4.12x | CGo:   0.006 ms/op | Pure-Go:   0.024 ms/op
bench_update_without_index          |  1.79x | CGo:  12.881 ms/op | Pure-Go:  23.078 ms/op
bench_delete_without_index          |  1.02x | CGo: 414.220 ms/op | Pure-Go: 420.956 ms/op
bench_delete_with_index             |  2.61x | CGo:  52.482 ms/op | Pure-Go: 137.178 ms/op
--- PASS: Test_BenchmarkSQLite (242.13s)
```