package interfaces

import (
	"context"

	"github.com/erinchen11/hr-system/internal/models"
	"github.com/google/uuid"
)

// EmploymentRepository 定義了與員工僱傭資訊 (Employment) 資料庫操作相關的介面
type EmploymentRepository interface {
	// CreateEmployment 創建新的僱傭記錄
	// 通常在創建 Account 後，由 Service 層在同一個事務中調用。
	CreateEmployment(ctx context.Context, employment *models.Employment) error

	// GetEmploymentByAccountID 根據 Account ID 獲取僱傭記錄
	// 假設目前一個帳戶只有一筆有效的僱傭記錄。
	// 如果未來需要支持僱傭歷史記錄，此方法的返回值或參數可能需要調整。
	GetEmploymentByAccountID(ctx context.Context, accountID uuid.UUID) (*models.Employment, error)

	// GetEmploymentByID 根據僱傭記錄自身的 ID (主鍵) 獲取記錄
	GetEmploymentByID(ctx context.Context, id uuid.UUID) (*models.Employment, error)

	// UpdateEmployment 更新僱傭記錄
	// 可以用於更新職等、職稱、薪資、狀態、離職日期等。
	// 實現時可能使用 GORM 的 Save (更新所有欄位) 或 Updates (更新指定欄位)。
	UpdateEmployment(ctx context.Context, employment *models.Employment) error

	// ListEmployments 列出僱傭記錄
	// 基礎版本，可擴展以支持過濾 (例如依狀態、部門、職等) 和分頁。
	// 根據使用場景，可能需要在實現中 Preload("Account") 或 Preload("JobGrade")。
	ListEmployments(ctx context.Context /*, filterOptions, paginationOptions */) ([]models.Employment, error)

	GetEmploymentCountByJobGradeID(ctx context.Context, jobGradeID uuid.UUID) (int64, error)
	// --- 可能需要的其他方法 ---

}
