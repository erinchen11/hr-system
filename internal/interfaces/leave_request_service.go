package interfaces

import (
	"context"
	"time" // 導入 time 套件

	"github.com/erinchen11/hr-system/internal/models" 
)

// LeaveRequestService 定義了與請假申請相關的業務邏輯操作
type LeaveRequestService interface {
	// ListAllRequests 獲取所有請假請求 (通常帶有關聯的 Account)
	ListAllRequests(ctx context.Context) ([]models.LeaveRequest, error)

	// ApproveRequest 批准指定的請假申請
	ApproveRequest(ctx context.Context, leaveRequestIDStr string, processorAccountIDStr string) error

	// RejectRequest 拒絕指定的請假申請
	RejectRequest(ctx context.Context, leaveRequestIDStr string, processorAccountIDStr string, reason string) error

	// ApplyForLeave 員工提交新的請假申請
	// ***  後的方法簽名 ***
	// 添加了 leaveType string 參數
	ApplyForLeave(ctx context.Context, accountIDStr, leaveType, reason string, startDate, endDate time.Time) (*models.LeaveRequest, error)

	// ListAccountRequests 列出指定帳戶的所有請假申請
	// ***  後的方法名 ***
	ListAccountRequests(ctx context.Context, accountIDStr string) ([]models.LeaveRequest, error)

	// GetLeaveRequestByID 根據 ID 獲取單個請假申請 (新增)
	// 這個方法通常很有用，例如在批准/拒絕前獲取詳細資訊
	GetLeaveRequestByID(ctx context.Context, leaveRequestIDStr string) (*models.LeaveRequest, error)
}
