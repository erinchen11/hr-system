package seeds

import (
	"errors"
	"fmt"

	"github.com/erinchen11/hr-system/internal/models"
	"gorm.io/gorm"
)

// findAccountByEmail 根據 Email 查找 Account 記錄
func findAccountByEmail(db *gorm.DB, email string) (*models.Account, error) {
	var account models.Account // ***  : 使用 models.Account ***
	// GORM 會根據 account 的類型查詢 accounts 表
	if err := db.Where("email = ?", email).First(&account).Error; err != nil {
		// *** 建議: 添加更清晰的錯誤包裝 (同 seed_leave 中的用法) ***
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("account with email '%s' not found: %w", email, err)
		}
		return nil, fmt.Errorf("error finding account '%s': %w", email, err)
	}
	// ***  : 返回 *models.Account ***
	return &account, nil
}
