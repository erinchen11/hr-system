package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/erinchen11/hr-system/internal/interfaces/mocks" 
	"github.com/erinchen11/hr-system/internal/models"
	"github.com/google/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gomock "github.com/golang/mock/gomock"
	"gorm.io/gorm" 
)

func TestLeaveRequestServiceImpl_ListAllRequests(t *testing.T) {
	ctx := context.Background()

	// Mock data setup
	accountID1 := uuid.New()
	accountID2 := uuid.New()
	mockAccount1 := models.Account{ID: accountID1, Password: "hash1"} 
	mockAccount2 := models.Account{ID: accountID2, Password: "hash2"}
	mockRequests := []models.LeaveRequest{
		{ID: uuid.New(), AccountID: accountID1, Account: mockAccount1, Status: models.LeaveStatusPending},
		{ID: uuid.New(), AccountID: accountID2, Account: mockAccount2, Status: models.LeaveStatusApproved, ApproverID: &accountID1, Approver: &mockAccount1}, // Example with approver
	}

	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockLeaveRepo := mocks.NewMockLeaveRequestRepository(ctrl)
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl) // Needed for New
		service := NewLeaveRequestServiceImpl(mockLeaveRepo, mockAccountRepo)

		mockLeaveRepo.EXPECT().ListAllWithAccount(gomock.Any()).Return(mockRequests, nil).Times(1)

		// Execute
		requests, err := service.ListAllRequests(ctx)

		// Assert
		require.NoError(t, err)
		require.Len(t, requests, 2)
		assert.Equal(t, mockRequests[0].ID, requests[0].ID)
		assert.Equal(t, mockRequests[1].ID, requests[1].ID)
		// Assert passwords are cleared
		assert.Equal(t, "", requests[0].Account.Password)
		assert.NotNil(t, requests[1].Approver)
		assert.Equal(t, "", requests[1].Approver.Password)
	})

	t.Run("Success - Empty List", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockLeaveRepo := mocks.NewMockLeaveRequestRepository(ctrl)
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		service := NewLeaveRequestServiceImpl(mockLeaveRepo, mockAccountRepo)

		emptyRequests := []models.LeaveRequest{}
		mockLeaveRepo.EXPECT().ListAllWithAccount(gomock.Any()).Return(emptyRequests, nil).Times(1)

		requests, err := service.ListAllRequests(ctx)

		require.NoError(t, err)
		assert.Len(t, requests, 0)
		assert.Empty(t, requests)
	})

	t.Run("Failure - Repository Error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockLeaveRepo := mocks.NewMockLeaveRequestRepository(ctrl)
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		service := NewLeaveRequestServiceImpl(mockLeaveRepo, mockAccountRepo)

		repoError := errors.New("db connection error")
		mockLeaveRepo.EXPECT().ListAllWithAccount(gomock.Any()).Return(nil, repoError).Times(1)

		requests, err := service.ListAllRequests(ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to retrieve leave requests from repository")
		assert.ErrorIs(t, err, repoError) // Check wrapped error
		assert.Nil(t, requests)
	})
}

