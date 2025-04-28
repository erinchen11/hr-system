package interfaces

import (
	"context"
	"time"

	"github.com/erinchen11/hr-system/internal/models" 
	"github.com/google/uuid"
)

// EmploymentService 定義了與僱傭資訊相關的業務邏輯操作
type EmploymentService interface {
	// GetEmploymentByAccountID 根據 Account ID 獲取當前有效的僱傭資訊
	GetEmploymentByAccountID(ctx context.Context, accountID uuid.UUID) (*models.Employment, error)

	// GetEmploymentByID 根據 Employment 記錄自身的 ID 獲取資訊
	GetEmploymentByID(ctx context.Context, employmentID uuid.UUID) (*models.Employment, error)

	// UpdateEmploymentDetails 更新僱傭記錄的詳細資訊 (例如: 職等、職稱、薪資)
	// 傳入要更新的 employmentID 以及包含新資訊的 models.Employment 物件
	// 具體實現需要決定哪些欄位允許被此方法更新
	UpdateEmploymentDetails(ctx context.Context, employmentID uuid.UUID, updates *models.Employment) (*models.Employment, error)

	// TerminateEmployment 處理員工離職
	// 設定離職日期和狀態
	TerminateEmployment(ctx context.Context, employmentID uuid.UUID, terminationDate time.Time) error

	// ListEmployments 列出僱傭記錄 (可擴展以支持過濾和分頁)
	ListEmployments(ctx context.Context /*, filters, pagination */) ([]models.Employment, error)

	// --- 可能需要的其他方法 ---

}
