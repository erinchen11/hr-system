package interfaces

import (
	"context"

	"github.com/erinchen11/hr-system/internal/models" 
	"github.com/google/uuid"
)

// AccountRepository 定義了與帳戶 (Account) 資料庫操作相關的介面
type AccountRepository interface {
	// GetAccountByEmail 根據 Email 查詢帳戶
	// 通常用於登入驗證或檢查 Email 是否已存在
	GetAccountByEmail(ctx context.Context, email string) (*models.Account, error)

	// CreateAccount 創建新的帳戶記錄
	// 注意：此方法只負責創建 Account 核心記錄 (包含姓名、Email、密碼、角色等)。
	// 不負責創建關聯的 Employment 記錄，那應該是 Service 層的職責。
	// 密碼應在傳入前由 Service 層進行 Hashing。
	CreateAccount(ctx context.Context, account *models.Account) error

	// GetAccountByID 根據 ID 查詢帳戶
	GetAccountByID(ctx context.Context, id uuid.UUID) (*models.Account, error)

	// UpdatePassword 更新指定帳戶的密碼
	// id 指的是 Account 的 ID。
	UpdatePassword(ctx context.Context, id uuid.UUID, newHashedPassword string) error

	// --- 可能需要的其他方法 ---

}
