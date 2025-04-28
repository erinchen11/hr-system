package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/erinchen11/hr-system/internal/interfaces"
	"github.com/erinchen11/hr-system/internal/models" 
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// gormEmploymentRepository 實現了 EmploymentRepository 介面
type gormEmploymentRepository struct {
	db *gorm.DB
}

// NewGormEmploymentRepository 是 gormEmploymentRepository 的構造函數
func NewGormEmploymentRepository(db *gorm.DB) interfaces.EmploymentRepository {
	return &gormEmploymentRepository{db: db}
}

// CreateEmployment 創建新的僱傭記錄
func (r *gormEmploymentRepository) CreateEmployment(ctx context.Context, employment *models.Employment) error {
	// BeforeCreate Hook 會處理 UUID 和預設 Status
	if err := r.db.WithContext(ctx).Create(employment).Error; err != nil {
		// 可以考慮檢查特定錯誤，例如外鍵約束失敗 (AccountID 不存在)
		// 或者唯一約束失敗 (如果 AccountID 被設為 unique)
		return fmt.Errorf("failed to create employment record: %w", err)
	}
	return nil
}

// GetEmploymentByAccountID 根據 Account ID 獲取僱傭記錄
// 假設目前只關心最新的/主要的僱傭記錄 (如果有多筆)
// 注意：這裡沒有 Preload Account 或 JobGrade，Service 層可以根據需要決定是否 Preload
func (r *gormEmploymentRepository) GetEmploymentByAccountID(ctx context.Context, accountID uuid.UUID) (*models.Employment, error) {
	var employment models.Employment
	// 根據 account_id 查找。如果未來支持歷史記錄，可能需要加入 status='active' 或其他排序邏輯
	err := r.db.WithContext(ctx).Where("account_id = ?", accountID).First(&employment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound // 直接返回 GORM 的 NotFound 錯誤
		}
		return nil, fmt.Errorf("error fetching employment by AccountID %s: %w", accountID, err)
	}
	return &employment, nil
}

// GetEmploymentByID 根據僱傭記錄自身的 ID (主鍵) 獲取記錄
// 注意：這裡沒有 Preload Account 或 JobGrade
func (r *gormEmploymentRepository) GetEmploymentByID(ctx context.Context, id uuid.UUID) (*models.Employment, error) {
	var employment models.Employment
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&employment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, fmt.Errorf("error fetching employment by ID %s: %w", id, err)
	}
	return &employment, nil
}

// UpdateEmployment 更新僱傭記錄
func (r *gormEmploymentRepository) UpdateEmployment(ctx context.Context, employment *models.Employment) error {
	// 使用 Save 更新所有非零值欄位，確保 employment.ID 有效
	if employment.ID == uuid.Nil {
		return errors.New("cannot update employment record with zero ID")
	}
	// GORM 的 Save 會自動更新 UpdatedAt (如果有 autoUpdateTime tag)
	result := r.db.WithContext(ctx).Save(employment)
	if result.Error != nil {
		return fmt.Errorf("failed to update employment record %s: %w", employment.ID, result.Error)
	}
	// 檢查是否有記錄被更新
	if result.RowsAffected == 0 {
		// 如果 Save 沒報錯，但 RowsAffected 為 0，表示找不到對應 ID 的記錄
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ListEmployments 列出僱傭記錄
// 注意：這裡沒有 Preload Account 或 JobGrade
func (r *gormEmploymentRepository) ListEmployments(ctx context.Context /*, filterOptions, paginationOptions */) ([]models.Employment, error) {
	var employments []models.Employment
	// 建議加入排序，例如按創建時間或 AccountID
	err := r.db.WithContext(ctx).Order("created_at desc").Find(&employments).Error
	if err != nil {
		// Find 不會因為找不到記錄而返回 gorm.ErrRecordNotFound
		return nil, fmt.Errorf("error fetching employments: %w", err)
	}
	// 如果沒有記錄，會返回空的 slice 和 nil error
	return employments, nil
}

// GetEmploymentCountByJobGradeID 根據 JobGrade ID 計算使用該職等的僱傭記錄數量
// 注意：這裡假設查詢所有狀態的僱傭記錄，您可能只想計算 'active' 狀態的。
func (r *gormEmploymentRepository) GetEmploymentCountByJobGradeID(ctx context.Context, jobGradeID uuid.UUID) (int64, error) {
	var count int64
	// 使用 Model 指定查詢 employments 表，並用 Where 過濾 job_grade_id
	err := r.db.WithContext(ctx).Model(&models.Employment{}).Where("job_grade_id = ?", jobGradeID).Count(&count).Error
	if err != nil {
		// Count 出錯時，GORM 可能不會返回 ErrRecordNotFound，直接返回錯誤
		return 0, fmt.Errorf("error counting employments for job grade %s: %w", jobGradeID, err)
	}
	return count, nil
}
