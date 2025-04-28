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

// gormJobGradeRepository 實現了 JobGradeRepository 介面
type gormJobGradeRepository struct {
	db *gorm.DB
}

// NewGormJobGradeRepository 是 gormJobGradeRepository 的構造函數
func NewGormJobGradeRepository(db *gorm.DB) interfaces.JobGradeRepository {
	return &gormJobGradeRepository{db: db}
}

// CreateJobGrade 創建新的職等記錄
func (r *gormJobGradeRepository) CreateJobGrade(ctx context.Context, jobGrade *models.JobGrade) error {
	// BeforeCreate Hook 會處理 UUID
	if err := r.db.WithContext(ctx).Create(jobGrade).Error; err != nil {
		// 可以考慮檢查是否為唯一約束錯誤 (e.g., duplicate Code)
		return fmt.Errorf("failed to create job grade: %w", err)
	}
	return nil
}

// GetJobGradeByID 根據 ID 獲取職等記錄
func (r *gormJobGradeRepository) GetJobGradeByID(ctx context.Context, id uuid.UUID) (*models.JobGrade, error) {
	var jobGrade models.JobGrade
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&jobGrade).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound // 直接返回 GORM 的 NotFound 錯誤
		}
		return nil, fmt.Errorf("error fetching job grade by ID %s: %w", id, err)
	}
	return &jobGrade, nil
}

// GetJobGradeByCode 根據職等代碼 (Code) 獲取職等記錄
func (r *gormJobGradeRepository) GetJobGradeByCode(ctx context.Context, code string) (*models.JobGrade, error) {
	var jobGrade models.JobGrade
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&jobGrade).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, fmt.Errorf("error fetching job grade by Code %s: %w", code, err)
	}
	return &jobGrade, nil
}

// UpdateJobGrade 更新現有的職等記錄
func (r *gormJobGradeRepository) UpdateJobGrade(ctx context.Context, jobGrade *models.JobGrade) error {
	// 使用 Save 會更新所有非零值欄位 (基於傳入的 jobGrade struct)
	// 確保 jobGrade.ID 是有效的
	if jobGrade.ID == uuid.Nil {
		return errors.New("cannot update job grade with zero ID")
	}
	result := r.db.WithContext(ctx).Save(jobGrade)
	if result.Error != nil {
		return fmt.Errorf("failed to update job grade %s: %w", jobGrade.ID, result.Error)
	}

	return nil
}

// ListJobGrades 列出所有職等記錄
func (r *gormJobGradeRepository) ListJobGrades(ctx context.Context) ([]models.JobGrade, error) {
	var jobGrades []models.JobGrade
	// 建議加入排序以確保結果順序穩定
	err := r.db.WithContext(ctx).Order("code asc").Find(&jobGrades).Error
	if err != nil {
		// Find 不會因為找不到記錄而返回 gorm.ErrRecordNotFound
		return nil, fmt.Errorf("error fetching job grades: %w", err)
	}
	return jobGrades, nil
}

// DeleteJobGrade 刪除職等記錄
func (r *gormJobGradeRepository) DeleteJobGrade(ctx context.Context, id uuid.UUID) error {
	// 執行物理刪除
	// 警告：如介面註解所述，直接刪除可能有風險，應謹慎使用
	// 在實際應用中，可能需要先檢查是否有 Employment 記錄關聯到此 JobGrade ID
	result := r.db.WithContext(ctx).Delete(&models.JobGrade{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete job grade %s: %w", id, result.Error)
	}
	// 檢查是否有記錄真的被刪除
	if result.RowsAffected == 0 {
		// 如果 RowsAffected 是 0，表示沒有找到該 ID 的記錄
		return gorm.ErrRecordNotFound // 返回未找到錯誤
	}
	return nil
}
