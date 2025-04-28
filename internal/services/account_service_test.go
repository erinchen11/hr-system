package services

import (
	"context"
	"errors" 
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock" 
	"github.com/erinchen11/hr-system/internal/interfaces/mocks" 
	"github.com/erinchen11/hr-system/internal/models"
	"github.com/google/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gomock "github.com/golang/mock/gomock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Helper function Ptr (保持不變)
func Ptr[T any](v T) *T { return &v }

// Helper to setup GORM with sqlmock (保持使用 Equal Matcher)
func setupGormWithSqlmock(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	mockDb, mockSql, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual)) // Back to Equal based on last attempt
	require.NoError(t, err)

	dialector := mysql.New(mysql.Config{
		Conn:                      mockDb,
		SkipInitializeWithVersion: true,
	})
	gormDb, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // Default to Silent
		// Logger: newLogger, // Uncomment to enable logging
	})
	require.NoError(t, err)
	return gormDb, mockSql
}

// --- Tests using Gomock (舊路徑) ---

func TestAccountServiceImpl_Authenticate_WithGoMock(t *testing.T) {
	ctx := context.Background()
	testEmail := "test@example.com"
	testPassword := "password123"
	hashedPassword := "hashed_password_from_db"
	baseAccountData := func() *models.Account {
		return &models.Account{ID: uuid.New(), FirstName: "Test", LastName: "User", Email: testEmail, Password: hashedPassword, Role: models.RoleEmployee, CreatedAt: time.Now().Add(-time.Hour), UpdatedAt: time.Now()}
	}

	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		mockPwChecker := mocks.NewMockPasswordChecker(ctrl)
		service := NewAccountServiceImpl(mockAccountRepo, nil, mockPwChecker, nil, nil, "", nil)
		localAccountData := baseAccountData()
		mockAccountRepo.EXPECT().GetAccountByEmail(gomock.Any(), gomock.Eq(testEmail)).Return(localAccountData, nil).Times(1)
		mockPwChecker.EXPECT().CheckPassword(gomock.Eq(localAccountData.Password), gomock.Eq(testPassword)).Return(true).Times(1)
		authenticatedAccount, err := service.Authenticate(ctx, testEmail, testPassword)
		require.NoError(t, err)
		require.NotNil(t, authenticatedAccount)
		assert.Equal(t, localAccountData.ID, authenticatedAccount.ID)
		assert.Equal(t, "", authenticatedAccount.Password)
	})
	t.Run("Failure - Invalid Email (Not Found)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		service := NewAccountServiceImpl(mockAccountRepo, nil, nil, nil, nil, "", nil)
		mockAccountRepo.EXPECT().GetAccountByEmail(gomock.Any(), gomock.Eq(testEmail)).Return(nil, gorm.ErrRecordNotFound).Times(1)
		authenticatedAccount, err := service.Authenticate(ctx, testEmail, testPassword)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidCredentials)
		assert.Nil(t, authenticatedAccount)
	})
	t.Run("Failure - Invalid Password", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		mockPwChecker := mocks.NewMockPasswordChecker(ctrl)
		service := NewAccountServiceImpl(mockAccountRepo, nil, mockPwChecker, nil, nil, "", nil)
		localAccountData := baseAccountData()
		mockAccountRepo.EXPECT().GetAccountByEmail(gomock.Any(), gomock.Eq(testEmail)).Return(localAccountData, nil).Times(1)
		mockPwChecker.EXPECT().CheckPassword(gomock.Eq(localAccountData.Password), gomock.Eq(testPassword)).Return(false).Times(1)
		authenticatedAccount, err := service.Authenticate(ctx, testEmail, testPassword)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidCredentials)
		assert.Nil(t, authenticatedAccount)
	})
	t.Run("Failure - Database Error on Fetch", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		service := NewAccountServiceImpl(mockAccountRepo, nil, nil, nil, nil, "", nil)
		dbError := errors.New("unexpected database connection error")
		mockAccountRepo.EXPECT().GetAccountByEmail(gomock.Any(), gomock.Eq(testEmail)).Return(nil, dbError).Times(1)
		authenticatedAccount, err := service.Authenticate(ctx, testEmail, testPassword)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidCredentials)
		assert.Nil(t, authenticatedAccount)
	})
}

