package handlers
import (
	"bytes"         
	"encoding/json" 
	"errors"
	"net/http"         
	"net/http/httptest" 
	"testing"

	"github.com/erinchen11/hr-system/internal/interfaces/mocks" 
	"github.com/erinchen11/hr-system/internal/models"
	common "github.com/erinchen11/hr-system/internal/models/common" 
	"github.com/erinchen11/hr-system/internal/services"            
	"github.com/gin-gonic/gin"                                   
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestLoginHandler_Login(t *testing.T) {
	// 設置 Gin 為測試模式
	gin.SetMode(gin.TestMode)

	// 準備通用的 Mock 返回數據
	mockUserID := uuid.New()
	mockUser := &models.Account{
		ID:        mockUserID,
		Email:     "test@example.com",
		Role:      2,
		FirstName: "Test",
		LastName:  "User",
		// Password HASH 不需要返回給 Handler
	}
	mockToken := "mock.jwt.token"

	testCases := []struct {
		name            string
		requestBody     string 
		setupMocks      func(authSvc *mocks.MockAuthService, tokenSvc *mocks.MockTokenService)
		expectedStatus  int  
		expectErrorBody bool  
		expectedToken   string 
		expectedEmail   string 
	}{
		{
			name:        "Success",
			requestBody: `{"email": "test@example.com", "password": "password123"}`,
			setupMocks: func(authSvc *mocks.MockAuthService, tokenSvc *mocks.MockTokenService) {
				// 期望 AuthService.Authenticate 成功返回用戶
				authSvc.EXPECT().Authenticate(gomock.Any(), "test@example.com", "password123").Return(mockUser, nil).Times(1)
				// 期望 TokenService.GenerateAndCacheToken 成功返回 Token
				// 注意：這裡需要傳遞 mockUser 指針
				tokenSvc.EXPECT().GenerateAndCacheToken(gomock.Any(), mockUser).Return(mockToken, nil).Times(1)
			},
			expectedStatus: http.StatusOK, // 200
			expectedToken:  mockToken,
			expectedEmail:  mockUser.Email,
		},
		{
			name:            "Invalid JSON Format",
			requestBody:     `{"email": "test@example.com", "password": }`, // 錯誤的 JSON
			setupMocks:      nil,                                           // Service 不應被調用
			expectedStatus:  http.StatusBadRequest,                         // 400
			expectErrorBody: true,
		},
		{
			name:            "Missing Required Field",
			requestBody:     `{"email": "test@example.com"}`, // 缺少 password
			setupMocks:      nil,
			expectedStatus:  http.StatusBadRequest,
			expectErrorBody: true,
		},
		{
			name:        "Authentication Failed",
			requestBody: `{"email": "test@example.com", "password": "wrongpassword"}`,
			setupMocks: func(authSvc *mocks.MockAuthService, tokenSvc *mocks.MockTokenService) {
				// 期望 AuthService.Authenticate 返回錯誤
				// 使用 Service 層定義的錯誤
				authSvc.EXPECT().Authenticate(gomock.Any(), "test@example.com", "wrongpassword").Return(nil, services.ErrInvalidCredentials).Times(1)
				// TokenService 不應被調用
			},
			expectedStatus:  http.StatusUnauthorized, // 401
			expectErrorBody: true,
		},
		{
			name:        "Token Generation Failed",
			requestBody: `{"email": "test@example.com", "password": "password123"}`,
			setupMocks: func(authSvc *mocks.MockAuthService, tokenSvc *mocks.MockTokenService) {
				authSvc.EXPECT().Authenticate(gomock.Any(), "test@example.com", "password123").Return(mockUser, nil).Times(1)
				// 期望 TokenService.GenerateAndCacheToken 返回錯誤
				tokenError := errors.New("failed to generate") // 可以用 services.ErrTokenGenerationFailed
				tokenSvc.EXPECT().GenerateAndCacheToken(gomock.Any(), mockUser).Return("", tokenError).Times(1)
			},
			expectedStatus:  http.StatusInternalServerError, // 500
			expectErrorBody: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			mockAuthSvc := mocks.NewMockAuthService(ctrl)
			mockTokenSvc := mocks.NewMockTokenService(ctrl)

			// 創建被測 Handler 實例，注入 Mocks
			loginHandler := NewLoginHandler(mockAuthSvc, mockTokenSvc)

			// 設置 Mock 的預期行為
			if tc.setupMocks != nil {
				tc.setupMocks(mockAuthSvc, mockTokenSvc)
			}

			// --- 模擬 HTTP 請求 ---
			// 創建一個 ResponseRecorder 來捕獲響應
			recorder := httptest.NewRecorder()
			// 創建一個測試用的 Gin Context
			c, _ := gin.CreateTestContext(recorder)
			// 創建一個 HTTP 請求
			// 使用 bytes.NewBufferString 將 JSON 字串轉為 io.Reader
			req, err := http.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(tc.requestBody))
			assert.NoError(t, err)                             // 確保創建請求沒問題
			req.Header.Set("Content-Type", "application/json") // 設置 Header
			// 將請求附加到 Gin Context
			c.Request = req

			// --- 調用 Handler 方法 ---
			loginHandler.Login(c)

			// --- 斷言響應狀態碼 ---
			assert.Equal(t, tc.expectedStatus, recorder.Code)

			// --- 斷言響應體 (可選但推薦) ---
			var resp common.Response
			err = json.Unmarshal(recorder.Body.Bytes(), &resp) // 解析響應體
			assert.NoError(t, err, "Response body should be valid JSON")

			if tc.expectedStatus == http.StatusOK {
				assert.Equal(t, http.StatusOK, resp.Code)
				assert.NotEmpty(t, resp.Message)
				// 檢查 Data 部分
				respData, ok := resp.Data.(map[string]interface{}) // Data 是 gin.H，底層是 map
				assert.True(t, ok, "Response data should be a map")
				if ok {
					assert.Equal(t, tc.expectedToken, respData["token"])
					userData, userOk := respData["user"].(map[string]interface{})
					assert.True(t, userOk, "User data should be a map")
					if userOk {
						assert.Equal(t, tc.expectedEmail, userData["email"])
						// 可以根據需要斷言其他用戶欄位
					}
				}
			} else if tc.expectErrorBody { // 如果期望是錯誤響應
				assert.Equal(t, tc.expectedStatus, resp.Code) // 驗證 Code 欄位
				assert.NotEmpty(t, resp.Message)              // 驗證有錯誤消息
				assert.Nil(t, resp.Data)                      // 錯誤時 Data 通常為 nil
			}
		})
	}
}