func TestLeaveRequestServiceImpl_ApproveRequest(t *testing.T) {
	ctx := context.Background()
	leaveRequestID := uuid.New()
	processorAccountID := uuid.New()
	applicantAccountID := uuid.New()

	pendingRequest := &models.LeaveRequest{
		ID:        leaveRequestID,
		AccountID: applicantAccountID,
		Status:    models.LeaveStatusPending, // Correct initial state
	}

	nonPendingRequest := &models.LeaveRequest{
		ID:        leaveRequestID,
		AccountID: applicantAccountID,
		Status:    models.LeaveStatusApproved, // Already approved
	}
	hrAccount := &models.Account{ID: processorAccountID, Role: models.RoleHR}
	nonHrAccount := &models.Account{ID: processorAccountID, Role: models.RoleEmployee}

	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockLeaveRepo := mocks.NewMockLeaveRequestRepository(ctrl)
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		service := NewLeaveRequestServiceImpl(mockLeaveRepo, mockAccountRepo)
		localPendingRequest := *pendingRequest // Use copy

		// 1. Expect processor validation
		mockAccountRepo.EXPECT().GetAccountByID(gomock.Any(), gomock.Eq(processorAccountID)).Return(hrAccount, nil).Times(1)
		// 2. Expect fetching the leave request
		mockLeaveRepo.EXPECT().GetByID(gomock.Any(), gomock.Eq(leaveRequestID)).Return(&localPendingRequest, nil).Times(1)
		// 3. Expect updating the leave request
		mockLeaveRepo.EXPECT().Update(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, req *models.LeaveRequest) error {
				assert.Equal(t, leaveRequestID, req.ID)
				assert.Equal(t, models.LeaveStatusApproved, req.Status)
				require.NotNil(t, req.ApproverID)
				assert.Equal(t, processorAccountID, *req.ApproverID)
				require.NotNil(t, req.ApprovedAt)
				assert.WithinDuration(t, time.Now(), *req.ApprovedAt, time.Second*2) // Check ApprovedAt is recent
				return nil
			}).Times(1)

		// Execute
		err := service.ApproveRequest(ctx, leaveRequestID.String(), processorAccountID.String())

		// Assert
		require.NoError(t, err)
	})

	t.Run("Failure - Invalid Processor ID", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockLeaveRepo := mocks.NewMockLeaveRequestRepository(ctrl)
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		service := NewLeaveRequestServiceImpl(mockLeaveRepo, mockAccountRepo)

		err := service.ApproveRequest(ctx, leaveRequestID.String(), "invalid-uuid")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid processor account identifier format")
	})

	t.Run("Failure - Processor Not Found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockLeaveRepo := mocks.NewMockLeaveRequestRepository(ctrl)
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		service := NewLeaveRequestServiceImpl(mockLeaveRepo, mockAccountRepo)

		mockAccountRepo.EXPECT().GetAccountByID(gomock.Any(), gomock.Eq(processorAccountID)).Return(nil, gorm.ErrRecordNotFound).Times(1)

		err := service.ApproveRequest(ctx, leaveRequestID.String(), processorAccountID.String())
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidProcessor)
	})

	t.Run("Failure - Processor Role Insufficient", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockLeaveRepo := mocks.NewMockLeaveRequestRepository(ctrl)
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		service := NewLeaveRequestServiceImpl(mockLeaveRepo, mockAccountRepo)

		mockAccountRepo.EXPECT().GetAccountByID(gomock.Any(), gomock.Eq(processorAccountID)).Return(nonHrAccount, nil).Times(1)

		err := service.ApproveRequest(ctx, leaveRequestID.String(), processorAccountID.String())
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidProcessor)
	})

	t.Run("Failure - Leave Request Not Found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockLeaveRepo := mocks.NewMockLeaveRequestRepository(ctrl)
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		service := NewLeaveRequestServiceImpl(mockLeaveRepo, mockAccountRepo)

		mockAccountRepo.EXPECT().GetAccountByID(gomock.Any(), gomock.Eq(processorAccountID)).Return(hrAccount, nil).Times(1)
		mockLeaveRepo.EXPECT().GetByID(gomock.Any(), gomock.Eq(leaveRequestID)).Return(nil, gorm.ErrRecordNotFound).Times(1)

		err := service.ApproveRequest(ctx, leaveRequestID.String(), processorAccountID.String())
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrLeaveRequestNotFound)
	})

	t.Run("Failure - Invalid State (Not Pending)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockLeaveRepo := mocks.NewMockLeaveRequestRepository(ctrl)
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		service := NewLeaveRequestServiceImpl(mockLeaveRepo, mockAccountRepo)
		localNonPendingRequest := *nonPendingRequest // Use copy

		mockAccountRepo.EXPECT().GetAccountByID(gomock.Any(), gomock.Eq(processorAccountID)).Return(hrAccount, nil).Times(1)
		mockLeaveRepo.EXPECT().GetByID(gomock.Any(), gomock.Eq(leaveRequestID)).Return(&localNonPendingRequest, nil).Times(1)

		err := service.ApproveRequest(ctx, leaveRequestID.String(), processorAccountID.String())
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidLeaveRequestState)
	})

	t.Run("Failure - Update Error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockLeaveRepo := mocks.NewMockLeaveRequestRepository(ctrl)
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		service := NewLeaveRequestServiceImpl(mockLeaveRepo, mockAccountRepo)
		localPendingRequest := *pendingRequest // Use copy
		updateError := errors.New("db update failed")

		mockAccountRepo.EXPECT().GetAccountByID(gomock.Any(), gomock.Eq(processorAccountID)).Return(hrAccount, nil).Times(1)
		mockLeaveRepo.EXPECT().GetByID(gomock.Any(), gomock.Eq(leaveRequestID)).Return(&localPendingRequest, nil).Times(1)
		mockLeaveRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(updateError).Times(1)

		err := service.ApproveRequest(ctx, leaveRequestID.String(), processorAccountID.String())
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrLeaveRequestUpdateFailed)
	})
}

