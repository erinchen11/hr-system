
package handlers 

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/erinchen11/hr-system/internal/interfaces/mocks" 
	"github.com/erinchen11/hr-system/internal/models"
	common "github.com/erinchen11/hr-system/internal/models/common"

	"github.com/gin-gonic/gin"
	gomock "github.com/golang/mock/gomock" 
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 測試 ListLeaveRequestsHandler 的 ListLeaveRequests 方法
func TestListLeaveRequestsHandler_ListLeaveRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// --- 通用測試數據 ---
	hrClaims := &models.Claims{UserID: uuid.New().String(), Role: models.RoleHR, Email: "hr@test.com"}
	employeeClaims := &models.Claims{UserID: uuid.New().String(), Role: models.RoleEmployee, Email: "emp@test.com"}
	superAdminClaims := &models.Claims{UserID: uuid.New().String(), Role: models.RoleSuperAdmin, Email: "super@test.com"}

	mockReq1ID := uuid.New()
	mockReq2ID := uuid.New()
	mockAccount1ID := uuid.New()
	mockAccount2ID := uuid.New()

	// 模擬 Service 返回的數據 (注意：密碼應已在 Service 層被清除)
	mockLeaveRequests := []models.LeaveRequest{
		{ID: mockReq1ID, AccountID: mockAccount1ID, LeaveType: models.LeaveTypeSick, Status: models.LeaveStatusApproved, Account: models.Account{ID: mockAccount1ID, Email: "test1@co.co"}},
		{ID: mockReq2ID, AccountID: mockAccount2ID, LeaveType: models.LeaveTypeAnnual, Status: models.LeaveStatusPending, Account: models.Account{ID: mockAccount2ID, Email: "test2@co.co"}},
	}

	// --- 表格驅動測試 ---
	testCases := []struct {
		name                 string
		callerClaims         interface{} // 使用 interface{} 以測試無效類型
		setupMocks           func(mockLeaveSvc *mocks.MockLeaveRequestService)
		expectedStatusCode   int
		expectedResponseCode int
		expectedMessage      string
		expectedDataLength   int // 預期返回列表的長度, -1 表示不檢查 Data 或預期 nil
	}{
		{
			name:         "Success - HR gets list",
			callerClaims: hrClaims, // HR 身份
			setupMocks: func(mockLeaveSvc *mocks.MockLeaveRequestService) {
				mockLeaveSvc.EXPECT().ListAllRequests(gomock.Any()).Return(mockLeaveRequests, nil).Times(1)
			},
			expectedStatusCode:   http.StatusOK,
			expectedResponseCode: http.StatusOK,
			expectedMessage:      "Success",
			expectedDataLength:   2, // 預期返回 2 筆
		},
		{
			name:         "Success - HR gets empty list",
			callerClaims: hrClaims,
			setupMocks: func(mockLeaveSvc *mocks.MockLeaveRequestService) {
				mockLeaveSvc.EXPECT().ListAllRequests(gomock.Any()).Return([]models.LeaveRequest{}, nil).Times(1) // 返回空列表
			},
			expectedStatusCode:   http.StatusOK,
			expectedResponseCode: http.StatusOK,
			expectedMessage:      "Success",
			expectedDataLength:   0, // 預期返回空列表
		},
		{
			name:                 "Forbidden - Employee tries to get list",
			callerClaims:         employeeClaims, // 員工身份
			setupMocks:           nil,            // Service 不應被調用
			expectedStatusCode:   http.StatusForbidden,
			expectedResponseCode: http.StatusForbidden,
			expectedMessage:      "Permission denied: Only HR can view leave requests",
			expectedDataLength:   -1,
		},
		{
			name:                 "Forbidden - SuperAdmin tries to get list (Based on current logic)",
			callerClaims:         superAdminClaims, // 超級管理員身份
			setupMocks:           nil,              // Service 不應被調用 (根據 Handler 目前的邏輯)
			expectedStatusCode:   http.StatusForbidden,
			expectedResponseCode: http.StatusForbidden,
			expectedMessage:      "Permission denied: Only HR can view leave requests",
			expectedDataLength:   -1,
		},
		{
			name:                 "Unauthorized - Missing Claims",
			callerClaims:         nil,
			setupMocks:           nil,
			expectedStatusCode:   http.StatusUnauthorized,
			expectedResponseCode: http.StatusUnauthorized,
			expectedMessage:      "Unauthorized: Missing user claims",
			expectedDataLength:   -1,
		},
		{
			name:                 "Internal Server Error - Invalid Claims Type",
			callerClaims:         "not a claims struct", // 錯誤類型
			setupMocks:           nil,
			expectedStatusCode:   http.StatusInternalServerError,
			expectedResponseCode: http.StatusInternalServerError,
			expectedMessage:      "Internal error processing user identity",
			expectedDataLength:   -1,
		},
		{
			name:         "Internal Server Error - Service Error",
			callerClaims: hrClaims,
			setupMocks: func(mockLeaveSvc *mocks.MockLeaveRequestService) {
				dbError := errors.New("database connection issue")
				mockLeaveSvc.EXPECT().ListAllRequests(gomock.Any()).Return(nil, dbError).Times(1) // 模擬 Service 出錯
			},
			expectedStatusCode:   http.StatusInternalServerError,
			expectedResponseCode: http.StatusInternalServerError,
			expectedMessage:      "Failed to fetch leave requests",
			expectedDataLength:   -1,
		},
	}

	// --- 執行測試迴圈 ---
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLeaveSvc := mocks.NewMockLeaveRequestService(ctrl)
			// 假設 Handler 構造函數名稱與結構體名對應
			listHandler := NewListLeaveRequestsHandler(mockLeaveSvc)

			// 設置 Mock 預期
			if tc.setupMocks != nil {
				tc.setupMocks(mockLeaveSvc)
			}

			// 模擬 HTTP 環境
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/hr/leave-requests", nil) // 假設路由
			c.Request = req

			// 模擬 Context (設置 claims)
			if tc.callerClaims != nil {
				c.Set("claims", tc.callerClaims)
			}

			// 調用 Handler 方法
			listHandler.ListLeaveRequests(c)

			// 斷言 HTTP 狀態碼
			require.Equal(t, tc.expectedStatusCode, recorder.Code, "HTTP status code mismatch")

			// 斷言響應體
			var actualResponse common.Response
			err := json.Unmarshal(recorder.Body.Bytes(), &actualResponse)
			require.NoError(t, err, "Response body should be valid JSON for case: %s", tc.name)

			assert.Equal(t, tc.expectedResponseCode, actualResponse.Code, "Response code mismatch")
			assert.Equal(t, tc.expectedMessage, actualResponse.Message, "Response message mismatch")

			// 驗證 Data 部分
			if tc.expectedDataLength >= 0 {
				require.NotNil(t, actualResponse.Data, "Response data should not be nil for success case")
				// 將 Data 斷言為一個 切片
				dataSlice, ok := actualResponse.Data.([]interface{})
				require.True(t, ok, "Response data should be a slice")
				assert.Len(t, dataSlice, tc.expectedDataLength, "Data slice length mismatch")

				// 可選：如果需要，可以進一步解析 dataSlice 中的元素並驗證內容
				if tc.expectedDataLength > 0 {
					// 例如，檢查第一個元素的結構 (假設 Service 返回的結構與 models.LeaveRequest 匹配)
					firstElementBytes, _ := json.Marshal(dataSlice[0])
					var firstElement models.LeaveRequest // 或者一個 DTO
					err = json.Unmarshal(firstElementBytes, &firstElement)
					require.NoError(t, err, "Failed to unmarshal first element of data")
					assert.Equal(t, mockReq1ID, firstElement.ID) // 比較 ID
					// 注意：這裡返回的是 models.LeaveRequest，如果 Service 清理了密碼，Account.Password 應為空
					assert.Equal(t, "", firstElement.Account.Password)
				}

			} else {
				assert.Nil(t, actualResponse.Data, "Response data should be nil for failure case")
			}
		})
	}
}
