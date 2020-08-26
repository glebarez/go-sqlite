# Copyright 2019 The Sqlite Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

.PHONY:	all clean cover cpu editor internalError later mem nuke todo edit

grep=--include=*.go --include=*.l --include=*.y --include=*.yy
ngrep='TODOOK\|internal\/bin'

all: editor
	date
	go version 2>&1 | tee log
	gofmt -l -s -w *.go
	go test -i
	go test 2>&1 -timeout 1h | tee -a log
	#TODO GOOS=linux GOARCH=arm go build
	#TODO GOOS=linux GOARCH=arm64 go build
	#TODO GOOS=linux GOARCH=386 go build
	GOOS=linux GOARCH=amd64 go build
	#TODO GOOS=windows GOARCH=386 go build
	#TODO GOOS=windows GOARCH=amd64 go build
	go vet 2>&1 | grep -v $(ngrep) || true
	golint 2>&1 | grep -v $(ngrep) || true
	make todo
	misspell *.go
	staticcheck | grep -v 'lexer\.go\|parser\.go' || true
	maligned || true
	grep -n 'FAIL\|PASS' log 
	go version
	date 2>&1 | tee -a log

clean:
	go clean
	rm -f *~ *.test *.out test.db* tt4-test*.db* test_sv.* testdb-*

cover:
	t=$(shell tempfile) ; go test -coverprofile $$t && go tool cover -html $$t && unlink $$t

cpu: clean
	go test -run @ -bench . -cpuprofile cpu.out
	go tool pprof -lines *.test cpu.out

edit:
	gvim -p Makefile *.go &

editor:
	gofmt -l -s -w *.go
	go test 2>&1 | tee log

internalError:
	egrep -ho '"internal error.*"' *.go | sort | cat -n

later:
	@grep -n $(grep) LATER * || true
	@grep -n $(grep) MAYBE * || true

mem: clean
	go test -run @ -bench . -memprofile mem.out
	go tool pprof -lines -web -alloc_space *.test mem.out

nuke: clean
	go clean -i

todo:
	@grep -nr $(grep) ^[[:space:]]*_[[:space:]]*=[[:space:]][[:alpha:]][[:alnum:]]* * | grep -v $(ngrep) || true
	@grep -nr $(grep) TODO * | grep -v $(ngrep) || true
	@grep -nr $(grep) BUG * | grep -v $(ngrep) || true
	@grep -nr $(grep) [^[:alpha:]]println * | grep -v $(ngrep) || true