// --- Test ApplyForLeave ---
func TestLeaveRequestServiceImpl_ApplyForLeave(t *testing.T) {
	ctx := context.Background()
	accountID := uuid.New()
	accountIDStr := accountID.String()
	leaveType := models.LeaveTypePersonal
	startDate := time.Now().AddDate(0, 0, 5)
	endDate := time.Now().AddDate(0, 0, 7)
	reason := "Family matter"

	mockAccount := &models.Account{ID: accountID, Role: models.RoleEmployee} // Applicant account

	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockLeaveRepo := mocks.NewMockLeaveRequestRepository(ctrl)
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		service := NewLeaveRequestServiceImpl(mockLeaveRepo, mockAccountRepo)

		// 1. Expect Account check
		mockAccountRepo.EXPECT().GetAccountByID(gomock.Any(), gomock.Eq(accountID)).Return(mockAccount, nil).Times(1)
		// 2. Expect Create call
		mockLeaveRepo.EXPECT().Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, req *models.LeaveRequest) error {
				assert.Equal(t, accountID, req.AccountID)
				assert.Equal(t, leaveType, req.LeaveType) // 檢查假單類型 "personal"
				assert.Equal(t, startDate, req.StartDate)
				assert.Equal(t, endDate, req.EndDate)
				assert.Equal(t, reason, req.Reason) // 檢查請假原因 "Family matter"
				assert.Equal(t, models.LeaveStatusPending, req.Status)
				req.ID = uuid.New()
				req.RequestedAt = time.Now()
				return nil
			}).Times(1)

		// Execute
		createdRequest, err := service.ApplyForLeave(ctx, accountIDStr, leaveType, reason, startDate, endDate)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, createdRequest)
		assert.NotEqual(t, uuid.Nil, createdRequest.ID)
		assert.Equal(t, accountID, createdRequest.AccountID)
		assert.Equal(t, leaveType, createdRequest.LeaveType)
		assert.Equal(t, models.LeaveStatusPending, createdRequest.Status)
	})

	t.Run("Failure - Invalid Account ID Format", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockLeaveRepo := mocks.NewMockLeaveRequestRepository(ctrl)
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		service := NewLeaveRequestServiceImpl(mockLeaveRepo, mockAccountRepo)

		createdRequest, err := service.ApplyForLeave(ctx, "invalid-uuid", leaveType, reason, startDate, endDate)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user identifier format")
		assert.Nil(t, createdRequest)
	})

	t.Run("Failure - Account Not Found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockLeaveRepo := mocks.NewMockLeaveRequestRepository(ctrl)
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		service := NewLeaveRequestServiceImpl(mockLeaveRepo, mockAccountRepo)

		mockAccountRepo.EXPECT().GetAccountByID(gomock.Any(), gomock.Eq(accountID)).Return(nil, gorm.ErrRecordNotFound).Times(1)

		createdRequest, err := service.ApplyForLeave(ctx, accountIDStr, leaveType, reason, startDate, endDate)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAccountNotFound) // Expect service level error
		assert.Nil(t, createdRequest)
	})

	t.Run("Failure - Invalid Date Range", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockLeaveRepo := mocks.NewMockLeaveRequestRepository(ctrl)
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		service := NewLeaveRequestServiceImpl(mockLeaveRepo, mockAccountRepo)
		invalidEndDate := startDate.AddDate(0, 0, -1) // End date before start date

		// Account check should still happen before date validation
		mockAccountRepo.EXPECT().GetAccountByID(gomock.Any(), gomock.Eq(accountID)).Return(mockAccount, nil).Times(1)
		// Create should NOT be called

		createdRequest, err := service.ApplyForLeave(ctx, accountIDStr, leaveType, reason, startDate, invalidEndDate)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidDateRange)
		assert.Nil(t, createdRequest)
	})

	t.Run("Failure - Create Repo Error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockLeaveRepo := mocks.NewMockLeaveRequestRepository(ctrl)
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		service := NewLeaveRequestServiceImpl(mockLeaveRepo, mockAccountRepo)
		repoError := errors.New("db create failed")

		mockAccountRepo.EXPECT().GetAccountByID(gomock.Any(), gomock.Eq(accountID)).Return(mockAccount, nil).Times(1)
		mockLeaveRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(repoError).Times(1)

		createdRequest, err := service.ApplyForLeave(ctx, accountIDStr, leaveType, reason, startDate, endDate)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrLeaveApplyFailed)
		assert.Nil(t, createdRequest)
	})
}

