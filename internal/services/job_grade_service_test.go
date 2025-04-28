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

	gomock "github.com/golang/mock/gomock" 
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// --- Test CreateJobGrade ---
func TestJobGradeServiceImpl_CreateJobGrade(t *testing.T) {
	ctx := context.Background()
	mockInputGrade := &models.JobGrade{
		Code: "TDD-P1", Name: "Test Engineer 1", Description: "Desc", MinSalary: decimal.NewFromInt(100), MaxSalary: decimal.NewFromInt(200),
	}

	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockJobGradeRepo := mocks.NewMockJobGradeRepository(ctrl)
		service := NewJobGradeServiceImpl(mockJobGradeRepo, nil) // Delete doesn't need employmentRepo here
		localInput := *mockInputGrade                            // Use copy

		// 1. Expect code check -> Not Found
		mockJobGradeRepo.EXPECT().GetJobGradeByCode(gomock.Any(), gomock.Eq(localInput.Code)).Return(nil, gorm.ErrRecordNotFound).Times(1)
		// 2. Expect create call -> Success
		mockJobGradeRepo.EXPECT().CreateJobGrade(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, grade *models.JobGrade) error {
				assert.Equal(t, localInput.Code, grade.Code) // Check passed data
				grade.ID = uuid.New()                        // Simulate hook/DB setting ID
				grade.CreatedAt = time.Now()
				grade.UpdatedAt = time.Now()
				return nil // Simulate success
			}).Times(1)

		createdGrade, err := service.CreateJobGrade(ctx, &localInput)

		require.NoError(t, err)
		require.NotNil(t, createdGrade)
		assert.NotEqual(t, uuid.Nil, createdGrade.ID)
		assert.Equal(t, localInput.Code, createdGrade.Code)
	})

	t.Run("Failure - Invalid Input (Empty Code)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockJobGradeRepo := mocks.NewMockJobGradeRepository(ctrl)
		service := NewJobGradeServiceImpl(mockJobGradeRepo, nil)
		invalidInput := &models.JobGrade{Code: " ", Name: "Valid Name"} // Empty Code

		createdGrade, err := service.CreateJobGrade(ctx, invalidInput)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidInput)
		assert.Nil(t, createdGrade)
	})

	t.Run("Failure - Code Exists", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockJobGradeRepo := mocks.NewMockJobGradeRepository(ctrl)
		service := NewJobGradeServiceImpl(mockJobGradeRepo, nil)
		localInput := *mockInputGrade
		existingGrade := &models.JobGrade{ID: uuid.New(), Code: localInput.Code}

		// Expect code check -> Found
		mockJobGradeRepo.EXPECT().GetJobGradeByCode(gomock.Any(), gomock.Eq(localInput.Code)).Return(existingGrade, nil).Times(1)
		// Create should not be called

		createdGrade, err := service.CreateJobGrade(ctx, &localInput)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrJobGradeCodeExists)
		assert.Contains(t, err.Error(), localInput.Code)
		assert.Nil(t, createdGrade)
	})

	t.Run("Failure - Code Check DB Error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockJobGradeRepo := mocks.NewMockJobGradeRepository(ctrl)
		service := NewJobGradeServiceImpl(mockJobGradeRepo, nil)
		localInputGrade := *mockInputGrade
		dbError := errors.New("db connection failed")

		mockJobGradeRepo.EXPECT().GetJobGradeByCode(gomock.Any(), gomock.Eq(localInputGrade.Code)).Return(nil, dbError).Times(1)

		createdGrade, err := service.CreateJobGrade(ctx, &localInputGrade)

		require.Error(t, err)
		assert.ErrorIs(t, err, dbError) // Service wraps this
		assert.Contains(t, err.Error(), "database error checking job grade code")
		assert.Nil(t, createdGrade)
	})

	t.Run("Failure - Create Repo Error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockJobGradeRepo := mocks.NewMockJobGradeRepository(ctrl)
		service := NewJobGradeServiceImpl(mockJobGradeRepo, nil)
		localInputGrade := *mockInputGrade
		repoError := errors.New("insert unique constraint failed")

		mockJobGradeRepo.EXPECT().GetJobGradeByCode(gomock.Any(), gomock.Eq(localInputGrade.Code)).Return(nil, gorm.ErrRecordNotFound).Times(1)
		mockJobGradeRepo.EXPECT().CreateJobGrade(gomock.Any(), gomock.Any()).Return(repoError).Times(1)

		createdGrade, err := service.CreateJobGrade(ctx, &localInputGrade)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrJobGradeCreateFailed)
		assert.ErrorIs(t, err, repoError) // Check underlying error
		assert.Nil(t, createdGrade)
	})
}