func TestAccountServiceImpl_ChangePassword_WithGoMock(t *testing.T) {
	ctx := context.Background()
	accountID := uuid.New()
	oldPassword := "oldPassword123"
	newPassword := "newPassword456"
	hashedOldPassword := "hashed_old_password"
	hashedNewPassword := "hashed_new_password"
	mockAccountData := func() *models.Account {
		return &models.Account{ID: accountID, Password: hashedOldPassword}
	}

	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		mockPwChecker := mocks.NewMockPasswordChecker(ctrl)
		mockPwHasher := mocks.NewMockPasswordHasher(ctrl)
		mockCacheRepo := mocks.NewMockCacheRepository(ctrl)
		service := NewAccountServiceImpl(mockAccountRepo, nil, mockPwChecker, mockPwHasher, mockCacheRepo, "", nil)
		localAccountData := mockAccountData()
		mockAccountRepo.EXPECT().GetAccountByID(gomock.Any(), gomock.Eq(accountID)).Return(localAccountData, nil).Times(1)
		mockPwChecker.EXPECT().CheckPassword(gomock.Eq(localAccountData.Password), gomock.Eq(oldPassword)).Return(true).Times(1)
		mockPwHasher.EXPECT().HashPassword(gomock.Eq(newPassword)).Return(hashedNewPassword, nil).Times(1)
		mockAccountRepo.EXPECT().UpdatePassword(gomock.Any(), gomock.Eq(accountID), gomock.Eq(hashedNewPassword)).Return(nil).Times(1)
		err := service.ChangePassword(ctx, accountID, oldPassword, newPassword)
		require.NoError(t, err)
	})
	t.Run("Failure - Account Not Found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		service := NewAccountServiceImpl(mockAccountRepo, nil, nil, nil, nil, "", nil)
		mockAccountRepo.EXPECT().GetAccountByID(gomock.Any(), gomock.Eq(accountID)).Return(nil, gorm.ErrRecordNotFound).Times(1)
		err := service.ChangePassword(ctx, accountID, oldPassword, newPassword)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAccountNotFound)
	})
	t.Run("Failure - Invalid Old Password", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		mockPwChecker := mocks.NewMockPasswordChecker(ctrl)
		service := NewAccountServiceImpl(mockAccountRepo, nil, mockPwChecker, nil, nil, "", nil)
		localAccountData := mockAccountData()
		mockAccountRepo.EXPECT().GetAccountByID(gomock.Any(), gomock.Eq(accountID)).Return(localAccountData, nil).Times(1)
		mockPwChecker.EXPECT().CheckPassword(gomock.Eq(localAccountData.Password), gomock.Eq(oldPassword)).Return(false).Times(1)
		err := service.ChangePassword(ctx, accountID, oldPassword, newPassword)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidCredentials)
	})
	t.Run("Failure - Password Hashing Error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		mockPwChecker := mocks.NewMockPasswordChecker(ctrl)
		mockPwHasher := mocks.NewMockPasswordHasher(ctrl)
		service := NewAccountServiceImpl(mockAccountRepo, nil, mockPwChecker, mockPwHasher, nil, "", nil)
		hashError := errors.New("bcrypt failed")
		localAccountData := mockAccountData()
		mockAccountRepo.EXPECT().GetAccountByID(gomock.Any(), gomock.Eq(accountID)).Return(localAccountData, nil).Times(1)
		mockPwChecker.EXPECT().CheckPassword(gomock.Eq(localAccountData.Password), gomock.Eq(oldPassword)).Return(true).Times(1)
		mockPwHasher.EXPECT().HashPassword(gomock.Eq(newPassword)).Return("", hashError).Times(1)
		err := service.ChangePassword(ctx, accountID, oldPassword, newPassword)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrPasswordHashingFailed)
	})
	t.Run("Failure - Update Password DB Error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		mockPwChecker := mocks.NewMockPasswordChecker(ctrl)
		mockPwHasher := mocks.NewMockPasswordHasher(ctrl)
		service := NewAccountServiceImpl(mockAccountRepo, nil, mockPwChecker, mockPwHasher, nil, "", nil)
		dbError := errors.New("connection failed")
		localAccountData := mockAccountData()
		mockAccountRepo.EXPECT().GetAccountByID(gomock.Any(), gomock.Eq(accountID)).Return(localAccountData, nil).Times(1)
		mockPwChecker.EXPECT().CheckPassword(gomock.Eq(localAccountData.Password), gomock.Eq(oldPassword)).Return(true).Times(1)
		mockPwHasher.EXPECT().HashPassword(gomock.Eq(newPassword)).Return(hashedNewPassword, nil).Times(1)
		mockAccountRepo.EXPECT().UpdatePassword(gomock.Any(), gomock.Eq(accountID), gomock.Eq(hashedNewPassword)).Return(dbError).Times(1)
		err := service.ChangePassword(ctx, accountID, oldPassword, newPassword)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrPasswordUpdateFailed)
	})
	t.Run("Failure - Update Password Not Found Error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		mockPwChecker := mocks.NewMockPasswordChecker(ctrl)
		mockPwHasher := mocks.NewMockPasswordHasher(ctrl)
		service := NewAccountServiceImpl(mockAccountRepo, nil, mockPwChecker, mockPwHasher, nil, "", nil)
		localAccountData := mockAccountData()
		mockAccountRepo.EXPECT().GetAccountByID(gomock.Any(), gomock.Eq(accountID)).Return(localAccountData, nil).Times(1)
		mockPwChecker.EXPECT().CheckPassword(gomock.Eq(localAccountData.Password), gomock.Eq(oldPassword)).Return(true).Times(1)
		mockPwHasher.EXPECT().HashPassword(gomock.Eq(newPassword)).Return(hashedNewPassword, nil).Times(1)
		mockAccountRepo.EXPECT().UpdatePassword(gomock.Any(), gomock.Eq(accountID), gomock.Eq(hashedNewPassword)).Return(gorm.ErrRecordNotFound).Times(1)
		err := service.ChangePassword(ctx, accountID, oldPassword, newPassword)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAccountNotFound)
	})
}

