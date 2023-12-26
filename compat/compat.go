package compat

import "github.com/glebarez/go-sqlite"

func init() {
	sqlite.RegisterAsSQLITE3()
}
