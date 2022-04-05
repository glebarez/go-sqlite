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
| -mem | bool | false   | if true: benchmarks will use in-memory SQLite instance,  otherwise: on-disk instance            |
| -rep | uint | 1       | run each benchmark multiple times and average the results. this may provide more stable results |


## Current results
```text
=== RUN   Test_BenchmarkSQLite

goos:   darwin
goarch: amd64
cpu:    Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
repeat: 1 time(s)
in-memory SQLite: false

bench_create_index                  |  1.80x | CGo: 120.880 ms/op | Pure-Go: 217.574 ms/op
bench_select_on_string_comparison   |  2.25x | CGo:  19.326 ms/op | Pure-Go:  43.498 ms/op
bench_select_with_index             |  5.84x | CGo:   0.002 ms/op | Pure-Go:   0.014 ms/op
bench_select_without_index          |  1.50x | CGo:   6.071 ms/op | Pure-Go:   9.111 ms/op
bench_insert                        |  1.17x | CGo:   0.481 ms/op | Pure-Go:   0.565 ms/op
bench_insert_in_transaction         |  1.78x | CGo:   0.004 ms/op | Pure-Go:   0.006 ms/op
bench_insert_into_indexed           |  1.62x | CGo:   0.008 ms/op | Pure-Go:   0.013 ms/op
bench_insert_from_select            |  1.80x | CGo:  30.409 ms/op | Pure-Go:  54.703 ms/op
bench_update_text_with_index        |  3.26x | CGo:   0.004 ms/op | Pure-Go:   0.013 ms/op
bench_update_with_index             |  4.20x | CGo:   0.003 ms/op | Pure-Go:   0.011 ms/op
bench_update_without_index          |  1.40x | CGo:   6.421 ms/op | Pure-Go:   9.010 ms/op
bench_delete_without_index          |  1.28x | CGo: 180.734 ms/op | Pure-Go: 231.105 ms/op
bench_delete_with_index             |  1.85x | CGo:  34.284 ms/op | Pure-Go:  63.569 ms/op
--- PASS: Test_BenchmarkSQLite (171.62s)
```