func TestAccountServiceImpl_CreateAccountWithEmployment_WithGoMock(t *testing.T) {
	ctx := context.Background()
	defaultPassword := "defaultpassword"
	hashedDefaultPassword := "hashed_default_pwd_gomock"
	accountInput := &models.Account{FirstName: "New", LastName: "User", Email: "newfinal@example.com", Role: models.RoleEmployee, PhoneNumber: ""}
	employmentInput := &models.Employment{PositionTitle: "Final Dev", Status: models.EmploymentStatusActive, JobGradeID: nil, Salary: nil, HireDate: Ptr(time.Now().Truncate(24 * time.Hour)), TerminationDate: nil}

	// Use exact SQL strings (User needs to verify with GORM logs)
	accInsertQuery := "INSERT INTO `accounts` (`id`,`first_name`,`last_name`,`email`,`password`,`role`,`phone_number`,`created_at`,`updated_at`) VALUES (?,?,?,?,?,?,?,?,?)"
	empInsertQuery := "INSERT INTO `employments` (`id`,`account_id`,`job_grade_id`,`position_title`,`salary`,`hire_date`,`termination_date`,`status`,`created_at`,`updated_at`) VALUES (?,?,?,?,?,?,?,?,?,?)"

	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		mockEmploymentRepo := mocks.NewMockEmploymentRepository(ctrl)
		mockPwHasher := mocks.NewMockPasswordHasher(ctrl)
		gormDb, mockSql := setupGormWithSqlmock(t)
		service := NewAccountServiceImpl(mockAccountRepo, mockEmploymentRepo, nil, mockPwHasher, nil, defaultPassword, gormDb)
		localAccountInput := *accountInput
		localEmploymentInput := *employmentInput
		// *** 移除測試自行生成的 createdAccountID ***
		// createdAccountID := uuid.New()

		// --- 設定預期 (保持不變) ---
		mockSql.ExpectBegin()
		mockAccountRepo.EXPECT().GetAccountByEmail(gomock.Any(), gomock.Eq(localAccountInput.Email)).Return(nil, gorm.ErrRecordNotFound).Times(1)
		mockPwHasher.EXPECT().HashPassword(gomock.Eq(defaultPassword)).Return(hashedDefaultPassword, nil).Times(1)
		// Account Insert (sqlmock - AnyArg 會匹配 GORM 生成的任何 UUID)
		mockSql.ExpectExec(accInsertQuery).
			WithArgs(sqlmock.AnyArg(), localAccountInput.FirstName, localAccountInput.LastName, localAccountInput.Email, hashedDefaultPassword, localAccountInput.Role, localAccountInput.PhoneNumber, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))
		// Employment Insert (sqlmock - AnyArg 會匹配 GORM 生成的任何 UUID)
		mockSql.ExpectExec(empInsertQuery).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), localEmploymentInput.JobGradeID, localEmploymentInput.PositionTitle, localEmploymentInput.Salary, localEmploymentInput.HireDate, localEmploymentInput.TerminationDate, localEmploymentInput.Status, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mockSql.ExpectCommit()

		// --- 預期設定結束 ---
		// --- 執行 ---
		createdAccount, err := service.CreateAccountWithEmployment(ctx, &localAccountInput, &localEmploymentInput)

		// --- 斷言 ---
		require.NoError(t, err)
		require.NotNil(t, createdAccount)
		assert.Equal(t, localAccountInput.Email, createdAccount.Email)
		// ***  斷言: 檢查 ID 是否非零值即可 ***
		assert.NotEqual(t, uuid.Nil, createdAccount.ID, "Account ID should have been generated by BeforeCreate hook")
		assert.Equal(t, "", createdAccount.Password) // 密碼已清除
		assert.NoError(t, mockSql.ExpectationsWereMet())
	})
	t.Run("Failure - Email Exists", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		gormDb, mockSql := setupGormWithSqlmock(t)
		service := NewAccountServiceImpl(mockAccountRepo, nil, nil, nil, nil, defaultPassword, gormDb)
		localAccountInput := *accountInput
		localEmploymentInput := *employmentInput
		existingAccount := models.Account{ID: uuid.New(), Email: localAccountInput.Email}

		mockSql.ExpectBegin()
		mockAccountRepo.EXPECT().GetAccountByEmail(gomock.Any(), gomock.Eq(localAccountInput.Email)).Return(&existingAccount, nil).Times(1)
		mockSql.ExpectRollback()

		createdAccount, err := service.CreateAccountWithEmployment(ctx, &localAccountInput, &localEmploymentInput)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmailExists)
		assert.Nil(t, createdAccount)
		assert.NoError(t, mockSql.ExpectationsWereMet())
	})

	t.Run("Failure - Hash Error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		mockPwHasher := mocks.NewMockPasswordHasher(ctrl)
		gormDb, mockSql := setupGormWithSqlmock(t)
		service := NewAccountServiceImpl(mockAccountRepo, nil, nil, mockPwHasher, nil, defaultPassword, gormDb)
		localAccountInput := *accountInput
		localEmploymentInput := *employmentInput
		hashError := errors.New("hashing failed badly")

		mockSql.ExpectBegin()
		mockAccountRepo.EXPECT().GetAccountByEmail(gomock.Any(), gomock.Eq(localAccountInput.Email)).Return(nil, gorm.ErrRecordNotFound).Times(1)
		mockPwHasher.EXPECT().HashPassword(gomock.Eq(defaultPassword)).Return("", hashError).Times(1)
		mockSql.ExpectRollback()

		createdAccount, err := service.CreateAccountWithEmployment(ctx, &localAccountInput, &localEmploymentInput)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrPasswordHashingFailed)
		assert.Nil(t, createdAccount)
		assert.NoError(t, mockSql.ExpectationsWereMet())
	})

	t.Run("Failure - Create Account DB Error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		mockPwHasher := mocks.NewMockPasswordHasher(ctrl)
		gormDb, mockSql := setupGormWithSqlmock(t)
		service := NewAccountServiceImpl(mockAccountRepo, nil, nil, mockPwHasher, nil, defaultPassword, gormDb)
		localAccountInput := *accountInput
		localEmploymentInput := *employmentInput
		dbError := errors.New("account insert db error")

		mockSql.ExpectBegin()
		mockAccountRepo.EXPECT().GetAccountByEmail(gomock.Any(), gomock.Eq(localAccountInput.Email)).Return(nil, gorm.ErrRecordNotFound).Times(1)
		mockPwHasher.EXPECT().HashPassword(gomock.Eq(defaultPassword)).Return(hashedDefaultPassword, nil).Times(1)
		mockSql.ExpectExec(accInsertQuery).
			WithArgs(sqlmock.AnyArg(), localAccountInput.FirstName, localAccountInput.LastName, localAccountInput.Email, hashedDefaultPassword, localAccountInput.Role, localAccountInput.PhoneNumber, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(dbError) // Simulate DB error on account insert
		mockSql.ExpectRollback()

		createdAccount, err := service.CreateAccountWithEmployment(ctx, &localAccountInput, &localEmploymentInput)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAccountCreationFailed) // Service should return this specific error
		assert.Nil(t, createdAccount)
		assert.NoError(t, mockSql.ExpectationsWereMet())
	})

	t.Run("Failure - Create Employment DB Error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		mockEmploymentRepo := mocks.NewMockEmploymentRepository(ctrl) // Needed for New
		mockPwHasher := mocks.NewMockPasswordHasher(ctrl)
		gormDb, mockSql := setupGormWithSqlmock(t)
		service := NewAccountServiceImpl(mockAccountRepo, mockEmploymentRepo, nil, mockPwHasher, nil, defaultPassword, gormDb)
		localAccountInput := *accountInput
		localEmploymentInput := *employmentInput
		dbError := errors.New("employment insert db error")

		mockSql.ExpectBegin()
		mockAccountRepo.EXPECT().GetAccountByEmail(gomock.Any(), gomock.Eq(localAccountInput.Email)).Return(nil, gorm.ErrRecordNotFound).Times(1)
		mockPwHasher.EXPECT().HashPassword(gomock.Eq(defaultPassword)).Return(hashedDefaultPassword, nil).Times(1)
		// Account Insert succeeds
		mockSql.ExpectExec(accInsertQuery).
			WithArgs(sqlmock.AnyArg(), localAccountInput.FirstName, localAccountInput.LastName, localAccountInput.Email, hashedDefaultPassword, localAccountInput.Role, localAccountInput.PhoneNumber, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))
		// Employment Insert fails
		mockSql.ExpectExec(empInsertQuery).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), localEmploymentInput.JobGradeID, localEmploymentInput.PositionTitle, localEmploymentInput.Salary, localEmploymentInput.HireDate, localEmploymentInput.TerminationDate, localEmploymentInput.Status, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(dbError)
		mockSql.ExpectRollback()

		createdAccount, err := service.CreateAccountWithEmployment(ctx, &localAccountInput, &localEmploymentInput)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmploymentCreationFailed) // Service should return this specific error
		assert.Nil(t, createdAccount)
		assert.NoError(t, mockSql.ExpectationsWereMet())
	})

	t.Run("Failure - Commit Error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		mockEmploymentRepo := mocks.NewMockEmploymentRepository(ctrl)
		mockPwHasher := mocks.NewMockPasswordHasher(ctrl)
		gormDb, mockSql := setupGormWithSqlmock(t)
		service := NewAccountServiceImpl(mockAccountRepo, mockEmploymentRepo, nil, mockPwHasher, nil, defaultPassword, gormDb)
		localAccountInput := *accountInput
		localEmploymentInput := *employmentInput
		commitError := errors.New("commit failed")

		mockSql.ExpectBegin()
		mockAccountRepo.EXPECT().GetAccountByEmail(gomock.Any(), gomock.Eq(localAccountInput.Email)).Return(nil, gorm.ErrRecordNotFound).Times(1)
		mockPwHasher.EXPECT().HashPassword(gomock.Eq(defaultPassword)).Return(hashedDefaultPassword, nil).Times(1)
		mockSql.ExpectExec(accInsertQuery).WithArgs(sqlmock.AnyArg(), localAccountInput.FirstName, localAccountInput.LastName, localAccountInput.Email, hashedDefaultPassword, localAccountInput.Role, localAccountInput.PhoneNumber, sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
		mockSql.ExpectExec(empInsertQuery).WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), localEmploymentInput.JobGradeID, localEmploymentInput.PositionTitle, localEmploymentInput.Salary, localEmploymentInput.HireDate, localEmploymentInput.TerminationDate, localEmploymentInput.Status, sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
		mockSql.ExpectCommit().WillReturnError(commitError) // Commit fails
		// *** REMOVED ExpectRollback here ***

		createdAccount, err := service.CreateAccountWithEmployment(ctx, &localAccountInput, &localEmploymentInput)

		require.Error(t, err)
		// *** Check the wrapped error directly ***
		assert.Contains(t, err.Error(), "failed to finalize account creation", "Outer error message mismatch")
		require.True(t, errors.Is(err, commitError), "Underlying commit error not found in chain") // Use require.True with errors.Is
		assert.Nil(t, createdAccount)
		assert.NoError(t, mockSql.ExpectationsWereMet())
	})
}
