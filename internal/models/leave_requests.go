package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// --- 請假狀態常量 (保持不變) ---
const (
	LeaveStatusPending  = "pending"
	LeaveStatusApproved = "approved"
	LeaveStatusRejected = "rejected"
	// LeaveStatusCancelled = "cancelled" // 如果將來需要
)

// --- 請假類型常量 (修正可能的拼寫錯誤) ---
const (
	LeaveTypeAnnual   = "annual"   // 年假
	LeaveTypeSick     = "sick"     // 病假
	LeaveTypePersonal = "personal" // 事假
	LeaveTypeVacation = "vacation" // 渡假
)

// LeaveRequest 定義了請假申請的模型 (已更新以適應 Account 拆分)
type LeaveRequest struct {
	ID uuid.UUID `gorm:"type:char(36);primaryKey" json:"id"`

	AccountID uuid.UUID `gorm:"type:char(36);not null;index" json:"account_id"` // 申請人帳戶 ID (原 EmployeeID)

	LeaveType string    `gorm:"type:varchar(50);not null;index" json:"leave_type"` // 假別
	StartDate time.Time `gorm:"type:date;not null;index" json:"start_date"`
	EndDate   time.Time `gorm:"type:date;not null;index" json:"end_date"`
	Reason    string    `gorm:"type:text" json:"reason,omitempty"`
	Status    string    `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"`

	ApproverID *uuid.UUID `gorm:"type:char(36);index" json:"approver_id,omitempty"` // 審核人帳戶 ID ( nullable )

	RequestedAt time.Time  `gorm:"column:requested_at;not null;autoCreateTime" json:"requested_at"`
	ApprovedAt  *time.Time `gorm:"index" json:"approved_at,omitempty"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	// --- GORM 關聯 ( 為關聯到 Account) ---
	Account  Account  `gorm:"foreignKey:AccountID" json:"account,omitempty"`   // 關聯到申請人帳戶
	Approver *Account `gorm:"foreignKey:ApproverID" json:"approver,omitempty"` // 關聯到審核人帳戶 (指標因為 ApproverID 可為 NULL)

}

// TableName 指定 GORM 對應的表格名稱 (保持不變)
func (LeaveRequest) TableName() string {
	return "leave_requests"
}

// BeforeCreate GORM Hook (保持不變)
func (lr *LeaveRequest) BeforeCreate(tx *gorm.DB) (err error) {
	if lr.ID == uuid.Nil {
		lr.ID = uuid.New()
	}
	return
}
