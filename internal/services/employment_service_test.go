package services

import (
	"context"
	"errors"
	"testing"
	"time"

	
	"github.com/erinchen11/hr-system/internal/interfaces/mocks" 
	"github.com/erinchen11/hr-system/internal/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal" 

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gomock "github.com/golang/mock/gomock"
	"gorm.io/gorm" 
)

func TestEmploymentServiceImpl_GetEmploymentByAccountID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEmploymentRepo := mocks.NewMockEmploymentRepository(ctrl)
	mockAccountRepo := mocks.NewMockAccountRepository(ctrl) // 雖然未使用，但 New 需要
	// *** service 在函數頂層宣告並在子測試中使用 ***
	service := NewEmploymentServiceImpl(mockEmploymentRepo, mockAccountRepo)

	ctx := context.Background()
	testAccountID := uuid.New()
	testEmploymentID := uuid.New()
	mockEmployment := &models.Employment{
		ID:        testEmploymentID,
		AccountID: testAccountID,
		Status:    models.EmploymentStatusActive,
	}

	t.Run("Success", func(t *testing.T) {
		mockEmploymentRepo.EXPECT().GetEmploymentByAccountID(gomock.Any(), gomock.Eq(testAccountID)).Return(mockEmployment, nil).Times(1)
		employment, err := service.GetEmploymentByAccountID(ctx, testAccountID) // 使用頂層 service
		require.NoError(t, err)
		require.NotNil(t, employment)
		assert.Equal(t, testEmploymentID, employment.ID)
		assert.Equal(t, testAccountID, employment.AccountID)
	})

	t.Run("Failure - Not Found", func(t *testing.T) {
		mockEmploymentRepo.EXPECT().GetEmploymentByAccountID(gomock.Any(), gomock.Eq(testAccountID)).Return(nil, gorm.ErrRecordNotFound).Times(1)
		employment, err := service.GetEmploymentByAccountID(ctx, testAccountID) // 使用頂層 service
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmploymentNotFound)
		assert.Nil(t, employment)
	})

	t.Run("Failure - DB Error", func(t *testing.T) {
		dbError := errors.New("some database error")
		mockEmploymentRepo.EXPECT().GetEmploymentByAccountID(gomock.Any(), gomock.Eq(testAccountID)).Return(nil, dbError).Times(1)
		employment, err := service.GetEmploymentByAccountID(ctx, testAccountID) // 使用頂層 service
		require.Error(t, err)
		assert.NotErrorIs(t, err, ErrEmploymentNotFound)
		assert.Contains(t, err.Error(), "database error fetching employment record")
		assert.Nil(t, employment)
	})
}

func TestEmploymentServiceImpl_GetEmploymentByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockEmploymentRepo := mocks.NewMockEmploymentRepository(ctrl)
	mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
	// *** service 在函數頂層宣告並在子測試中使用 ***
	service := NewEmploymentServiceImpl(mockEmploymentRepo, mockAccountRepo)

	ctx := context.Background()
	testEmploymentID := uuid.New()
	testAccountID := uuid.New()
	mockEmployment := &models.Employment{ID: testEmploymentID, AccountID: testAccountID, Status: models.EmploymentStatusActive}

	t.Run("Success", func(t *testing.T) {
		mockEmploymentRepo.EXPECT().GetEmploymentByID(gomock.Any(), gomock.Eq(testEmploymentID)).Return(mockEmployment, nil).Times(1)
		employment, err := service.GetEmploymentByID(ctx, testEmploymentID) // 使用頂層 service
		require.NoError(t, err)
		require.NotNil(t, employment)
		assert.Equal(t, testEmploymentID, employment.ID)
	})

	t.Run("Failure - Not Found", func(t *testing.T) {
		mockEmploymentRepo.EXPECT().GetEmploymentByID(gomock.Any(), gomock.Eq(testEmploymentID)).Return(nil, gorm.ErrRecordNotFound).Times(1)
		employment, err := service.GetEmploymentByID(ctx, testEmploymentID) // 使用頂層 service
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmploymentNotFound)
		assert.Nil(t, employment)
	})

	t.Run("Failure - DB Error", func(t *testing.T) {
		dbError := errors.New("db error get by id")
		mockEmploymentRepo.EXPECT().GetEmploymentByID(gomock.Any(), gomock.Eq(testEmploymentID)).Return(nil, dbError).Times(1)
		employment, err := service.GetEmploymentByID(ctx, testEmploymentID) // 使用頂層 service
		require.Error(t, err)
		assert.NotErrorIs(t, err, ErrEmploymentNotFound)
		assert.Contains(t, err.Error(), "database error fetching employment record")
		assert.Nil(t, employment)
	})
}

