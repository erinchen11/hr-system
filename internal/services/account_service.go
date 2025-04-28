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

// accountServiceImpl 實現了 AccountService 介面
type accountServiceImpl struct {
	accountRepo     interfaces.AccountRepository    // *** 使用 AccountRepository ***
	employmentRepo  interfaces.EmploymentRepository // *** 新增 EmploymentRepository 依賴 ***
	pwChecker       interfaces.PasswordChecker
	pwHasher        interfaces.PasswordHasher
	cacheRepo       interfaces.CacheRepository // Cache 依然可能需要，用於 Token 或 Profile 緩存
	defaultPassword string                     // 用於創建帳戶時的預設密碼
	db              *gorm.DB                   // *** 新增: 注入 DB 以便管理事務 ***
}

// NewAccountServiceImpl 構造函數
func NewAccountServiceImpl(
	accountRepo interfaces.AccountRepository,
	employmentRepo interfaces.EmploymentRepository,
	pwChecker interfaces.PasswordChecker,
	pwHasher interfaces.PasswordHasher,
	cacheRepo interfaces.CacheRepository,
	defaultPassword string,
	db *gorm.DB, 
) interfaces.AccountService { // *** 返回 AccountService ***
	return &accountServiceImpl{
		accountRepo:     accountRepo,    
		employmentRepo:  employmentRepo,
		pwChecker:       pwChecker,
		pwHasher:        pwHasher,
		cacheRepo:       cacheRepo,
		defaultPassword: defaultPassword,
		db:              db, 
	}
}

// Authenticate 方法現在操作 Account (來自之前的 auth_service 範例，整合進來)
func (s *accountServiceImpl) Authenticate(ctx context.Context, email, password string) (*models.Account, error) {
	account, err := s.accountRepo.GetAccountByEmail(ctx, email)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("Error fetching account by email '%s' during auth: %v", email, err)
		}
		return nil, ErrInvalidCredentials
	}

	isValidPassword := s.pwChecker.CheckPassword(account.Password, password)
	if !isValidPassword {
		return nil, ErrInvalidCredentials
	}

	// 返回前清除密碼是個好習慣，雖然 Handler 層也應該做
	account.Password = ""
	return account, nil
}

// ChangePassword 實現 密碼的業務邏輯
// ***  操作 Account ***
func (s *accountServiceImpl) ChangePassword(ctx context.Context, accountID uuid.UUID, oldPassword, newPassword string) error {
	// 1. 獲取帳戶資訊
	account, err := s.accountRepo.GetAccountByID(ctx, accountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAccountNotFound // 使用新的錯誤常量
		}
		log.Printf("Error fetching account %s for password change: %v", accountID, err)
		return fmt.Errorf("failed to retrieve account data")
	}

	// 2. 驗證舊密碼
	if !s.pwChecker.CheckPassword(account.Password, oldPassword) {
		return ErrInvalidCredentials
	}

	// 3. Hash 新密碼
	hashedNewPassword, err := s.pwHasher.HashPassword(newPassword)
	if err != nil {
		log.Printf("Error hashing new password for account %s: %v", accountID, err)
		return ErrPasswordHashingFailed
	}

	// 4. 更新資料庫中的密碼
	err = s.accountRepo.UpdatePassword(ctx, accountID, hashedNewPassword)
	if err != nil {
		// Repository 層現在應該在 RowsAffected=0 時返回 ErrRecordNotFound
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("Attempted to update password for non-existent account %s", accountID)
			return ErrAccountNotFound
		}
		log.Printf("Error updating password for account %s: %v", accountID, err)
		return ErrPasswordUpdateFailed
	}

	return nil
}

