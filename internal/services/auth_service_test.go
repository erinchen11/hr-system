package services

import (
	"context"
	"testing"
	"time"

	"github.com/erinchen11/hr-system/internal/interfaces/mocks" 
	"github.com/erinchen11/hr-system/internal/models"

	gomock "github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)


func TestAuthServiceImpl_Authenticate_WithGoMock(t *testing.T) {
	ctx := context.Background()
	testEmail := "test@example.com"
	testPassword := "password123"
	hashedPassword := "hashed_password_from_db"
	baseAccountData := func() *models.Account { // Helper function (保持不變)
		return &models.Account{
			ID:        uuid.New(),
			FirstName: "Test",
			LastName:  "User",
			Email:     testEmail,
			Password:  hashedPassword,
			Role:      models.RoleEmployee,
			CreatedAt: time.Now().Add(-time.Hour),
			UpdatedAt: time.Now(),
		}
	}

	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t) // *** 創建 Controller ***
		defer ctrl.Finish()             // *** 自動驗證 Mock 調用 ***

		// *** 使用生成的 Mock 構造函數 ***
		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		mockPwChecker := mocks.NewMockPasswordChecker(ctrl)

		// *** 使用 NewAuthServiceImpl 創建 Service 實例 ***
		// (假設 NewAuthServiceImpl 接受 AccountRepository 和 PasswordChecker 介面)
		authService := NewAuthServiceImpl(mockAccountRepo, mockPwChecker)
		localAccountData := baseAccountData()

		// 1. 設定預期 (使用 gomock 風格)
		mockAccountRepo.EXPECT().
			GetAccountByEmail(gomock.Any(), gomock.Eq(testEmail)). // gomock.Any() 匹配 ctx, gomock.Eq() 精確匹配 email
			Return(localAccountData, nil).                         // 返回值
			Times(1)                                               // 預期調用一次 (預設)

		mockPwChecker.EXPECT().
			CheckPassword(gomock.Eq(hashedPassword), gomock.Eq(testPassword)). // 精確匹配參數
			Return(true).                                                      // 返回 true
			Times(1)

		// 2. 執行被測方法
		authenticatedAccount, err := authService.Authenticate(ctx, testEmail, testPassword)

		// 3. 斷言結果
		require.NoError(t, err)
		require.NotNil(t, authenticatedAccount)
		assert.Equal(t, localAccountData.ID, authenticatedAccount.ID)
		assert.Equal(t, testEmail, authenticatedAccount.Email)
		assert.Equal(t, "", authenticatedAccount.Password) // 驗證密碼被清除

		// 4. Mock 驗證由 defer ctrl.Finish() 處理
	})

	t.Run("Failure - Invalid Email (Not Found)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAccountRepo := mocks.NewMockAccountRepository(ctrl)
		mockPwChecker := mocks.NewMockPasswordChecker(ctrl)
		authService := NewAuthServiceImpl(mockAccountRepo, mockPwChecker)

		// 1. 設定預期
		mockAccountRepo.EXPECT().
			GetAccountByEmail(gomock.Any(), gomock.Eq(testEmail)).
			Return(nil, gorm.ErrRecordNotFound). // 返回未找到錯誤
			Times(1)
		// CheckPassword 不應被調用，無需設定 EXPECT

		// 2. 執行
		authenticatedAccount, err := authService.Authenticate(ctx, testEmail, testPassword)

		// 3. 斷言
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidCredentials)
		assert.Nil(t, authenticatedAccount)

		// 4. Mock 驗證由 defer ctrl.Finish() 處理
	})

}
