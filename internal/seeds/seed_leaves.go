// 檔案路徑: internal/seeds/seed_leave.go
package seeds

import (
	"errors" // 導入 errors
	"fmt"    // 導入 fmt
	"log"
	"time"

	"github.com/erinchen11/hr-system/internal/models"
	"github.com/erinchen11/hr-system/internal/utils"

	// "github.com/erinchen11/hr-system/internal/utils" // 如果 utils.Ptr 不在 utils 中，則不需要導入
	"gorm.io/gorm"
)

func SeedLeaveRequests(db *gorm.DB) (err error) { // Named return err
	log.Println("Starting leave request seeding...")

	// 1. 查找需要的 Account 記錄
	johnDoe, errJohn := findAccountByEmail(db, "john.doe@example.com")
	aliceChiang, errAlice := findAccountByEmail(db, "alice.chiang@example.com")
	dannyPaul, errDanny := findAccountByEmail(db, "danny.paul@example.com")
	graceHopper, errGrace := findAccountByEmail(db, "grace.hopper@example.com")
	alanTuring, errAlan := findAccountByEmail(db, "alan.turing@example.com")
	hrManager, errHR := findAccountByEmail(db, "hr@example.com") // Approver

	// 檢查是否有任何必要的帳戶未找到
	if errJohn != nil || errAlice != nil || errDanny != nil || errGrace != nil || errAlan != nil || errHR != nil {
		log.Printf("Cannot seed leave requests because some accounts were not found. Errors: John[%v], Alice[%v], Danny[%v], Grace[%v], Alan[%v], HR[%v]",
			errJohn, errAlice, errDanny, errGrace, errAlan, errHR)
		return errors.New("required accounts for leave request seeding not found")
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// 2. 定義多樣化的請假記錄
	requests := []models.LeaveRequest{
		{ // John Doe - Annual Leave - Approved
			// ***  : EmployeeID -> AccountID ***
			AccountID:  johnDoe.ID,
			LeaveType:  models.LeaveTypeAnnual,
			StartDate:  today.AddDate(0, 0, 7),
			EndDate:    today.AddDate(0, 0, 9),
			Status:     models.LeaveStatusApproved,
			Reason:     "Vacation",                     // ***  : 使用本地 utils.Ptr (假設) ***
			ApproverID: utils.Ptr(hrManager.ID),        // ***  : 使用本地 utils.Ptr (假設) ***
			ApprovedAt: utils.Ptr(now.Add(-time.Hour)), // ***  : 使用本地 utils.Ptr (假設) ***
		},
		{ // Alice Chiang - Sick Leave - Pending
			AccountID: aliceChiang.ID, // ***   ***
			LeaveType: models.LeaveTypeSick,
			StartDate: today.AddDate(0, 0, 1),
			EndDate:   today.AddDate(0, 0, 1),
			Status:    models.LeaveStatusPending,
			Reason:    "Feeling unwell",
		},
		{ // Danny Paul - Personal Leave - Rejected
			AccountID:  dannyPaul.ID, // ***   ***
			LeaveType:  models.LeaveTypePersonal,
			StartDate:  today.AddDate(0, 1, 0),
			EndDate:    today.AddDate(0, 1, 0),
			Status:     models.LeaveStatusRejected,
			Reason:     "Personal appointment",                // ***   ***
			ApproverID: utils.Ptr(hrManager.ID),               // ***   ***
			ApprovedAt: utils.Ptr(now.Add(-30 * time.Minute)), // ***   ***
		},
		{ // Grace Hopper - Vacation - Pending
			AccountID: graceHopper.ID, // ***   ***
			// ***  : LeaveTypeSick -> LeaveTypeVacation ***
			LeaveType: models.LeaveTypeVacation,
			StartDate: today.AddDate(0, 2, 1),
			EndDate:   today.AddDate(0, 2, 10),
			Status:    models.LeaveStatusPending,
			Reason:    "Long holiday trip", // ***   ***
		},
		{ // Alan Turing - Annual Leave - Pending
			AccountID: alanTuring.ID, // ***   ***
			LeaveType: models.LeaveTypeAnnual,
			StartDate: today.AddDate(0, 0, 14),
			EndDate:   today.AddDate(0, 0, 15),
			Status:    models.LeaveStatusPending,
			Reason:    "Short break", // ***   ***
		},
		{ // John Doe - Sick Leave - Approved (Past)
			AccountID:  johnDoe.ID, // ***   ***
			LeaveType:  models.LeaveTypeSick,
			StartDate:  today.AddDate(0, -1, 1),
			EndDate:    today.AddDate(0, -1, 2),
			Status:     models.LeaveStatusApproved,
			Reason:     "Flu recovery",                     // ***   ***
			ApproverID: utils.Ptr(hrManager.ID),            // ***   ***
			ApprovedAt: utils.Ptr(today.AddDate(0, -1, 3)), // ***   ***
		},
	}

	// 3. 使用事務創建記錄
	tx := db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin seed transaction for leave requests: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		} else if err != nil {
			log.Printf("Rolling back leave request seed transaction due to error: %v", err)
			tx.Rollback()
		}
	}()

	createdCount := 0
	for _, req := range requests {
		// BeforeCreate handles ID. autoCreateTime handles RequestedAt.
		if createErr := tx.Create(&req).Error; createErr != nil {
			// ***  : 使用 req.AccountID ***
			err = fmt.Errorf("failed to create leave request for account %s (%s): %w", req.AccountID, req.LeaveType, createErr)
			log.Println(err)
			return err // 觸發 Rollback
		}
		createdCount++
	}

	log.Printf("Leave request seeding finished. Created: %d.", createdCount)

	// 4. 提交事務
	err = tx.Commit().Error
	if err != nil {
		return fmt.Errorf("failed to commit leave request seed transaction: %w", err)
	}

	log.Println("Leave request seeding completed successfully.")
	return nil
}
