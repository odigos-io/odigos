package db

import (
	"context"
	"database/sql"

	"gorm.io/gorm"
)

// Database defines common DB operations
type Database interface {
	GetDB() *gorm.DB
	Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row
	Close() error
}