// --- Test GetJobGradeByID ---
func TestJobGradeServiceImpl_GetJobGradeByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockJobGradeRepo := mocks.NewMockJobGradeRepository(ctrl)
	service := NewJobGradeServiceImpl(mockJobGradeRepo, nil)
	ctx := context.Background()
	testID := uuid.New()
	mockGrade := &models.JobGrade{ID: testID, Code: "TID", Name: "TestByID"}

	t.Run("Success", func(t *testing.T) {
		mockJobGradeRepo.EXPECT().GetJobGradeByID(gomock.Any(), gomock.Eq(testID)).Return(mockGrade, nil).Times(1)
		grade, err := service.GetJobGradeByID(ctx, testID)
		require.NoError(t, err)
		require.NotNil(t, grade)
		assert.Equal(t, testID, grade.ID)
	})
	t.Run("Failure - Not Found", func(t *testing.T) {
		mockJobGradeRepo.EXPECT().GetJobGradeByID(gomock.Any(), gomock.Eq(testID)).Return(nil, gorm.ErrRecordNotFound).Times(1)
		grade, err := service.GetJobGradeByID(ctx, testID)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrJobGradeNotFound)
		assert.Nil(t, grade)
	})
	t.Run("Failure - DB Error", func(t *testing.T) {
		dbError := errors.New("get by id db error")
		mockJobGradeRepo.EXPECT().GetJobGradeByID(gomock.Any(), gomock.Eq(testID)).Return(nil, dbError).Times(1)
		grade, err := service.GetJobGradeByID(ctx, testID)
		require.Error(t, err)
		assert.ErrorIs(t, err, dbError)
		assert.Nil(t, grade)
		assert.Contains(t, err.Error(), "database error fetching job grade by ID")
	})
}

// --- Test GetJobGradeByCode ---
func TestJobGradeServiceImpl_GetJobGradeByCode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockJobGradeRepo := mocks.NewMockJobGradeRepository(ctrl)
	service := NewJobGradeServiceImpl(mockJobGradeRepo, nil)
	ctx := context.Background()
	testCode := "TCODE"
	mockGrade := &models.JobGrade{ID: uuid.New(), Code: testCode, Name: "TestByCode"}

	t.Run("Success", func(t *testing.T) {
		mockJobGradeRepo.EXPECT().GetJobGradeByCode(gomock.Any(), gomock.Eq(testCode)).Return(mockGrade, nil).Times(1)
		grade, err := service.GetJobGradeByCode(ctx, testCode)
		require.NoError(t, err)
		require.NotNil(t, grade)
		assert.Equal(t, testCode, grade.Code)
	})
	t.Run("Failure - Not Found", func(t *testing.T) {
		mockJobGradeRepo.EXPECT().GetJobGradeByCode(gomock.Any(), gomock.Eq(testCode)).Return(nil, gorm.ErrRecordNotFound).Times(1)
		grade, err := service.GetJobGradeByCode(ctx, testCode)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrJobGradeNotFound)
		assert.Nil(t, grade)
	})
	t.Run("Failure - DB Error", func(t *testing.T) {
		dbError := errors.New("get by code db error")
		mockJobGradeRepo.EXPECT().GetJobGradeByCode(gomock.Any(), gomock.Eq(testCode)).Return(nil, dbError).Times(1)
		grade, err := service.GetJobGradeByCode(ctx, testCode)
		require.Error(t, err)
		assert.ErrorIs(t, err, dbError)
		assert.Nil(t, grade)
		assert.Contains(t, err.Error(), "database error fetching job grade by code")
	})
	t.Run("Failure - Empty Code Input", func(t *testing.T) {
		grade, err := service.GetJobGradeByCode(ctx, " ")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidInput)
		assert.Nil(t, grade)
	})
}

