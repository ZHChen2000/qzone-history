package sqlite

import (
	"fmt"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"qzone-history/pkg/database"
)

type SQLiteDB struct {
	db *gorm.DB
}

// NewSQLiteDB creates a new SQLite database instance.
func NewSQLiteDB() database.Database {
	return &SQLiteDB{}
}

// Connect connects to the SQLite database.
// If `DBName` is empty, it will use an in-memory database.
func (s *SQLiteDB) Connect(config *database.Config) error {
	var dsn string
	if config.DBName != "" {
		dsn = config.DBName
	} else {
		// Use in-memory database
		dsn = ":memory:"
	}

	var err error
	s.db, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to SQLite database: %w", err)
	}

	return nil
}

// Close closes the SQLite database connection.
func (s *SQLiteDB) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}
	return sqlDB.Close()
}

// DB returns the GORM DB instance.
func (s *SQLiteDB) DB() *gorm.DB {
	return s.db
}
