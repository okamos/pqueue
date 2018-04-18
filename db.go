package pqueue

import (
	"database/sql"

	// pq package called only init
	_ "github.com/lib/pq"
)

func dsn() string {
	return cached.m["psql_dsn"]
}

var db *sql.DB

// NewDB creates an instance of DB handler
func NewDB() error {
	var err error
	db, err = sql.Open("postgres", dsn())
	return err
}
