package database

import (
	"fmt"

	"github.com/erinchen11/hr-system/internal/models"
	"gorm.io/gorm"
)

func RunMigrations(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&models.Account{},
		&models.Employment{},
		&models.JobGrade{},
		&models.LeaveRequest{},
	); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}
	return nil
}
