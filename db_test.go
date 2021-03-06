package pqueue

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	os.Exit(runMain(m))
}

func runMain(m *testing.M) int {
	err := NewDB()
	if err != nil {
		return 1
	}
	return m.Run()
}

func TruncateJob() {
	db.Exec("TRUNCATE job")
}
