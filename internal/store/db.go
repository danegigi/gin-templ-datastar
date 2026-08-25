package store

import (
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

// Connect opens and returns a *sqlx.DB using the READ_DATABASE_URL env var.
// Falls back to DATABASE_URL if not set.
func Connect() (*sqlx.DB, error) {
	// dsn := os.Getenv("READ_DATABASE_URL")
	dsn := os.Getenv("DATABASE_URL")
	log.Printf("DNS: %s", dsn)
	if dsn == "" {
		return nil, fmt.Errorf("No database URL configured (DATABASE_URL)")
	}

	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("Unable to reach database: %w", err)
	}
	return db, nil
}

// WritableConnect opens a second pool using WRITE_DATABASE_URL (or falls back
// to READ_DATABASE_URL / DATABASE_URL).  Returns the read pool when only one
// DSN is configured – suitable for environments without a read replica.
func WritableConnect() (*sqlx.DB, error) {
	dsn := os.Getenv("WRITE_DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("READ_DATABASE_URL")
	}
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		return nil, fmt.Errorf("no writable database URL configured")
	}

	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("unable to reach writable database: %w", err)
	}
	return db, nil
}
