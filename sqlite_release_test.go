package sqlite

import (
	"database/sql"
	"log"
	"testing"
	"time"
)

func TestSQLiteVersion(t *testing.T) {

	db, err := sql.Open(driverName, ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	var (
		version  string
		sourceID string
	)

	row := db.QueryRow("select sqlite_version(), sqlite_source_id()")
	if row.Scan(&version, &sourceID) != nil {
		log.Fatal(err)
	}

	releaseDate, err := time.Parse(`2006-01-02`, sourceID[:10])
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%s (%s)\n", version, releaseDate.Format(`02/Jan/2006`))
}
