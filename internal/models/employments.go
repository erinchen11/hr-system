package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// --- 僱傭狀態常量 (保持不變) ---
const (
	EmploymentStatusActive     = "active"
	EmploymentStatusOnLeave    = "on_leave"
	EmploymentStatusTerminated = "terminated"
)

// Employment 定義了員工的僱傭詳細資訊
type Employment struct {
	ID              uuid.UUID        `gorm:"type:char(36);primaryKey" json:"id"`
	AccountID       uuid.UUID        `gorm:"type:char(36);not null;index" json:"account_id"`                 // *** FK renamed to AccountID ***
	JobGradeID      *uuid.UUID       `gorm:"type:char(36);index" json:"job_grade_id,omitempty"`              // FK to JobGrade, nullable
	PositionTitle   string           `gorm:"type:varchar(50)" json:"position_title,omitempty"`               // 具體職稱, 可為 NULL
	Salary          *decimal.Decimal `gorm:"type:decimal(12,2)" json:"salary,omitempty"`                     // 薪資, 可為 NULL
	HireDate        *time.Time       `gorm:"type:date;index" json:"hire_date,omitempty"`                     // 入職日期, 可為 NULL
	TerminationDate *time.Time       `gorm:"type:date;index" json:"termination_date,omitempty"`              // 離職日期, 可為 NULL
	Status          string           `gorm:"type:varchar(20);not null;default:'active';index" json:"status"` // 僱傭狀態
	CreatedAt       time.Time        `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time        `gorm:"autoUpdateTime" json:"updated_at"`

	// --- GORM 關聯 (屬於 Belongs To) ---
	Account  Account   `gorm:"foreignKey:AccountID" json:"account,omitempty"`
	JobGrade *JobGrade `gorm:"foreignKey:JobGradeID" json:"job_grade,omitempty"`
}

// TableName 指定 GORM 對應的表格名稱
func (Employment) TableName() string {
	return "employments"
}

// BeforeCreate GORM Hook: 在建立記錄前自動產生 UUID
func (e *Employment) BeforeCreate(tx *gorm.DB) (err error) {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	if e.Status == "" {
		e.Status = EmploymentStatusActive
	}
	return
}
