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

// leaveRequestServiceImpl 實現了 LeaveRequestService 介面
type leaveRequestServiceImpl struct {
	leaveRepo   interfaces.LeaveRequestRepository
	accountRepo interfaces.AccountRepository
}

// NewLeaveRequestServiceImpl 構造函數
func NewLeaveRequestServiceImpl(
	leaveRepo interfaces.LeaveRequestRepository,
	accountRepo interfaces.AccountRepository,
) interfaces.LeaveRequestService { // 返回介面類型
	return &leaveRequestServiceImpl{
		leaveRepo:   leaveRepo,
		accountRepo: accountRepo,
	}
}

// ListAllRequests 實現獲取所有請假請求的業務邏輯
func (s *leaveRequestServiceImpl) ListAllRequests(ctx context.Context) ([]models.LeaveRequest, error) {
	requests, err := s.leaveRepo.ListAllWithAccount(ctx) // 假設 Repo 方法已更名並 Preload
	if err != nil {
		log.Printf("Service error fetching leave requests: %v", err)
		return nil, fmt.Errorf("failed to retrieve leave requests from repository: %w", err)
	}
	for i := range requests {
		if requests[i].Account.Password != "" {
			requests[i].Account.Password = ""
		}
		if requests[i].Approver != nil && requests[i].Approver.Password != "" {
			requests[i].Approver.Password = ""
		}
	}
	return requests, nil
}

// ApproveRequest 實現批准請假單的業務邏輯
func (s *leaveRequestServiceImpl) ApproveRequest(ctx context.Context, leaveRequestIDStr string, processorAccountIDStr string) error {
	leaveRequestUUID, err := uuid.Parse(leaveRequestIDStr)
	if err != nil {
		return errors.New("invalid leave request identifier format")
	}
	processorAccountUUID, err := uuid.Parse(processorAccountIDStr)
	if err != nil {
		return errors.New("invalid processor account identifier format")
	}

	processor, err := s.accountRepo.GetAccountByID(ctx, processorAccountUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrInvalidProcessor
		}
		log.Printf("Error fetching processor account %s: %v", processorAccountUUID, err)
		return fmt.Errorf("failed to verify processor account")
	}
	if processor.Role != models.RoleHR && processor.Role != models.RoleSuperAdmin {
		log.Printf("Account %s (Role: %d) does not have permission to approve leave requests", processorAccountUUID, processor.Role)
		return ErrInvalidProcessor
	}

	request, err := s.leaveRepo.GetByID(ctx, leaveRequestUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrLeaveRequestNotFound
		}
		log.Printf("Error fetching leave request %s for approval: %v", leaveRequestUUID, err)
		return fmt.Errorf("failed to retrieve leave request data")
	}

	if request.Status != models.LeaveStatusPending {
		log.Printf("Attempted to approve leave request %s with status %s", request.ID, request.Status)
		return ErrInvalidLeaveRequestState
	}

	now := time.Now()
	request.Status = models.LeaveStatusApproved
	request.ApproverID = &processorAccountUUID
	request.ApprovedAt = &now

	err = s.leaveRepo.Update(ctx, request)
	if err != nil {
		log.Printf("Error updating leave request %s status to approved: %v", request.ID, err)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrLeaveRequestNotFound
		}
		return ErrLeaveRequestUpdateFailed
	}
	return nil
}

// RejectRequest 實現拒絕請假單的業務邏輯
func (s *leaveRequestServiceImpl) RejectRequest(ctx context.Context, leaveRequestIDStr string, processorAccountIDStr string, reason string) error {
	leaveRequestUUID, err := uuid.Parse(leaveRequestIDStr)
	if err != nil {
		return errors.New("invalid leave request identifier format")
	}
	processorAccountUUID, err := uuid.Parse(processorAccountIDStr)
	if err != nil {
		return errors.New("invalid processor account identifier format")
	}

	processor, err := s.accountRepo.GetAccountByID(ctx, processorAccountUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrInvalidProcessor
		}
		log.Printf("Error fetching processor account %s: %v", processorAccountUUID, err)
		return fmt.Errorf("failed to verify processor account")
	}
	if processor.Role != models.RoleHR && processor.Role != models.RoleSuperAdmin {
		log.Printf("Account %s (Role: %d) does not have permission to reject leave requests", processorAccountUUID, processor.Role)
		return ErrInvalidProcessor
	}

	request, err := s.leaveRepo.GetByID(ctx, leaveRequestUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrLeaveRequestNotFound
		}
		log.Printf("Error fetching leave request %s for rejection: %v", leaveRequestUUID, err)
		return fmt.Errorf("failed to retrieve leave request data")
	}

	if request.Status != models.LeaveStatusPending {
		log.Printf("Attempted to reject leave request %s with status %s", request.ID, request.Status)
		return ErrInvalidLeaveRequestState
	}

	now := time.Now()
	request.Status = models.LeaveStatusRejected
	request.ApproverID = &processorAccountUUID
	request.ApprovedAt = &now
	if reason != "" {
		request.Reason = reason
	} else {
		request.Reason = ""
	}

	err = s.leaveRepo.Update(ctx, request)
	if err != nil {
		log.Printf("Error updating leave request %s status to rejected: %v", request.ID, err)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrLeaveRequestNotFound
		}
		return ErrLeaveRequestUpdateFailed
	}
	return nil
}

