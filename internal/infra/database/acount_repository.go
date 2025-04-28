package database

import (
	"context"
	"errors"

	"github.com/erinchen11/hr-system/internal/interfaces"
	"github.com/erinchen11/hr-system/internal/models" 
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// gormAccountRepository 實現了 AccountRepository 介面
type gormAccountRepository struct {
	db *gorm.DB
}

// NewGormAccountRepository 是 gormAccountRepository 的構造函數
// 注意：返回類型應為 interfaces.AccountRepository
func NewGormAccountRepository(db *gorm.DB) interfaces.AccountRepository {
	return &gormAccountRepository{db: db}
}

// GetAccountByEmail 根據 Email 查詢帳戶 (原 GetUserByEmail)
func (r *gormAccountRepository) GetAccountByEmail(ctx context.Context, email string) (*models.Account, error) {
	var account models.Account // 使用 models.Account
	// GORM 會根據 Account 模型的 TableName() 方法查詢 'accounts' 表
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&account).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 返回標準錯誤，Service 層可以統一處理 Not Found
			// (或者根據需要返回您在 interfaces 中定義的 ErrNotFound)
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err // 返回其他資料庫錯誤
	}
	return &account, nil
}

// CreateAccount 創建新的帳戶記錄 (原 CreateEmployee)
// 注意：此方法現在只負責創建 Account 記錄。
// 密碼應已在 Service 層 Hashing。
// 創建關聯的 Employment 記錄是 Service 層的職責。
func (r *gormAccountRepository) CreateAccount(ctx context.Context, account *models.Account) error {
	// BeforeCreate Hook 會處理 UUID
	err := r.db.WithContext(ctx).Create(account).Error
	// 考慮錯誤包裝或特定錯誤檢查 (例如 unique email constraint violation)
	return err
}

// GetAccountByID 根據 ID 查詢帳戶 (原 GetEmployeeByID)
func (r *gormAccountRepository) GetAccountByID(ctx context.Context, id uuid.UUID) (*models.Account, error) {
	var account models.Account // 使用 models.Account
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&account).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return &account, nil
}

// UpdatePassword 更新指定帳戶的密碼
func (r *gormAccountRepository) UpdatePassword(ctx context.Context, id uuid.UUID, newHashedPassword string) error {
	// 使用 Model(&models.Account{}) 指定要更新 'accounts' 表
	result := r.db.WithContext(ctx).Model(&models.Account{}).Where("id = ?", id).Update("password", newHashedPassword)
	if result.Error != nil {
		return result.Error
	}
	// 檢查是否有記錄被更新
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound // 返回未找到錯誤
	}
	return nil
}