// CreateAccountWithEmployment 創建帳戶和對應的初始僱傭記錄
func (s *accountServiceImpl) CreateAccountWithEmployment(ctx context.Context, acc *models.Account, emp *models.Employment) (*models.Account, error) {
	// 1. 使用事務確保原子性
	tx := s.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		log.Printf("Failed to begin transaction for creating account: %v", tx.Error)
		return nil, fmt.Errorf("failed to start transaction")
	}
	// Defer Rollback in case of panic or error
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r) // re-panic after rollback
		} else if tx.Error != nil {
			log.Printf("Rolling back transaction due to error: %v", tx.Error)
			tx.Rollback()
		}
	}()

	// 2. 檢查 Email 是否已存在 (使用注入的 accountRepo)
	//    注意：這裡使用 tx 來執行事務內的操作
	_, err := s.accountRepo.GetAccountByEmail(tx.Statement.Context, acc.Email) // 在事務中檢查
	if err == nil {
		// 如果 err 是 nil，表示找到了現有帳戶
		tx.Rollback() // 不需要繼續了，回滾事務
		return nil, ErrEmailExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		// 如果是其他資料庫錯誤
		log.Printf("Error checking email existence for %s: %v", acc.Email, err)
		tx.Rollback() // 回滾事務
		return nil, fmt.Errorf("database error checking email existence")
	}

	// 3. Hashing 密碼 (如果傳入的 Account 物件還沒有密碼)
	//    通常密碼應該由 Handler 層處理或在此處使用預設密碼
	if acc.Password == "" {
		if s.defaultPassword == "" {
			tx.Rollback()
			log.Println("Cannot create account: default password is not configured and no password provided.")
			return nil, errors.New("cannot create account without a password")
		}
		hashedPassword, hashErr := s.pwHasher.HashPassword(s.defaultPassword)
		if hashErr != nil {
			tx.Rollback()
			log.Printf("Error hashing default password: %v", hashErr)
			return nil, ErrPasswordHashingFailed
		}
		acc.Password = hashedPassword
	}

	//    或者直接調用 GORM 方法：
	acc.ID = uuid.Nil // 確保觸發 BeforeCreate hook
	if err := tx.Create(acc).Error; err != nil {
		tx.Rollback()
		log.Printf("Error creating account in database for email %s: %v", acc.Email, err)
		return nil, ErrAccountCreationFailed
	}

	// 5. 創建 Employment 記錄 (使用注入的 employmentRepo)
	emp.AccountID = acc.ID                       // 將 Account 的 ID 關聯給 Employment
	emp.ID = uuid.Nil                            // 確保觸發 BeforeCreate hook
	if err := tx.Create(emp).Error; err != nil { // 直接使用 tx 創建 Employment
		tx.Rollback()
		log.Printf("Error creating employment record for account %s: %v", acc.ID, err)
		return nil, ErrEmploymentCreationFailed
	}

	// 6. 提交事務
	if err := tx.Commit().Error; err != nil {
		log.Printf("Failed to commit transaction for account creation %s: %v", acc.Email, err)
		// 注意：提交失敗，但之前的操作可能已部分寫入 (雖然不太可能在 Commit 失敗)
		return nil, fmt.Errorf("failed to finalize account creation: %w", err)
	}

	// 7. 創建成功，返回創建的帳戶資訊 (清除密碼)
	acc.Password = ""
	return acc, nil
}

// GetAccount 從 Redis 快取優先獲取帳戶資料，找不到才從資料庫撈
func (s *accountServiceImpl) GetAccount(ctx context.Context, accountID uuid.UUID) (*models.Account, error) {
	cacheKey := fmt.Sprintf("user_profile:%s", accountID.String())

	// 1. 先從 Redis 讀取
	var cachedProfile models.Account
	err := s.cacheRepo.Get(ctx, cacheKey, &cachedProfile)
	if err == nil {
		// 成功從 Redis 命中
		log.Printf("Cache hit for user profile: %s", accountID)
		return &cachedProfile, nil
	}

	log.Printf("Cache miss for user profile: %s, fallback to database...", accountID)

	// 2. 從 DB 查詢
	account, err := s.accountRepo.GetAccountByID(ctx, accountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAccountNotFound
		}
		log.Printf("Error fetching profile for account %s: %v", accountID, err)
		return nil, fmt.Errorf("failed to retrieve account profile")
	}

	// 清除密碼敏感資訊
	account.Password = ""

	// 3. 回寫 Redis 快取，設過期時間 (1小時)
	if cacheErr := s.cacheRepo.Set(ctx, cacheKey, account, time.Hour); cacheErr != nil {
		log.Printf("Warning: Failed to cache user profile %s: %v", accountID, cacheErr)
	}

	return account, nil
}

func (s *accountServiceImpl) GetJobGradeByCode(ctx context.Context, code string) (*models.JobGrade, error) {
	var jobGrade models.JobGrade
	if err := s.db.WithContext(ctx).Where("code = ?", code).First(&jobGrade).Error; err != nil {
		return nil, err
	}
	return &jobGrade, nil
}

// ... 其他 AccountService 方法的實現 ...
