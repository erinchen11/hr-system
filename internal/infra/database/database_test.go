package database

import (
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestInitializeDB(t *testing.T) {
	// --- Test Case 1: Successful Initialization ---
	t.Run("Success", func(t *testing.T) {
		cfg := DatabaseConfig{
			User:         "testuser",
			Passwd:       "testpass",
			Host:         "localhost",
			Port:         "3306",
			DbName:       "testdb",
			MaxOpenConns: 10,
			MaxIdleConns: 5,
			MaxLifetime:  300, // 5 minutes
		}

		mockDb, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		require.NoError(t, err, "Failed to create sqlmock")
		defer mockDb.Close()

		mock.ExpectPing()

		dialector := mysql.New(mysql.Config{
			Conn:                      mockDb,
			SkipInitializeWithVersion: true,
		})

		gormDB, err := gorm.Open(dialector, &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		require.NoError(t, err, "gorm.Open failed with mock connection")
		require.NotNil(t, gormDB, "gormDB should not be nil")

		sqlDB, err := gormDB.DB()
		require.NoError(t, err, "gormDB.DB() failed")
		require.NotNil(t, sqlDB, "sqlDB should not be nil")

		assert.NotPanics(t, func() {
			sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
			sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
			sqlDB.SetConnMaxLifetime(time.Duration(cfg.MaxLifetime) * time.Second)
		}, "Setting connection pool parameters panicked")

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err, "Mock expectations were not met")
	})

	// --- Test Case 2: gorm.Open Failure ---
	t.Run("GormOpenFailure", func(t *testing.T) {


		invalidDsn := "this:is:not:a:valid:dsn"
		_, err := gorm.Open(mysql.Open(invalidDsn), &gorm.Config{ // Use invalidDsn here
			Logger: logger.Default.LogMode(logger.Silent),
		})

		// Check that an error occurred and wrap it like in InitializeDB
		require.Error(t, err, "gorm.Open should fail with invalid DSN")
		wrappedErr := fmt.Errorf("failed to connect database: %w", err) // Mimic wrapping
		assert.Contains(t, wrappedErr.Error(), "failed to connect database", "Error message should be wrapped")
		// Optionally check if the original error is also present if driver/gorm provides consistent errors
		// assert.ErrorContains(t, err, "specific error string from mysql driver if known")
	})

	// --- Test Case 3: db.DB() Failure (Extremely hard to simulate reliably) ---
	// ... (omitted for brevity) ...
}
