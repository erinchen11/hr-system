package seeds

import (
	"errors"
	"fmt"
	"log"

	"github.com/erinchen11/hr-system/internal/models"
	"github.com/google/uuid"        // 雖然 ID 自動生成，但保留以備不時之需
	"github.com/shopspring/decimal" // 用於處理薪資
	"gorm.io/gorm"
)

// SeedJobGrades 負責向 job_grades 表植入初始數據
func SeedJobGrades(db *gorm.DB) (err error) { // Named return err for easier handling in defer

	// --- 定義職等種子資料 ---
	jobGrades := []models.JobGrade{
		{
			Code:        "P1",
			Name:        "Associate Engineer",
			Description: "Entry-level professional contributor.",
			MinSalary:   decimal.NewFromInt(50000),
			MaxSalary:   decimal.NewFromInt(75000),
		},
		{
			Code:        "P2",
			Name:        "Engineer",
			Description: "Intermediate professional contributor.",
			MinSalary:   decimal.NewFromInt(65000),
			MaxSalary:   decimal.NewFromInt(90000),
		},
		{
			Code:        "P3",
			Name:        "Senior Engineer",
			Description: "Experienced professional contributor.",
			MinSalary:   decimal.NewFromInt(80000),
			MaxSalary:   decimal.NewFromInt(120000),
		},
		{
			Code:        "M1",
			Name:        "Manager",
			Description: "First-level manager.",
			MinSalary:   decimal.NewFromInt(90000),
			MaxSalary:   decimal.NewFromInt(140000),
		},
		{
			Code:        "M2",
			Name:        "Senior Manager",
			Description: ("Experienced manager."),
			MinSalary:   decimal.NewFromInt(110000),
			MaxSalary:   decimal.NewFromInt(170000),
		},
		{
			Code:        "D1",
			Name:        "Director",
			Description: "Senior leader.",
			MinSalary:   decimal.NewFromInt(150000),
			MaxSalary:   decimal.NewFromInt(250000),
		},
		{
			Code:        "IC0", // Intern / Contractor Example
			Name:        "Intern/Contractor",
			Description: ("Non-permanent or entry-level role."),
			// 薪資範圍可以設為 0 或特定值
			MinSalary: decimal.NewFromInt(0),
			MaxSalary: decimal.NewFromInt(0),
		},
	}

	// --- 使用事務 ---
	tx := db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin seed transaction for job grades: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		} else if err != nil {
			log.Printf("Rolling back job grade seed transaction due to error: %v", err)
			tx.Rollback()
		}
	}()

	log.Println("🌱 Starting job grade seeding...")
	createdCount := 0
	skippedCount := 0

	for _, grade := range jobGrades {
		// 檢查 Code 是否已存在
		var existing models.JobGrade
		findErr := tx.Where("code = ?", grade.Code).First(&existing).Error

		if findErr == nil {
			// Code 已存在，跳過
			skippedCount++
			continue
		}

		if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			// 查詢時發生其他資料庫錯誤
			err = fmt.Errorf("database error checking existence for job grade code %s: %w", grade.Code, findErr)
			log.Println(err)
			return err // 觸發 Rollback
		}

		// --- 職等不存在，創建 ---
		grade.ID = uuid.Nil // 確保觸發 BeforeCreate hook
		if createErr := tx.Create(&grade).Error; createErr != nil {
			err = fmt.Errorf("failed to create job grade with code %s: %w", grade.Code, createErr)
			log.Println(err)
			return err // 觸發 Rollback
		}
		createdCount++
	}

	log.Printf("Job grade seeding finished. Created: %d, Skipped: %d.", createdCount, skippedCount)

	// --- 提交事務 ---
	err = tx.Commit().Error
	if err != nil {
		return fmt.Errorf("failed to commit job grade seed transaction: %w", err)
	}

	log.Println("✅ Job grade seeding completed successfully.")
	return nil // 成功返回 nil
}
