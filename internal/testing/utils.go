package testing

import (
	"database/sql"

	_ "github.com/lib/pq"
)

func connectToPostgreSqlDb() *sql.DB {
	db, _ := sql.Open("postgres", "postgresql:///?sslmode=disable&user=postgres&password=postgres&host=localhost&port=5432")
	return db
}