// --- Test ListAccountRequests ---
func TestLeaveRequestServiceImpl_ListAccountRequests(t *testing.T) {
	ctx := context.Background()
	accountID := uuid.New()
	accountIDStr := accountID.String()

	mockAccount := &models.Account{ID: accountID, Role: models.RoleEmployee}
	mockRequests := []models.LeaveRequest{
		{ID: uuid.New(), AccountID: accountID, Status: models.LeaveStatusPending, Account: models.Account{Password: "hash1"}},
		{ID: uuid.New(), AccountID: accountID, Status: models.LeaveStatusApproved, Account: models.Account{Password: "hash2"}, Approver: &models.Account{Password: "hash3"}},
	}

	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockLeaveRepo := mocks.NewMockLeaveRequestRepository(ctrl)
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		service := NewLeaveRequestServiceImpl(mockLeaveRepo, mockAccountRepo)

		// 1. Expect account validation
		mockAccountRepo.EXPECT().GetAccountByID(gomock.Any(), gomock.Eq(accountID)).Return(mockAccount, nil).Times(1)
		// 2. Expect list call
		mockLeaveRepo.EXPECT().ListByAccountID(gomock.Any(), gomock.Eq(accountID)).Return(mockRequests, nil).Times(1)

		// Execute
		requests, err := service.ListAccountRequests(ctx, accountIDStr)

		// Assert
		require.NoError(t, err)
		require.Len(t, requests, 2)
		assert.Equal(t, mockRequests[0].ID, requests[0].ID)
		// Assert passwords cleared
		assert.Equal(t, "", requests[0].Account.Password)
		assert.NotNil(t, requests[1].Approver)
		assert.Equal(t, "", requests[1].Approver.Password)
	})

	t.Run("Failure - Account Not Found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockLeaveRepo := mocks.NewMockLeaveRequestRepository(ctrl)
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		service := NewLeaveRequestServiceImpl(mockLeaveRepo, mockAccountRepo)

		mockAccountRepo.EXPECT().GetAccountByID(gomock.Any(), gomock.Eq(accountID)).Return(nil, gorm.ErrRecordNotFound).Times(1)
		// ListByAccountID should NOT be called

		requests, err := service.ListAccountRequests(ctx, accountIDStr)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAccountNotFound)
		assert.Nil(t, requests)
	})

	t.Run("Failure - List Repo Error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockLeaveRepo := mocks.NewMockLeaveRequestRepository(ctrl)
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		service := NewLeaveRequestServiceImpl(mockLeaveRepo, mockAccountRepo)
		repoError := errors.New("db list error")

		mockAccountRepo.EXPECT().GetAccountByID(gomock.Any(), gomock.Eq(accountID)).Return(mockAccount, nil).Times(1)
		mockLeaveRepo.EXPECT().ListByAccountID(gomock.Any(), gomock.Eq(accountID)).Return(nil, repoError).Times(1)

		requests, err := service.ListAccountRequests(ctx, accountIDStr)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to retrieve leave requests")
		assert.Nil(t, requests)
	})

}

