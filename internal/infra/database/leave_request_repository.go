// 檔案路徑: internal/infra/database/leave_request_repository.go
package database

import (
	"context"
	"errors"
	"log" // 用於記錄錯誤

	"github.com/erinchen11/hr-system/internal/interfaces"
	"github.com/erinchen11/hr-system/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// gormLeaveRequestRepository 實現了 LeaveRequestRepository 介面
type gormLeaveRequestRepository struct {
	db *gorm.DB
}

// NewGormLeaveRequestRepository 構造函數
func NewGormLeaveRequestRepository(db *gorm.DB) interfaces.LeaveRequestRepository {
	return &gormLeaveRequestRepository{db: db}
}

// ListAllWithEmployee 實現獲取所有請假記錄（含員工資訊）
func (r *gormLeaveRequestRepository) ListAllWithAccount(ctx context.Context) ([]models.LeaveRequest, error) {
	var requests []models.LeaveRequest
	if err := r.db.WithContext(ctx).
		Preload("Account").
		Preload("Approver").
		Find(&requests).Error; err != nil {
		return nil, err
	}
	return requests, nil
}

// GetByID 根據 ID 獲取請假單
func (r *gormLeaveRequestRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.LeaveRequest, error) {
	var request models.LeaveRequest
	// 預加載 Employee 資訊可能也有用，取決于 Service 是否需要
	err := r.db.WithContext(ctx).Preload("Account").Preload("Approver").First(&request, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound // 返回 GORM 錯誤
		}
		log.Printf("Error fetching leave request by ID %s: %v", id, err)
		return nil, err // 返回其他錯誤
	}
	return &request, nil
}

// Update 更新請假單記錄
func (r *gormLeaveRequestRepository) Update(ctx context.Context, request *models.LeaveRequest) error {
	// 使用 Save 會更新所有欄位，適用於從 GetByID 獲取狀態等欄位
	// GORM 的 Save 會自動處理主鍵，執行更新操作
	// 它也會自動更新 UpdatedAt 欄位 (如果模型中有 gorm:"autoUpdateTime")
	err := r.db.WithContext(ctx).Save(request).Error
	if err != nil {
		log.Printf("Error updating leave request %s: %v", request.ID, err)
		// 可以檢查是否是記錄不存在的錯誤 (雖然 Save 通常不報)
		// if errors.Is(err, gorm.ErrRecordNotFound) { return gorm.ErrRecordNotFound }
	}
	// 檢查 RowsAffected 可能更可靠些，如果 Save 對象沒有主鍵會插入
	// 但在此場景下是先 Get 再 Save，所以對象一定有 ID
	return err // 可包裝錯誤
}

// Create 創建新的請假記錄
func (r *gormLeaveRequestRepository) Create(ctx context.Context, request *models.LeaveRequest) error {
	// BeforeCreate Hook 會處理 ID 和 RequestedAt (如果模型中配置了 autoCreateTime)
	err := r.db.WithContext(ctx).Create(request).Error
	if err != nil {
		// 可以檢查特定錯誤，例如外鍵約束失敗
		log.Printf("Error creating leave request for employee %s: %v", request.AccountID, err)
		// 返回原始錯誤或包裝後的錯誤
	}
	return err
}

// ListByAccountID 根據員工 ID 查詢其所有請假記錄
func (r *gormLeaveRequestRepository) ListByAccountID(ctx context.Context, accountID uuid.UUID) ([]models.LeaveRequest, error) {
	var requests []models.LeaveRequest
	// 根據 employee_id 查詢，可以按申請時間排序
	// 不需要 Preload Employee，因為是員工自己查詢
	err := r.db.WithContext(ctx).Where("account_id = ?", accountID).Order("requested_at desc").Find(&requests).Error
	if err != nil {
		// Find 在找不到記錄時不返回 ErrRecordNotFound，而是返回空 slice 和 nil error
		log.Printf("Error fetching leave requests for employee %s: %v", accountID, err)
		return nil, err // 返回查詢錯誤
	}
	// 如果沒找到，requests 會是空的 slice `[]`，err 是 nil
	return requests, nil
}

