package db

import (
	"context"
	"database/sql"
	"fmt"

	sqlitedialect "gorm.io/driver/sqlite"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	_ "modernc.org/sqlite" // Pure Go SQLite driver (no CGO required)
)

type SQLiteDB struct {
	db *gorm.DB
}

// NewSQLiteDB creates a new SQLite database connection
func NewSQLiteDB(dbPath string) (*SQLiteDB, error) {
	// By default, GORM's SQLite dialect uses github.com/mattn/go-sqlite3 which requires CGO.
	// Here, we're creating a custom setup that uses modernc.org/sqlite (pure Go SQLite implementation) instead.
	// This allows us to build with CGO_ENABLED=0 while still using GORM's SQLite dialect.
	// The approach works because:
	// 1. modernc.org/sqlite registers itself with the same driver name ("sqlite")
	// 2. We can pass our own sql.DB connection to GORM's dialector
	// 3. GORM will use our connection with the modernc driver instead of creating its own with mattn/go-sqlite3
	sqlDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Create a custom dialector using the opened connection
	dialector := sqlitedialect.Dialector{
		DriverName: "sqlite",
		DSN:        dbPath,
		Conn:       sqlDB,
	}

	// Open a GORM connection using the custom dialector
	gormDB, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error), // supress any logs from GORM except errors
	})
	if err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to open gorm database: %w", err)
	}

	return &SQLiteDB{
		db: gormDB,
	}, nil
}

func (s *SQLiteDB) GetDB() *gorm.DB {
	return s.db
}

// Execute a SQL command
func (s *SQLiteDB) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	sqlDB, err := s.db.DB()
	if err != nil {
		return nil, err
	}
	return sqlDB.ExecContext(ctx, query, args...)
}

// Execute a SQL query
func (s *SQLiteDB) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	sqlDB, err := s.db.DB()
	if err != nil {
		return nil, err
	}
	return sqlDB.QueryContext(ctx, query, args...)
}

// Execute a single-row SQL query
func (s *SQLiteDB) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	sqlDB, err := s.db.DB()
	if err != nil {
		return nil
	}
	return sqlDB.QueryRowContext(ctx, query, args...)
}

// Close the database connection
func (s *SQLiteDB) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