// --- Test GetLeaveRequestByID ---
func TestLeaveRequestServiceImpl_GetLeaveRequestByID(t *testing.T) {
	ctx := context.Background()
	leaveRequestID := uuid.New()
	leaveRequestIDStr := leaveRequestID.String()
	accountID := uuid.New()
	approverID := uuid.New()

	mockRequest := &models.LeaveRequest{
		ID:         leaveRequestID,
		AccountID:  accountID,
		Status:     models.LeaveStatusApproved,
		ApproverID: &approverID,
		Account:    models.Account{ID: accountID, Password: "hash1"}, // Include associated data with passwords
		Approver:   &models.Account{ID: approverID, Password: "hash2"},
	}

	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockLeaveRepo := mocks.NewMockLeaveRequestRepository(ctrl)
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		service := NewLeaveRequestServiceImpl(mockLeaveRepo, mockAccountRepo)

		mockLeaveRepo.EXPECT().GetByID(gomock.Any(), gomock.Eq(leaveRequestID)).Return(mockRequest, nil).Times(1)

		request, err := service.GetLeaveRequestByID(ctx, leaveRequestIDStr)

		require.NoError(t, err)
		require.NotNil(t, request)
		assert.Equal(t, leaveRequestID, request.ID)
		// Assert passwords cleared
		assert.Equal(t, "", request.Account.Password)
		require.NotNil(t, request.Approver)
		assert.Equal(t, "", request.Approver.Password)
	})

	t.Run("Failure - Invalid ID Format", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockLeaveRepo := mocks.NewMockLeaveRequestRepository(ctrl)
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		service := NewLeaveRequestServiceImpl(mockLeaveRepo, mockAccountRepo)

		request, err := service.GetLeaveRequestByID(ctx, "not-a-uuid")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid leave request identifier format")
		assert.Nil(t, request)
	})

	t.Run("Failure - Not Found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockLeaveRepo := mocks.NewMockLeaveRequestRepository(ctrl)
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		service := NewLeaveRequestServiceImpl(mockLeaveRepo, mockAccountRepo)

		mockLeaveRepo.EXPECT().GetByID(gomock.Any(), gomock.Eq(leaveRequestID)).Return(nil, gorm.ErrRecordNotFound).Times(1)

		request, err := service.GetLeaveRequestByID(ctx, leaveRequestIDStr)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrLeaveRequestNotFound)
		assert.Nil(t, request)
	})

	t.Run("Failure - DB Error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockLeaveRepo := mocks.NewMockLeaveRequestRepository(ctrl)
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		service := NewLeaveRequestServiceImpl(mockLeaveRepo, mockAccountRepo)
		dbError := errors.New("get by id db error")

		mockLeaveRepo.EXPECT().GetByID(gomock.Any(), gomock.Eq(leaveRequestID)).Return(nil, dbError).Times(1)

		request, err := service.GetLeaveRequestByID(ctx, leaveRequestIDStr)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to retrieve leave request")
		assert.Nil(t, request)
	})
}
