// 檔案路徑: internal/api/handlers/account_password_handler_test.go
package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing" // 導入 testing

	// ***  導入 AccountService 的 Mock ***
	"github.com/erinchen11/hr-system/internal/interfaces/mocks"
	common "github.com/erinchen11/hr-system/internal/models/common"
	"github.com/erinchen11/hr-system/internal/services" // 導入 services 以比較錯誤
	"github.com/gin-gonic/gin"

	gomock "github.com/golang/mock/gomock"
	"github.com/google/uuid" 
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAccountPasswordHandler_ChangePassword 測試 密碼 Handler
// ***  測試函數名反映 Handler 名稱 (可選) ***
func TestAccountPasswordHandler_ChangePassword(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 準備通用測試數據
	testUserUUID := uuid.New()            
	testUserIDStr := testUserUUID.String() // Context 中通常是 string
	validRequestBody := `{"old_password": "oldPassword123", "new_password": "newPassword456"}`
	oldPassword := "oldPassword123"
	newPassword := "newPassword456"

	// --- Test Cases ---
	testCases := []struct {
		name             string
		userIDInContext  interface{} // 模擬 Context 中的 user_id
		requestBody      string
		setupMocks       func(accountSvc *mocks.MockAccountService) // ***  依賴 MockAccountService ***
		expectedStatus   int
		expectedResponse common.Response
	}{
		{
			name:            "Success",
			userIDInContext: testUserIDStr,
			requestBody:     validRequestBody,
			setupMocks: func(accountSvc *mocks.MockAccountService) {
				// ***  預期 AccountService 的 ChangePassword 被調用，參數是 UUID ***
				accountSvc.EXPECT().
					ChangePassword(gomock.Any(), /* ctx */
							gomock.Eq(testUserUUID), /* accountID uuid.UUID */
							gomock.Eq(oldPassword),
							gomock.Eq(newPassword)).
					Return(nil). // 模擬成功
					Times(1)
			},
			expectedStatus:   http.StatusOK,
			expectedResponse: common.Response{Code: http.StatusOK, Message: "Password updated successfully."}, // 移除 "Please login again."
		},
		{
			name:             "Unauthorized - Missing UserID in Context",
			userIDInContext:  nil,
			requestBody:      validRequestBody,
			setupMocks:       nil, // Service 不應被調用
			expectedStatus:   http.StatusUnauthorized,
			expectedResponse: common.Response{Code: http.StatusUnauthorized, Message: "Unauthorized: Missing user identity"},
		},
		{
			name:             "Internal Error - Invalid UserID Type in Context",
			userIDInContext:  123, // 模擬錯誤類型
			requestBody:      validRequestBody,
			setupMocks:       nil,
			expectedStatus:   http.StatusInternalServerError,
			expectedResponse: common.Response{Code: http.StatusInternalServerError, Message: "Internal error processing user identity"},
		},
		{
			name:             "Internal Error - Invalid UserID Format in Context",
			userIDInContext:  "not-a-uuid", // 模擬無效 UUID 格式
			requestBody:      validRequestBody,
			setupMocks:       nil, // Service 不應被調用
			expectedStatus:   http.StatusInternalServerError,
			expectedResponse: common.Response{Code: http.StatusInternalServerError, Message: "Internal error processing user identity format"},
		},
		{
			name:             "Bad Request - Invalid JSON",
			userIDInContext:  testUserIDStr,
			requestBody:      `{"old_password": "pass", "new_password": }`,
			setupMocks:       nil,
			expectedStatus:   http.StatusBadRequest,
			expectedResponse: common.Response{Code: http.StatusBadRequest, Message: "Invalid request body"},
		},
		{
			name:             "Bad Request - Missing Field",
			userIDInContext:  testUserIDStr,
			requestBody:      `{"old_password": "oldPassword123"}`,
			setupMocks:       nil,
			expectedStatus:   http.StatusBadRequest,
			expectedResponse: common.Response{Code: http.StatusBadRequest, Message: "Invalid request body"},
		},
		{
			name:             "Bad Request - New Password same as Old",
			userIDInContext:  testUserIDStr,
			requestBody:      `{"old_password": "password123", "new_password": "password123"}`,
			setupMocks:       nil, // Service 不應被調用
			expectedStatus:   http.StatusBadRequest,
			expectedResponse: common.Response{Code: http.StatusBadRequest, Message: "New password cannot be the same as the old password"},
		},
		{
			name:            "Service Error - Invalid Credentials",
			userIDInContext: testUserIDStr,
			requestBody:     validRequestBody,
			setupMocks: func(accountSvc *mocks.MockAccountService) {
				// ***  預期 AccountService 返回 ErrInvalidCredentials ***
				accountSvc.EXPECT().ChangePassword(gomock.Any(), testUserUUID, oldPassword, newPassword).Return(services.ErrInvalidCredentials).Times(1)
			},
			expectedStatus:   http.StatusUnauthorized,
			expectedResponse: common.Response{Code: http.StatusUnauthorized, Message: "Old password is incorrect"},
		},
		{
			name:            "Service Error - Account Not Found",
			userIDInContext: testUserIDStr,
			requestBody:     validRequestBody,
			setupMocks: func(accountSvc *mocks.MockAccountService) {
				// ***  預期 AccountService 返回 ErrAccountNotFound ***
				accountSvc.EXPECT().ChangePassword(gomock.Any(), testUserUUID, oldPassword, newPassword).Return(services.ErrAccountNotFound).Times(1)
			},
			expectedStatus:   http.StatusNotFound,
			expectedResponse: common.Response{Code: http.StatusNotFound, Message: "User account not found"},
		},
		{
			name:            "Service Error - Hashing Failed",
			userIDInContext: testUserIDStr,
			requestBody:     validRequestBody,
			setupMocks: func(accountSvc *mocks.MockAccountService) {
				accountSvc.EXPECT().ChangePassword(gomock.Any(), testUserUUID, oldPassword, newPassword).Return(services.ErrPasswordHashingFailed).Times(1)
			},
			expectedStatus:   http.StatusInternalServerError,
			expectedResponse: common.Response{Code: http.StatusInternalServerError, Message: "Failed to process new password"},
		},
		{
			name:            "Service Error - Update Failed",
			userIDInContext: testUserIDStr,
			requestBody:     validRequestBody,
			setupMocks: func(accountSvc *mocks.MockAccountService) {
				accountSvc.EXPECT().ChangePassword(gomock.Any(), testUserUUID, oldPassword, newPassword).Return(services.ErrPasswordUpdateFailed).Times(1)
			},
			expectedStatus:   http.StatusInternalServerError,
			expectedResponse: common.Response{Code: http.StatusInternalServerError, Message: "Failed to update password"},
		},
		{
			name:            "Service Error - Unexpected",
			userIDInContext: testUserIDStr,
			requestBody:     validRequestBody,
			setupMocks: func(accountSvc *mocks.MockAccountService) {
				unexpectedError := errors.New("some unexpected service error")
				accountSvc.EXPECT().ChangePassword(gomock.Any(), testUserUUID, oldPassword, newPassword).Return(unexpectedError).Times(1)
			},
			expectedStatus:   http.StatusInternalServerError,
			expectedResponse: common.Response{Code: http.StatusInternalServerError, Message: "An unexpected error occurred"},
		},
	}

	// --- 執行測試案例 ---
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish() // 在每個子測試結束時驗證 mock

			mockAccountSvc := mocks.NewMockAccountService(ctrl) // *** 使用 MockAccountService ***

			// 創建 Handler 實例，注入 Mock Service
			// ***  使用新的構造函數和 Mock ***
			accountPasswordHandler := NewAccountPasswordHandler(mockAccountSvc)

			// 設置 Mock 預期 (如果有的話)
			if tc.setupMocks != nil {
				tc.setupMocks(mockAccountSvc)
			}

			// 模擬 HTTP 環境
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/account/password", bytes.NewBufferString(tc.requestBody)) // 假設路徑
			req.Header.Set("Content-Type", "application/json")
			c.Request = req

			// 模擬 Context (設置 user_id)
			if tc.userIDInContext != nil {
				c.Set("user_id", tc.userIDInContext)
			}

			// 調用 Handler 方法
			// ***  : 調用新的 Handler 實例的方法 ***
			accountPasswordHandler.ChangePassword(c)

			// 斷言 HTTP 狀態碼
			assert.Equal(t, tc.expectedStatus, recorder.Code)

			// 斷言響應體
			var actualResponse common.Response
			err := json.Unmarshal(recorder.Body.Bytes(), &actualResponse)
			require.NoError(t, err, "Response body should be valid JSON")

			// 比較 Code 和 Message
			assert.Equal(t, tc.expectedResponse.Code, actualResponse.Code, "Response code mismatch")
			// 對於 Bad Request 和 Internal Error，可以只比較 Code，因為 Message 可能不同
			if tc.expectedStatus == http.StatusBadRequest || tc.expectedStatus == http.StatusInternalServerError {
				// 可以選擇不比較 Message，或者比較部分內容
				// assert.Contains(t, actualResponse.Message, "...")
			} else {
				assert.Equal(t, tc.expectedResponse.Message, actualResponse.Message, "Response message mismatch")
			}
			// 成功時 Data 應為 nil
			if tc.expectedStatus == http.StatusOK {
				assert.Nil(t, actualResponse.Data)
			}
		})
	}
}