func TestEmploymentServiceImpl_UpdateEmploymentDetails(t *testing.T) {

	ctx := context.Background()
	employmentID := uuid.New()
	accountID := uuid.New()
	originalHireDate := time.Now().AddDate(-1, 0, 0)
	newJobGradeID := uuid.New()
	newSalary := decimal.NewFromFloat(60000.50)

	existingEmp := &models.Employment{
		ID:            employmentID,
		AccountID:     accountID,
		Status:        models.EmploymentStatusActive,
		PositionTitle: "Junior Developer",
		JobGradeID:    nil,
		Salary:        Ptr(decimal.NewFromFloat(50000.00)),
		HireDate:      Ptr(originalHireDate),
	}

	updates := &models.Employment{
		JobGradeID:    &newJobGradeID,
		PositionTitle: "Mid-Level Developer",
		Salary:        &newSalary,
	}

	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish() // 在子測試內宣告
		mockEmploymentRepo := mocks.NewMockEmploymentRepository(ctrl)
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)                  // 雖然未使用，但 New 需要
		service := NewEmploymentServiceImpl(mockEmploymentRepo, mockAccountRepo) // 在子測試內宣告
		localExistingEmp := *existingEmp                                         // Create copy

		mockEmploymentRepo.EXPECT().GetEmploymentByID(gomock.Any(), gomock.Eq(employmentID)).Return(&localExistingEmp, nil).Times(1)
		mockEmploymentRepo.EXPECT().UpdateEmployment(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, emp *models.Employment) error {
				assert.Equal(t, employmentID, emp.ID)
				assert.Equal(t, accountID, emp.AccountID)
				assert.Equal(t, updates.JobGradeID, emp.JobGradeID)
				assert.Equal(t, updates.PositionTitle, emp.PositionTitle)
				require.NotNil(t, emp.Salary)
				require.NotNil(t, updates.Salary)
				assert.True(t, updates.Salary.Equals(*emp.Salary))
				assert.Equal(t, existingEmp.Status, emp.Status)
				assert.Equal(t, existingEmp.HireDate, emp.HireDate)
				return nil
			}).Times(1)

		updatedEmp, err := service.UpdateEmploymentDetails(ctx, employmentID, updates)

		require.NoError(t, err)
		require.NotNil(t, updatedEmp)
		assert.Equal(t, employmentID, updatedEmp.ID)
		assert.Equal(t, updates.JobGradeID, updatedEmp.JobGradeID)
		assert.Equal(t, updates.PositionTitle, updatedEmp.PositionTitle)
		require.NotNil(t, updatedEmp.Salary)
		assert.True(t, updates.Salary.Equals(*updatedEmp.Salary))
	})

	t.Run("Success - No Changes", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish() // 在子測試內宣告
		mockEmploymentRepo := mocks.NewMockEmploymentRepository(ctrl)
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		service := NewEmploymentServiceImpl(mockEmploymentRepo, mockAccountRepo) // 在子測試內宣告
		localExistingEmp := *existingEmp
		noChangeUpdates := &models.Employment{
			JobGradeID:    localExistingEmp.JobGradeID,
			PositionTitle: localExistingEmp.PositionTitle,
			Salary:        localExistingEmp.Salary,
		}

		mockEmploymentRepo.EXPECT().GetEmploymentByID(gomock.Any(), gomock.Eq(employmentID)).Return(&localExistingEmp, nil).Times(1)
		// UpdateEmployment 不應被調用

		updatedEmp, err := service.UpdateEmploymentDetails(ctx, employmentID, noChangeUpdates)

		require.NoError(t, err)
		require.NotNil(t, updatedEmp)
		assert.Equal(t, existingEmp.ID, updatedEmp.ID)
	})

	t.Run("Failure - Not Found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish() // 在子測試內宣告
		mockEmploymentRepo := mocks.NewMockEmploymentRepository(ctrl)
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		service := NewEmploymentServiceImpl(mockEmploymentRepo, mockAccountRepo) // 在子測試內宣告

		mockEmploymentRepo.EXPECT().GetEmploymentByID(gomock.Any(), gomock.Eq(employmentID)).Return(nil, gorm.ErrRecordNotFound).Times(1)

		updatedEmp, err := service.UpdateEmploymentDetails(ctx, employmentID, updates)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmploymentNotFound)
		assert.Nil(t, updatedEmp)
	})

	t.Run("Failure - Already Terminated", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish() // 在子測試內宣告
		mockEmploymentRepo := mocks.NewMockEmploymentRepository(ctrl)
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		service := NewEmploymentServiceImpl(mockEmploymentRepo, mockAccountRepo) // 在子測試內宣告
		terminatedEmp := *existingEmp
		terminatedEmp.Status = models.EmploymentStatusTerminated

		mockEmploymentRepo.EXPECT().GetEmploymentByID(gomock.Any(), gomock.Eq(employmentID)).Return(&terminatedEmp, nil).Times(1)
		// UpdateEmployment 不應被調用

		updatedEmp, err := service.UpdateEmploymentDetails(ctx, employmentID, updates)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot update terminated employment record")
		assert.Nil(t, updatedEmp)
	})

	t.Run("Failure - Update Repo Error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish() // 在子測試內宣告
		mockEmploymentRepo := mocks.NewMockEmploymentRepository(ctrl)
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		service := NewEmploymentServiceImpl(mockEmploymentRepo, mockAccountRepo) // 在子測試內宣告
		localExistingEmp := *existingEmp
		updateError := errors.New("repo update failed")

		mockEmploymentRepo.EXPECT().GetEmploymentByID(gomock.Any(), gomock.Eq(employmentID)).Return(&localExistingEmp, nil).Times(1)
		mockEmploymentRepo.EXPECT().UpdateEmployment(gomock.Any(), gomock.Any()).Return(updateError).Times(1)

		updatedEmp, err := service.UpdateEmploymentDetails(ctx, employmentID, updates)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrUpdateFailed)
		assert.Nil(t, updatedEmp)
	})
}

