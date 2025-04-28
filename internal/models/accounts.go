package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Account 定義了系統帳戶的核心身份、登入和基礎權限資訊
type Account struct {
	ID          uuid.UUID `gorm:"type:char(36);primaryKey" json:"id"`
	FirstName   string    `gorm:"type:varchar(50);not null" json:"first_name"`         // Consider if this belongs here or in a separate profile if truly needed
	LastName    string    `gorm:"type:varchar(50);not null;index" json:"last_name"`    // Consider if this belongs here
	Email       string    `gorm:"type:varchar(100);not null;uniqueIndex" json:"email"` // Login ID
	Password    string    `gorm:"type:varchar(255);not null" json:"-"`                 // Auth credential
	Role        uint8     `gorm:"type:tinyint unsigned;not null;index" json:"role"`    // 0:super, 1:hr, 2:employee
	PhoneNumber string    `gorm:"type:varchar(20)" json:"phone_number,omitempty"`      // Nullable contact info
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`

}

// TableName 指定 GORM 對應的表格名稱
func (Account) TableName() string {
	return "accounts" // Renamed table
}

// BeforeCreate GORM Hook: 在建立記錄前自動產生 UUID
func (a *Account) BeforeCreate(tx *gorm.DB) (err error) {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return
}

// Constants for Role (could be in a central constants file)
const (
	RoleSuperAdmin uint8 = 0
	RoleHR         uint8 = 1
	RoleEmployee   uint8 = 2
)
