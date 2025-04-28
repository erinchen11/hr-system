package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// JobGrade 定義了公司內部的職級/職等
type JobGrade struct {
	ID          uuid.UUID       `gorm:"type:char(36);primaryKey" json:"id"`
	Code        string          `gorm:"type:varchar(20);uniqueIndex;not null" json:"code"`           // 職等代碼 (e.g., P1, M1)
	Name        string          `gorm:"type:varchar(50);not null" json:"name"`                       // 職等名稱 (e.g., Engineer Level 1)
	Description string          `gorm:"type:text" json:"description,omitempty"`                      // 職等描述 (可選)
	MinSalary   decimal.Decimal `gorm:"type:decimal(12,2);default:0.00" json:"min_salary,omitempty"` // 最低薪資 (可選) - 注意 omitempty
	MaxSalary   decimal.Decimal `gorm:"type:decimal(12,2);default:0.00" json:"max_salary,omitempty"` // 最高薪資 (可選) - 注意 omitempty
	CreatedAt   time.Time       `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time       `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 指定 GORM 對應的表格名稱
func (JobGrade) TableName() string {
	return "job_grades"
}

// BeforeCreate GORM Hook: 在建立記錄前自動產生 UUID
func (jg *JobGrade) BeforeCreate(tx *gorm.DB) (err error) {
	if jg.ID == uuid.Nil {
		jg.ID = uuid.New()
	}
	return
}
