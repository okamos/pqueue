package pqueue

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	os.Exit(runMain(m))
}

func runMain(m *testing.M) int {
	SetConfig("psql_dsn", "host=localhost user=postgres dbname=test sslmode=disable")
	err := NewDB()
	if err != nil {
		return 1
	}
	return m.Run()
}

func TruncateJob() {
	db.Exec("TRUNCATE job")
}
