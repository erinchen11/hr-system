package services

import (
	"context"
	"errors"
	"log"

	"github.com/erinchen11/hr-system/internal/interfaces"
	"github.com/erinchen11/hr-system/internal/models"
	"gorm.io/gorm"                                    
)
// --- AuthService 的具體實現 ---

// authServiceImpl 結構體，實現了 AuthService 介面
type authServiceImpl struct {
	accountRepo interfaces.AccountRepository // 依賴 AccountRepository 介面
	pwChecker   interfaces.PasswordChecker   // 依賴 PasswordChecker 介面
}

// NewAuthServiceImpl 是 authServiceImpl 的構造函數
// 接收 AccountRepository 和 PasswordChecker 介面作為參數
// 返回 AuthService 介面類型
func NewAuthServiceImpl(
	accountRepo interfaces.AccountRepository,
	pwChecker interfaces.PasswordChecker,
) interfaces.AuthService { // 返回 AuthService 介面
	return &authServiceImpl{
		accountRepo: accountRepo,
		pwChecker:   pwChecker,
	}
}

// Authenticate 方法實現了 AuthService 介面的 Authenticate 方法
// 驗證用戶郵箱和密碼，成功返回 Account 資訊
func (s *authServiceImpl) Authenticate(ctx context.Context, email, password string) (*models.Account, error) {
	// 1. 使用注入的 accountRepo 根據 Email 查找帳戶
	account, err := s.accountRepo.GetAccountByEmail(ctx, email)
	if err != nil {

		if !errors.Is(err, gorm.ErrRecordNotFound) {
			// 但如果是 RecordNotFound 以外的錯誤，我們應該記錄下來以供排查。
			log.Printf("Error fetching account by email '%s' during auth: %v", email, err)
		}
		return nil, ErrInvalidCredentials // 統一返回無效憑證
	}

	// 2. 如果找到了帳戶，使用注入的 pwChecker 驗證密碼
	isValidPassword := s.pwChecker.CheckPassword(account.Password, password)
	if !isValidPassword {
		// 密碼不匹配，同樣返回 "無效憑證"
		return nil, ErrInvalidCredentials
	}

	// 3. 郵箱存在且密碼匹配，認證成功，返回帳戶資訊
	// 清除密碼 HASH 是個好習慣，避免將其洩漏到上層或日誌中
	account.Password = ""
	return account, nil
}

// 注意：AuthService 介面目前只有 Authenticate 方法。
// 如果將來有其他與「認證/授權」相關的業務邏輯（例如：登出、刷新 Token、檢查權限等），
// 可以添加到 AuthService 介面和這個 authServiceImpl 實現中。
