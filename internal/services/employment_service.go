package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/erinchen11/hr-system/internal/interfaces"
	"github.com/erinchen11/hr-system/internal/models" 
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// employmentServiceImpl 實現了 EmploymentService 介面
type employmentServiceImpl struct {
	employmentRepo interfaces.EmploymentRepository
	accountRepo    interfaces.AccountRepository // 可能需要用來驗證 Account 狀態
	
}

// NewEmploymentServiceImpl 構造函數
func NewEmploymentServiceImpl(
	employmentRepo interfaces.EmploymentRepository,
	accountRepo interfaces.AccountRepository, // 注入依賴
	
) interfaces.EmploymentService {
	return &employmentServiceImpl{
		employmentRepo: employmentRepo,
		accountRepo:    accountRepo,
	}
}

// GetEmploymentByAccountID 根據 Account ID 獲取當前有效的僱傭資訊
func (s *employmentServiceImpl) GetEmploymentByAccountID(ctx context.Context, accountID uuid.UUID) (*models.Employment, error) {
	employment, err := s.employmentRepo.GetEmploymentByAccountID(ctx, accountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {

			return nil, ErrEmploymentNotFound
		}
		log.Printf("Error fetching employment for account %s: %v", accountID, err)
		return nil, fmt.Errorf("database error fetching employment record")
	}

	return employment, nil
}

// GetEmploymentByID 根據 Employment 記錄自身的 ID 獲取資訊
func (s *employmentServiceImpl) GetEmploymentByID(ctx context.Context, employmentID uuid.UUID) (*models.Employment, error) {
	employment, err := s.employmentRepo.GetEmploymentByID(ctx, employmentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEmploymentNotFound
		}
		log.Printf("Error fetching employment by ID %s: %v", employmentID, err)
		return nil, fmt.Errorf("database error fetching employment record")
	}
	// 可選: Preload
	return employment, nil
}

// UpdateEmploymentDetails 更新僱傭記錄的詳細資訊
func (s *employmentServiceImpl) UpdateEmploymentDetails(ctx context.Context, employmentID uuid.UUID, updates *models.Employment) (*models.Employment, error) {
	// 1. 先獲取現有的記錄
	existingEmp, err := s.employmentRepo.GetEmploymentByID(ctx, employmentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEmploymentNotFound
		}
		log.Printf("Error fetching employment %s for update: %v", employmentID, err)
		return nil, fmt.Errorf("failed to retrieve employment record for update")
	}

	// 2. 檢查是否允許更新 (例如，不能更新已離職的記錄)
	if existingEmp.Status == models.EmploymentStatusTerminated {
		return nil, fmt.Errorf("cannot update terminated employment record")
	}

	// 3. 將 updates 中的允許更新的欄位複製到 existingEmp

	updated := false
	if updates.JobGradeID != nil && existingEmp.JobGradeID != updates.JobGradeID {
		// 可選：驗證 JobGradeID 是否存在 (需要 jobGradeRepo)
		existingEmp.JobGradeID = updates.JobGradeID
		updated = true
	}
	if updates.PositionTitle != "" && existingEmp.PositionTitle != updates.PositionTitle {
		existingEmp.PositionTitle = updates.PositionTitle
		updated = true
	}
	if updates.Salary != nil && (existingEmp.Salary == nil || !existingEmp.Salary.Equals(*updates.Salary)) {
		// 可選：驗證 Salary 是否在 JobGrade 的範圍內 (需要 jobGradeRepo)
		existingEmp.Salary = updates.Salary
		updated = true
	}
	// 其他允許更新的欄位...
	// 例如 Status，但可能需要單獨的方法來處理狀態變更及其副作用

	// 如果沒有任何欄位需要更新
	if !updated {
		log.Printf("No fields to update for employment %s", employmentID)
		// 可以選擇返回原記錄或 nil, nil 或特定錯誤
		return existingEmp, nil // 返回原記錄表示沒有變化
	}

	// 4. 調用 Repository 更新資料庫
	//    傳遞 後的 existingEmp 物件給 UpdateEmployment
	err = s.employmentRepo.UpdateEmployment(ctx, existingEmp)
	if err != nil {
		// UpdateEmployment 內部應處理 NotFound 的情況
		log.Printf("Error updating employment %s in repository: %v", employmentID, err)
		return nil, ErrUpdateFailed
	}

	// 5. 返回更新後的記錄
	return existingEmp, nil
}

// TerminateEmployment 處理員工離職
func (s *employmentServiceImpl) TerminateEmployment(ctx context.Context, employmentID uuid.UUID, terminationDate time.Time) error {
	// 1. 獲取記錄
	employment, err := s.employmentRepo.GetEmploymentByID(ctx, employmentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrEmploymentNotFound
		}
		log.Printf("Error fetching employment %s for termination: %v", employmentID, err)
		return fmt.Errorf("failed to retrieve employment record for termination")
	}

	// 2. 檢查狀態
	if employment.Status == models.EmploymentStatusTerminated {
		log.Printf("Employment record %s is already terminated.", employmentID)
		return ErrAlreadyTerminated // 或者直接返回 nil 表示操作已完成
	}

	// 3. 更新狀態和離職日期
	employment.Status = models.EmploymentStatusTerminated
	employment.TerminationDate = &terminationDate // 設置離職日期

	// 4. 調用 Repository 更新
	err = s.employmentRepo.UpdateEmployment(ctx, employment)
	if err != nil {
		// UpdateEmployment 內部應處理 NotFound
		log.Printf("Error terminating employment %s in repository: %v", employmentID, err)
		return ErrTerminationFailed
	}

	log.Printf("Employment record %s terminated successfully.", employmentID)
	return nil
}

// ListEmployments 列出僱傭記錄
func (s *employmentServiceImpl) ListEmployments(ctx context.Context /*, filters, pagination */) ([]models.Employment, error) {
	employments, err := s.employmentRepo.ListEmployments(ctx)
	if err != nil {
		log.Printf("Error listing employments: %v", err)
		// 根據需要包裝錯誤
		return nil, fmt.Errorf("failed to list employment records")
	}
	// 可選: 在這裡 Preload 相關資訊，或應用過濾/分頁邏輯 (如果 Repo 沒做)
	return employments, nil
}