func TestEmploymentServiceImpl_TerminateEmployment(t *testing.T) {

	ctx := context.Background()
	employmentID := uuid.New()
	accountID := uuid.New()
	terminationDate := time.Now().Truncate(24 * time.Hour)

	activeEmp := &models.Employment{
		ID:        employmentID,
		AccountID: accountID,
		Status:    models.EmploymentStatusActive,
	}

	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish() // 在子測試內宣告
		mockEmploymentRepo := mocks.NewMockEmploymentRepository(ctrl)
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		service := NewEmploymentServiceImpl(mockEmploymentRepo, mockAccountRepo) // 在子測試內宣告
		localActiveEmp := *activeEmp

		mockEmploymentRepo.EXPECT().GetEmploymentByID(gomock.Any(), gomock.Eq(employmentID)).Return(&localActiveEmp, nil).Times(1)
		mockEmploymentRepo.EXPECT().UpdateEmployment(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, emp *models.Employment) error {
				assert.Equal(t, models.EmploymentStatusTerminated, emp.Status)
				require.NotNil(t, emp.TerminationDate)
				assert.Equal(t, terminationDate, emp.TerminationDate.Truncate(24*time.Hour))
				return nil
			}).Times(1)

		err := service.TerminateEmployment(ctx, employmentID, terminationDate)

		require.NoError(t, err)
	})

	t.Run("Failure - Not Found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish() // 在子測試內宣告
		mockEmploymentRepo := mocks.NewMockEmploymentRepository(ctrl)
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		service := NewEmploymentServiceImpl(mockEmploymentRepo, mockAccountRepo) // 在子測試內宣告

		mockEmploymentRepo.EXPECT().GetEmploymentByID(gomock.Any(), gomock.Eq(employmentID)).Return(nil, gorm.ErrRecordNotFound).Times(1)

		err := service.TerminateEmployment(ctx, employmentID, terminationDate)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmploymentNotFound)
	})

	t.Run("Failure - Already Terminated", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish() // 在子測試內宣告
		mockEmploymentRepo := mocks.NewMockEmploymentRepository(ctrl)
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		service := NewEmploymentServiceImpl(mockEmploymentRepo, mockAccountRepo) // 在子測試內宣告
		terminatedEmp := *activeEmp
		terminatedEmp.Status = models.EmploymentStatusTerminated

		mockEmploymentRepo.EXPECT().GetEmploymentByID(gomock.Any(), gomock.Eq(employmentID)).Return(&terminatedEmp, nil).Times(1)

		err := service.TerminateEmployment(ctx, employmentID, terminationDate)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAlreadyTerminated)
	})

	t.Run("Failure - Update Repo Error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish() // 在子測試內宣告
		mockEmploymentRepo := mocks.NewMockEmploymentRepository(ctrl)
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		service := NewEmploymentServiceImpl(mockEmploymentRepo, mockAccountRepo) // 在子測試內宣告
		localActiveEmp := *activeEmp
		updateError := errors.New("repo terminate failed")

		mockEmploymentRepo.EXPECT().GetEmploymentByID(gomock.Any(), gomock.Eq(employmentID)).Return(&localActiveEmp, nil).Times(1)
		mockEmploymentRepo.EXPECT().UpdateEmployment(gomock.Any(), gomock.Any()).Return(updateError).Times(1)

		err := service.TerminateEmployment(ctx, employmentID, terminationDate)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrTerminationFailed)
	})
}

