package middleware

import (
	// <-- 導入 context
	"encoding/json"
	"errors"
	"log"
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
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware_Authenticate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// ---- 通用測試資料 ----
	mockUserID := uuid.New().String()
	mockClaims := &models.Claims{UserID: mockUserID, Email: "test@middleware.com", Role: 1}
	validToken := "valid.test.token"
	invalidToken := "invalid.test.token"

	testCases := []struct {
		name             string
		authHeader       string
		setupMocks       func(tokenSvc *mocks.MockTokenService)
		expectedStatus   int
		expectNextCalled bool // *** 用 IsAborted 判斷 ***
		expectedResponse *common.Response
		expectedUserID   string // 用於驗證 Context 設置
		// 可以添加其他期望的 Context 值
	}{
		{
			name:       "Success - Valid Token",
			authHeader: "Bearer " + validToken,
			setupMocks: func(tokenSvc *mocks.MockTokenService) {
				tokenSvc.EXPECT().ValidateToken(gomock.Any(), validToken).Return(mockClaims, nil).Times(1)
			},
			expectedStatus:   http.StatusOK, // 期望成功時狀態碼是 200 (由後續 handler 設置)
			expectNextCalled: true,          // 期望 c.Next() 被調用
			expectedResponse: nil,
			expectedUserID:   mockUserID,
		},
		{
			name:             "Fail - Missing Authorization Header",
			authHeader:       "",
			setupMocks:       nil,
			expectedStatus:   http.StatusUnauthorized,
			expectNextCalled: false, // 期望請求被中止
			expectedResponse: &common.Response{Code: http.StatusUnauthorized, Message: "Authorization header required"},
		},
		{
			name:             "Fail - Invalid Format (No Bearer)",
			authHeader:       validToken,
			setupMocks:       nil,
			expectedStatus:   http.StatusUnauthorized,
			expectNextCalled: false,
			expectedResponse: &common.Response{Code: http.StatusUnauthorized, Message: "Invalid Authorization format (Bearer <token>)"},
		},
		{
			name:             "Fail - Invalid Format (Wrong Scheme)",
			authHeader:       "Basic " + validToken,
			setupMocks:       nil,
			expectedStatus:   http.StatusUnauthorized,
			expectNextCalled: false,
			expectedResponse: &common.Response{Code: http.StatusUnauthorized, Message: "Invalid Authorization format (Bearer <token>)"},
		},
		{
			name:       "Fail - Token Validation Error",
			authHeader: "Bearer " + invalidToken,
			setupMocks: func(tokenSvc *mocks.MockTokenService) {
				validationError := errors.New("token signature is invalid")
				tokenSvc.EXPECT().ValidateToken(gomock.Any(), invalidToken).Return(nil, validationError).Times(1)
			},
			expectedStatus:   http.StatusUnauthorized,
			expectNextCalled: false,
			expectedResponse: &common.Response{Code: http.StatusUnauthorized, Message: "Invalid or expired token"},
		},
		{
			name:       "Fail - Token Expired (Simulated by Service Error)",
			authHeader: "Bearer " + validToken,
			setupMocks: func(tokenSvc *mocks.MockTokenService) {
				tokenSvc.EXPECT().ValidateToken(gomock.Any(), validToken).Return(nil, services.ErrTokenExpiredOrRevoked).Times(1)
			},
			expectedStatus:   http.StatusUnauthorized,
			expectNextCalled: false,
			expectedResponse: &common.Response{Code: http.StatusUnauthorized, Message: "Invalid or expired token"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			// defer ctrl.Finish()

			mockTokenSvc := mocks.NewMockTokenService(ctrl)
			authMiddleware := NewAuthMiddleware(mockTokenSvc)

			// 設置 Mock 預期
			if tc.setupMocks != nil {
				tc.setupMocks(mockTokenSvc)
			}

			// --- 直接調用 Middleware 返回的 HandlerFunc ---
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)                 // 創建 Context
			req, _ := http.NewRequest(http.MethodGet, "/test", nil) // 創建請求
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			c.Request = req // 附加請求到 Context

			// *** 獲取 Middleware 的 HandlerFunc ***
			middlewareFunc := authMiddleware.Authenticate()

			// *** 執行 HandlerFunc ***
			middlewareFunc(c)

			// 1. 斷言請求是否按預期被中止或繼續
			assert.Equal(t, !tc.expectNextCalled, c.IsAborted(), "Middleware abortion status mismatch")

			// 2. 如果期望請求被中止 (即有錯誤發生)
			if !tc.expectNextCalled {
				
				if recorder.Code != 0 { // 只有在 recorder.Code 被設置時才比較
					assert.Equal(t, tc.expectedStatus, recorder.Code, "HTTP status code mismatch (might be unreliable on abort)")
				} else {
					log.Printf("Warning: recorder.Code is 0 in test '%s', relying on response body check.", tc.name)
				}

				// *** 關鍵：斷言響應體中的 Code 和 Message ***
				require.NotNil(t, tc.expectedResponse, "expectedResponse should be set for failed cases")
				var actualResponse common.Response
				err := json.Unmarshal(recorder.Body.Bytes(), &actualResponse)
				require.NoError(t, err, "Error response body should be valid JSON")

				assert.Equal(t, tc.expectedResponse.Code, actualResponse.Code, "Error response JSON code mismatch")
				assert.Equal(t, tc.expectedResponse.Message, actualResponse.Message, "Error response JSON message mismatch")

			} else { // 如果期望請求成功繼續
				assert.False(t, c.IsAborted()) // 確保未中止
				// 成功時，狀態碼通常由後續的實際 handler 決定，
				// dummy handler 中是 200，所以可以斷言 200
				assert.Equal(t, http.StatusOK, recorder.Code, "Expected StatusOK when next handler is called")

				// 斷言 Context 中的值
				userID, exists := c.Get("user_id")
				assert.True(t, exists, "user_id should exist in context on success")
				assert.Equal(t, tc.expectedUserID, userID, "user_id in context mismatch")

				claims, exists := c.Get("claims")
				assert.True(t, exists, "claims should exist in context on success")
				assert.NotNil(t, claims)
				if claimsTyped, ok := claims.(*models.Claims); ok {
					assert.Equal(t, tc.expectedUserID, claimsTyped.UserID)
				} else {
					t.Errorf("Claims in context are not of expected type *models.Claims")
				}
			}
		})
	}
}