// --- Test UpdateJobGrade ---
func TestJobGradeServiceImpl_UpdateJobGrade(t *testing.T) {
	ctx := context.Background()
	testID := uuid.New()
	existingCode := "P3"
	existingName := "Senior Engineer"
	newCode := "P4"
	newName := "Lead Engineer"
	existingGrade := &models.JobGrade{ID: testID, Code: existingCode, Name: existingName}
	updatesOnlyName := &models.JobGrade{Name: newName}
	updatesCodeAndName := &models.JobGrade{Code: newCode, Name: newName}

	t.Run("Success - Update Name Only", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockJobGradeRepo := mocks.NewMockJobGradeRepository(ctrl)
		service := NewJobGradeServiceImpl(mockJobGradeRepo, nil)
		localExistingGrade := *existingGrade

		mockJobGradeRepo.EXPECT().GetJobGradeByID(gomock.Any(), gomock.Eq(testID)).Return(&localExistingGrade, nil).Times(1)
		// Code check should not be called
		mockJobGradeRepo.EXPECT().UpdateJobGrade(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, grade *models.JobGrade) error {
				assert.Equal(t, newName, grade.Name)
				assert.Equal(t, existingCode, grade.Code) // Verify code didn't change
				return nil
			}).Times(1)

		updatedGrade, err := service.UpdateJobGrade(ctx, testID, updatesOnlyName)
		require.NoError(t, err)
		require.NotNil(t, updatedGrade)
		assert.Equal(t, newName, updatedGrade.Name)
	})

	t.Run("Success - Update Code and Name", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockJobGradeRepo := mocks.NewMockJobGradeRepository(ctrl)
		service := NewJobGradeServiceImpl(mockJobGradeRepo, nil)
		localExistingGrade := *existingGrade

		mockJobGradeRepo.EXPECT().GetJobGradeByID(gomock.Any(), gomock.Eq(testID)).Return(&localExistingGrade, nil).Times(1)
		mockJobGradeRepo.EXPECT().GetJobGradeByCode(gomock.Any(), gomock.Eq(newCode)).Return(nil, gorm.ErrRecordNotFound).Times(1) // New code is available
		mockJobGradeRepo.EXPECT().UpdateJobGrade(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, grade *models.JobGrade) error {
				assert.Equal(t, newCode, grade.Code)
				assert.Equal(t, newName, grade.Name)
				return nil
			}).Times(1)

		updatedGrade, err := service.UpdateJobGrade(ctx, testID, updatesCodeAndName)
		require.NoError(t, err)
		require.NotNil(t, updatedGrade)
		assert.Equal(t, newCode, updatedGrade.Code)
		assert.Equal(t, newName, updatedGrade.Name)
	})

}

// --- Test ListJobGrades ---
func TestJobGradeServiceImpl_ListJobGrades(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockJobGradeRepo := mocks.NewMockJobGradeRepository(ctrl)
	service := NewJobGradeServiceImpl(mockJobGradeRepo, nil)
	ctx := context.Background()
	mockGrades := []models.JobGrade{{ID: uuid.New(), Code: "P1"}, {ID: uuid.New(), Code: "P2"}}

	t.Run("Success", func(t *testing.T) {
		mockJobGradeRepo.EXPECT().ListJobGrades(gomock.Any()).Return(mockGrades, nil).Times(1)
		grades, err := service.ListJobGrades(ctx)
		require.NoError(t, err)
		assert.Len(t, grades, 2)
	})
	t.Run("Success - Empty", func(t *testing.T) {
		mockJobGradeRepo.EXPECT().ListJobGrades(gomock.Any()).Return([]models.JobGrade{}, nil).Times(1)
		grades, err := service.ListJobGrades(ctx)
		require.NoError(t, err)
		assert.Empty(t, grades)
	})
	t.Run("Failure - DB Error", func(t *testing.T) {
		dbError := errors.New("list db error")
		mockJobGradeRepo.EXPECT().ListJobGrades(gomock.Any()).Return(nil, dbError).Times(1)
		grades, err := service.ListJobGrades(ctx)
		require.Error(t, err)
		assert.ErrorIs(t, err, dbError)
		assert.Nil(t, grades)
		assert.Contains(t, err.Error(), "failed to list job grades")
	})
}

