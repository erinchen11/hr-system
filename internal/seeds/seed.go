package seeds

import (
	"log"

	"gorm.io/gorm"
)

// Run 對外呼叫，用來一鍵 Seed
func Run(db *gorm.DB) error {
	log.Println("Starting database seeding...")

	if err := SeedJobGrades(db); err != nil {
		log.Printf("Failed to seed job grades: %v", err)
	}

	if err := SeedAccountsAndEmployments(db); err != nil {
		log.Printf("Failed to seed employees: %v", err)
	}

	if err := SeedLeaveRequests(db); err != nil {
		log.Printf("Failed to seed leave requests: %v", err)
	}

	log.Println("Database seeding finished.")
	return nil
}
