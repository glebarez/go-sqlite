module github.com/glebarez/go-sqlite

go 1.16

require (
	golang.org/x/sys v0.0.0-20211007075335-d3039528d8ac
	modernc.org/ccgo/v3 v3.12.88
	modernc.org/libc v1.11.90
	modernc.org/mathutil v1.4.1
	modernc.org/sqlite v1.14.2
	modernc.org/tcl v1.9.1 // indirect
)

retract (
	v1.14.2 // non-working
	v1.9.0 // non-working
	v0.0.1 // early bird
)