// --- Test DeleteJobGrade ---
func TestJobGradeServiceImpl_DeleteJobGrade(t *testing.T) {
	ctx := context.Background()
	testID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockJobGradeRepo := mocks.NewMockJobGradeRepository(ctrl)
		mockEmploymentRepo := mocks.NewMockEmploymentRepository(ctrl)
		service := NewJobGradeServiceImpl(mockJobGradeRepo, mockEmploymentRepo)

		// 1. Expect check for employment usage -> returns 0
		mockEmploymentRepo.EXPECT().GetEmploymentCountByJobGradeID(gomock.Any(), gomock.Eq(testID)).Return(int64(0), nil).Times(1)
		// 2. Expect delete call -> returns success
		mockJobGradeRepo.EXPECT().DeleteJobGrade(gomock.Any(), gomock.Eq(testID)).Return(nil).Times(1)

		err := service.DeleteJobGrade(ctx, testID)
		require.NoError(t, err)
	})

	t.Run("Failure - Job Grade In Use", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockJobGradeRepo := mocks.NewMockJobGradeRepository(ctrl)
		mockEmploymentRepo := mocks.NewMockEmploymentRepository(ctrl)
		service := NewJobGradeServiceImpl(mockJobGradeRepo, mockEmploymentRepo)

		// Expect check for employment usage -> returns count > 0
		mockEmploymentRepo.EXPECT().GetEmploymentCountByJobGradeID(gomock.Any(), gomock.Eq(testID)).Return(int64(1), nil).Times(1)
		// DeleteJobGrade should NOT be called

		err := service.DeleteJobGrade(ctx, testID)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrJobGradeInUse)
	})

	t.Run("Failure - Employment Check Error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockJobGradeRepo := mocks.NewMockJobGradeRepository(ctrl)
		mockEmploymentRepo := mocks.NewMockEmploymentRepository(ctrl)
		service := NewJobGradeServiceImpl(mockJobGradeRepo, mockEmploymentRepo)
		checkError := errors.New("db error checking usage")

		mockEmploymentRepo.EXPECT().GetEmploymentCountByJobGradeID(gomock.Any(), gomock.Eq(testID)).Return(int64(0), checkError).Times(1)
		// DeleteJobGrade should NOT be called

		err := service.DeleteJobGrade(ctx, testID)
		require.Error(t, err)
		assert.ErrorIs(t, err, checkError)
		assert.ErrorIs(t, err, ErrEmploymentCheckFailed)
	})

	t.Run("Failure - Delete Not Found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockJobGradeRepo := mocks.NewMockJobGradeRepository(ctrl)
		mockEmploymentRepo := mocks.NewMockEmploymentRepository(ctrl)
		service := NewJobGradeServiceImpl(mockJobGradeRepo, mockEmploymentRepo)

		mockEmploymentRepo.EXPECT().GetEmploymentCountByJobGradeID(gomock.Any(), gomock.Eq(testID)).Return(int64(0), nil).Times(1)
		// Expect delete call -> returns NotFound
		mockJobGradeRepo.EXPECT().DeleteJobGrade(gomock.Any(), gomock.Eq(testID)).Return(gorm.ErrRecordNotFound).Times(1)

		err := service.DeleteJobGrade(ctx, testID)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrJobGradeNotFound)
	})

	t.Run("Failure - Delete DB Error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockJobGradeRepo := mocks.NewMockJobGradeRepository(ctrl)
		mockEmploymentRepo := mocks.NewMockEmploymentRepository(ctrl)
		service := NewJobGradeServiceImpl(mockJobGradeRepo, mockEmploymentRepo)
		deleteError := errors.New("db delete constraint error")

		mockEmploymentRepo.EXPECT().GetEmploymentCountByJobGradeID(gomock.Any(), gomock.Eq(testID)).Return(int64(0), nil).Times(1)
		mockJobGradeRepo.EXPECT().DeleteJobGrade(gomock.Any(), gomock.Eq(testID)).Return(deleteError).Times(1)

		err := service.DeleteJobGrade(ctx, testID)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrJobGradeDeleteFailed)
		assert.ErrorIs(t, err, deleteError)
	})
}