func TestEmploymentServiceImpl_ListEmployments(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockEmploymentRepo := mocks.NewMockEmploymentRepository(ctrl)
	mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
	// *** service 在函數頂層宣告並在子測試中使用 ***
	service := NewEmploymentServiceImpl(mockEmploymentRepo, mockAccountRepo)

	ctx := context.Background()
	mockEmployments := []models.Employment{
		{ID: uuid.New(), AccountID: uuid.New(), Status: models.EmploymentStatusActive},
		{ID: uuid.New(), AccountID: uuid.New(), Status: models.EmploymentStatusTerminated},
	}

	t.Run("Success", func(t *testing.T) {
		mockEmploymentRepo.EXPECT().ListEmployments(gomock.Any()).Return(mockEmployments, nil).Times(1)
		employments, err := service.ListEmployments(ctx) // 使用頂層 service
		require.NoError(t, err)
		assert.Len(t, employments, 2)
		assert.Equal(t, mockEmployments[0].ID, employments[0].ID)
	})

	t.Run("Success - Empty List", func(t *testing.T) {
		mockEmploymentRepo.EXPECT().ListEmployments(gomock.Any()).Return([]models.Employment{}, nil).Times(1)
		employments, err := service.ListEmployments(ctx) // 使用頂層 service
		require.NoError(t, err)
		assert.Len(t, employments, 0)
		assert.Empty(t, employments)
	})

	t.Run("Failure - DB Error", func(t *testing.T) {
		dbError := errors.New("db list error")
		mockEmploymentRepo.EXPECT().ListEmployments(gomock.Any()).Return(nil, dbError).Times(1)
		employments, err := service.ListEmployments(ctx) // 使用頂層 service
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list employment records")
		assert.Nil(t, employments)
	})
}