// ApplyForLeave 實現提交請假申請的業務邏輯
func (s *leaveRequestServiceImpl) ApplyForLeave(ctx context.Context, accountIDStr, leaveType, reason string, startDate, endDate time.Time) (*models.LeaveRequest, error) {
	accountUUID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return nil, errors.New("invalid user identifier format")
	}

	_, err = s.accountRepo.GetAccountByID(ctx, accountUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAccountNotFound
		}
		log.Printf("Error verifying applying account %s: %v", accountUUID, err)
		return nil, fmt.Errorf("failed to verify applicant account")
	}

	if endDate.Before(startDate) {
		return nil, ErrInvalidDateRange
	}

	leaveRequest := &models.LeaveRequest{
		AccountID: accountUUID,
		LeaveType: leaveType,
		StartDate: startDate,
		EndDate:   endDate,
		Reason:    reason,
		Status:    models.LeaveStatusPending,
	}

	err = s.leaveRepo.Create(ctx, leaveRequest)
	if err != nil {
		log.Printf("Error creating leave request for account %s: %v", accountUUID, err)
		return nil, ErrLeaveApplyFailed
	}
	return leaveRequest, nil
}

// ListAccountRequests 實現獲取特定帳戶請假列表的業務邏輯
func (s *leaveRequestServiceImpl) ListAccountRequests(ctx context.Context, accountIDStr string) ([]models.LeaveRequest, error) {
	accountUUID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return nil, errors.New("invalid user identifier format")
	}

	_, err = s.accountRepo.GetAccountByID(ctx, accountUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAccountNotFound
		}
		log.Printf("Error verifying account %s for listing requests: %v", accountUUID, err)
		return nil, fmt.Errorf("failed to verify account")
	}

	requests, err := s.leaveRepo.ListByAccountID(ctx, accountUUID) // 假設 Repo 方法已更名
	if err != nil {
		log.Printf("Service error fetching leave requests for account %s: %v", accountUUID, err)
		return nil, fmt.Errorf("failed to retrieve leave requests")
	}

	for i := range requests {
		// 安全地清除關聯模型的密碼
		if requests[i].Account.Password != "" {
			requests[i].Account.Password = ""
		}
		if requests[i].Approver != nil && requests[i].Approver.Password != "" {
			requests[i].Approver.Password = ""
		}
	}
	return requests, nil
}

// *** 新增 GetLeaveRequestByID 方法的實現 ***
func (s *leaveRequestServiceImpl) GetLeaveRequestByID(ctx context.Context, leaveRequestIDStr string) (*models.LeaveRequest, error) {
	// 1. 解析 ID
	leaveRequestUUID, err := uuid.Parse(leaveRequestIDStr)
	if err != nil {
		log.Printf("Error parsing leave request ID '%s': %v", leaveRequestIDStr, err)
		return nil, errors.New("invalid leave request identifier format")
	}

	// 2. 調用 Repository 獲取數據
	request, err := s.leaveRepo.GetByID(ctx, leaveRequestUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLeaveRequestNotFound // 使用 Service 層錯誤
		}
		log.Printf("Error fetching leave request by ID %s: %v", leaveRequestUUID, err)
		return nil, fmt.Errorf("failed to retrieve leave request")
	}

	// 3. (可選) 清除可能預加載的敏感資訊
	if request.Account.Password != "" {
		request.Account.Password = ""
	}
	if request.Approver != nil && request.Approver.Password != "" {
		request.Approver.Password = ""
	}

	// 4. 返回結果
	return request, nil
}

// ... 其他 Service 方法的實現 ...
