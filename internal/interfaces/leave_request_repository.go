package interfaces

import (
	"context"

	"github.com/erinchen11/hr-system/internal/models" // 導入 models 包
	"github.com/google/uuid"
)

// LeaveRequestRepository 定義了與請假申請 (LeaveRequest) 資料庫操作相關的介面
type LeaveRequestRepository interface {
	// Create 創建新的請假申請記錄
	Create(ctx context.Context, request *models.LeaveRequest) error

	// GetByID 根據請假申請的 ID (主鍵) 獲取記錄
	// 實現時應仔細考慮是否需要預加載 (Preload) 關聯數據 (Account, Approver)，
	// 或者將 Preload 的決策留給 Service 層。
	GetByID(ctx context.Context, id uuid.UUID) (*models.LeaveRequest, error)

	// Update 更新現有的請假申請記錄
	// 主要用於 Service 層更新狀態、審批人、審批時間等。
	// 實現時可能使用 GORM 的 Save 或 Updates。
	Update(ctx context.Context, request *models.LeaveRequest) error

	// ListAllWithAccount 列出所有請假申請記錄，並預加載申請人 (Account) 資訊
	// 這個方法名明確了它會包含 Account 資訊。
	// 可擴展以支持過濾和分頁。
	ListAllWithAccount(ctx context.Context) ([]models.LeaveRequest, error)

	// ListByAccountID 列出指定帳戶 (AccountID) 的所有請假申請記錄
	// 通常需要按申請時間排序。
	ListByAccountID(ctx context.Context, accountID uuid.UUID) ([]models.LeaveRequest, error)

	// --- 可能需要的其他方法 ---
	
}