// 檔案路徑: internal/seeds/seed_employee.go
package seeds

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/erinchen11/hr-system/environment"
	"github.com/erinchen11/hr-system/internal/models" // 使用 models 包
	"github.com/erinchen11/hr-system/internal/utils"  // 導入 utils 以使用 Ptr (根據您之前的程式碼)
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SeedAccountsAndEmployments 函數現在負責同時植入 Account 和 Employment 記錄，並關聯 JobGrade
func SeedAccountsAndEmployments(db *gorm.DB) (err error) { // Named return err for easier handling in defer
	now := time.Now()
	hireDate := now.Truncate(24 * time.Hour) // Truncate time for DATE type

	// --- 1. 獲取必要的職等資訊 ---
	log.Println("Fetching required job grades for seeding...")
	jobGradesMap := make(map[string]models.JobGrade)
	requiredCodes := []string{"P1", "P2", "P3", "M1", "M2", "IC0"} // 您需要植入的職等代碼
	var fetchedGrades []models.JobGrade
	if errDb := db.Where("code IN ?", requiredCodes).Find(&fetchedGrades).Error; errDb != nil {
		return fmt.Errorf("failed to fetch job grades (%v): %w", requiredCodes, errDb)
	}
	// 將查找到的職等放入 Map 中，方便查找 ID
	for _, jg := range fetchedGrades {
		jobGradesMap[jg.Code] = jg
	}
	// 檢查是否所有必要的職等都找到了
	for _, code := range requiredCodes {
		if _, ok := jobGradesMap[code]; !ok {
			log.Printf("⚠️ Warning: Job grade with code '%s' not found in database. Skipping related assignments.", code)
			// return fmt.Errorf("required job grade '%s' not found", code) // 或者直接報錯
		}
	}
	log.Printf("Fetched %d job grades.", len(jobGradesMap))

	// --- 輔助函數：安全地獲取職等 ID 指標 ---
	getGradeIDPtr := func(code string) *uuid.UUID {
		if grade, ok := jobGradesMap[code]; ok {
			return &grade.ID
		}
		log.Printf("Warning: Job grade '%s' not found, JobGradeID will be nil.", code)
		return nil // 如果找不到對應的 Code，則返回 nil
	}
	// -----------------------------------

	// --- 2. 準備密碼 ---
	defaultPassword := environment.DefaultPassword
	if defaultPassword == "" {
		log.Println("Warning: Default password is empty.")
	}
	hasher := utils.NewBcryptPasswordHasher()
	hashedPassword, hashErr := hasher.HashPassword(defaultPassword)
	if hashErr != nil {
		return fmt.Errorf("failed to hash default password for seeding: %w", hashErr)
	}

	// --- 3. 定義種子用戶資料 (Account + Employment + JobGradeID) ---
	type seedUser struct {
		Account    models.Account
		Employment models.Employment
	}

	seedData := []seedUser{
		// === Super User ===
		{
			Account:    models.Account{FirstName: "Super", LastName: "Admin", Email: "super@example.com", Password: hashedPassword, Role: models.RoleSuperAdmin},
			Employment: models.Employment{HireDate: &hireDate, Status: models.EmploymentStatusActive, PositionTitle: ("System Administrator"), JobGradeID: nil}, // Super Admin 可能沒有職等
		},
		// === HR Users (分配 M1 或 M2 職等) ===
		{
			Account:    models.Account{FirstName: "HR", LastName: "Manager", Email: "hr@example.com", Password: hashedPassword, Role: models.RoleHR},
			Employment: models.Employment{HireDate: &hireDate, Status: models.EmploymentStatusActive, PositionTitle: ("HR Manager"), JobGradeID: getGradeIDPtr("M2")}, // <<< 分配 M2
		},
		{
			Account:    models.Account{FirstName: "Helen", LastName: "Resource", Email: "helen.r@example.com", Password: hashedPassword, Role: models.RoleHR},
			Employment: models.Employment{HireDate: &hireDate, Status: models.EmploymentStatusActive, PositionTitle: ("HR Specialist"), JobGradeID: getGradeIDPtr("M1")}, // <<< 分配 M1
		},
		{
			Account:    models.Account{FirstName: "Harry", LastName: "Rules", Email: "harry.r@example.com", Password: hashedPassword, Role: models.RoleHR},
			Employment: models.Employment{HireDate: &hireDate, Status: models.EmploymentStatusActive, PositionTitle: ("HR Assistant"), JobGradeID: getGradeIDPtr("P3")}, // <<< 分配 P3 (假設)
		},
		// === Regular Employees (分配 P1, P2, P3 職等) ===
		{
			Account:    models.Account{FirstName: "John", LastName: "Doe", Email: "john.doe@example.com", Password: hashedPassword, Role: models.RoleEmployee},
			Employment: models.Employment{HireDate: &hireDate, Status: models.EmploymentStatusActive, PositionTitle: ("Software Engineer"), JobGradeID: getGradeIDPtr("P2")}, // <<< 分配 P2
		},
		{
			Account:    models.Account{FirstName: "Alice", LastName: "Chiang", Email: "alice.chiang@example.com", Password: hashedPassword, Role: models.RoleEmployee},
			Employment: models.Employment{HireDate: &hireDate, Status: models.EmploymentStatusActive, PositionTitle: ("Frontend Developer"), JobGradeID: getGradeIDPtr("P2")}, // <<< 分配 P2
		},
		{
			Account:    models.Account{FirstName: "Danny", LastName: "Paul", Email: "danny.paul@example.com", Password: hashedPassword, Role: models.RoleEmployee},
			Employment: models.Employment{HireDate: &hireDate, Status: models.EmploymentStatusActive, PositionTitle: ("Backend Developer"), JobGradeID: getGradeIDPtr("P3")}, // <<< 分配 P3
		},
		{
			Account:    models.Account{FirstName: "Frank", LastName: "Bessle", Email: "frank.bessle@example.com", Password: hashedPassword, Role: models.RoleEmployee},
			Employment: models.Employment{HireDate: &hireDate, Status: models.EmploymentStatusActive, PositionTitle: ("QA Engineer"), JobGradeID: getGradeIDPtr("P1")}, // <<< 分配 P1
		},
		{
			Account:    models.Account{FirstName: "Grace", LastName: "Hopper", Email: "grace.hopper@example.com", Password: hashedPassword, Role: models.RoleEmployee},
			Employment: models.Employment{HireDate: &hireDate, Status: models.EmploymentStatusActive, PositionTitle: ("System Analyst"), JobGradeID: getGradeIDPtr("P3")}, // <<< 分配 P3
		},
		{
			Account:    models.Account{FirstName: "Alan", LastName: "Turing", Email: "alan.turing@example.com", Password: hashedPassword, Role: models.RoleEmployee},
			Employment: models.Employment{HireDate: &hireDate, Status: models.EmploymentStatusActive, PositionTitle: ("Data Scientist"), JobGradeID: getGradeIDPtr("P3")}, // <<< 分配 P3
		},
	}

	// --- 4. 使用事務創建記錄 ---
	tx := db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin seed transaction for accounts/employments: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		} else if err != nil {
			log.Printf("Rolling back account/employment seed transaction due to error: %v", err)
			tx.Rollback()
		}
	}()

	log.Println("Starting account and employment seeding...")
	createdCount := 0
	skippedCount := 0

	for _, data := range seedData {
		accountData := data.Account
		employmentData := data.Employment // employmentData 已經包含了 JobGradeID

		var existingAccount models.Account
		findErr := tx.Where("email = ?", accountData.Email).First(&existingAccount).Error

		if findErr == nil {
			skippedCount++
			continue
		} // 跳過已存在的帳戶
		if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			err = fmt.Errorf("db error checking %s: %w", accountData.Email, findErr)
			log.Println(err)
			return err
		}

		// 創建 Account
		accountData.ID = uuid.Nil
		if createErr := tx.Create(&accountData).Error; createErr != nil {
			err = fmt.Errorf("failed to create account %s: %w", accountData.Email, createErr)
			log.Println(err)
			return err
		}

		// 創建 Employment (現在包含了 JobGradeID)
		employmentData.AccountID = accountData.ID
		employmentData.ID = uuid.Nil
		if employmentData.Status == "" {
			employmentData.Status = models.EmploymentStatusActive
		}
		if createErr := tx.Create(&employmentData).Error; createErr != nil {
			err = fmt.Errorf("failed to create employment for account %s: %w", accountData.Email, createErr)
			log.Println(err)
			return err
		}

		log.Printf("Created account %s with JobGradeID %v and employment record.", accountData.Email, employmentData.JobGradeID)
		createdCount++
	}

	log.Printf("Account/Employment seeding finished. Created: %d, Skipped: %d.", createdCount, skippedCount)

	// --- 5. 提交事務 ---
	err = tx.Commit().Error
	if err != nil {
		return fmt.Errorf("failed to commit account/employment seed transaction: %w", err)
	}

	log.Println("Account and Employment seeding completed successfully.")
	return nil
}
