package interfaces

import (
	"context"

	"github.com/erinchen11/hr-system/internal/models" 
	"github.com/google/uuid"
)

// AccountService 定義了與帳戶相關的業務邏輯操作
type AccountService interface {
	// Authenticate 驗證使用者 Email 和密碼，成功返回 Account 資訊
	Authenticate(ctx context.Context, email, password string) (*models.Account, error)

	// ChangePassword 更改指定帳戶的密碼
	ChangePassword(ctx context.Context, accountID uuid.UUID, oldPassword, newPassword string) error

	// CreateAccountWithEmployment 創建一個新的帳戶及其初始僱傭記錄
	// 需要提供帳戶基本資訊 (name, email) 和僱傭資訊 (role, etc.)
	// 返回創建成功的 Account (密碼已清除)
	CreateAccountWithEmployment(ctx context.Context, acc *models.Account, emp *models.Employment) (*models.Account, error)

	// GetAccount 獲取指定 ID 的帳戶資訊 (可能包含關聯的 Employment，取決於實現)
	GetAccount(ctx context.Context, accountID uuid.UUID) (*models.Account, error)
	GetJobGradeByCode(ctx context.Context, code string) (*models.JobGrade, error)

	// // --- 可能需要的其他方法 ---

}
