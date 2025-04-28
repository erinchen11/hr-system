package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/erinchen11/hr-system/internal/interfaces/mocks" 
	"github.com/erinchen11/hr-system/internal/models"
	common "github.com/erinchen11/hr-system/internal/models/common"
	"github.com/erinchen11/hr-system/internal/services" 
	"github.com/erinchen11/hr-system/internal/utils"
	"github.com/gin-gonic/gin"
	gomock "github.com/golang/mock/gomock" 
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestViewLeaveStatusHandler_ViewLeaveStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// --- 通用測試數據 ---
	employeeID := uuid.New()
	employeeIDStr := employeeID.String()
	employeeClaims := &models.Claims{UserID: employeeIDStr, Role: models.RoleEmployee, Email: "emp@test.com"}
	hrClaims := &models.Claims{UserID: uuid.New().String(), Role: models.RoleHR, Email: "hr@test.com"}

	now := time.Now()
	mockLeaveRequests := []models.LeaveRequest{
		{
			ID:          uuid.New(),
			AccountID:   employeeID,
			LeaveType:   models.LeaveTypeSick,
			StartDate:   now.AddDate(0, 0, -5),
			EndDate:     now.AddDate(0, 0, -4),
			Reason:      "Flu",
			Status:      models.LeaveStatusApproved,
			RequestedAt: now.AddDate(0, 0, -6),
			ApprovedAt:  utils.Ptr(now.AddDate(0, 0, -5)),
			// Account 和 Approver 欄位可能由 Service 層返回，但 Handler 的 DTO 會忽略
		},
		{
			ID:          uuid.New(),
			AccountID:   employeeID,
			LeaveType:   models.LeaveTypeAnnual,
			StartDate:   now.AddDate(0, 0, 10),
			EndDate:     now.AddDate(0, 0, 12),
			Reason:      "", // No reason
			Status:      models.LeaveStatusPending,
			RequestedAt: now.AddDate(0, 0, -1),
			ApprovedAt:  nil,
		},
	}

	// --- 表格驅動測試 ---
	testCases := []struct {
		name                 string
		callerClaims         interface{}
		setupMocks           func(mockLeaveSvc *mocks.MockLeaveRequestService)
		expectedStatusCode   int
		expectedResponseCode int
		expectedMessage      string
		expectedDataLength   int    // -1 表示不檢查 Data 或預期 Data 為 nil
		expectedFirstStatus  string // 用於成功案例檢查第一筆資料
	}{
		{
			name:         "Success - Employee views their requests",
			callerClaims: employeeClaims,
			setupMocks: func(mockLeaveSvc *mocks.MockLeaveRequestService) {
				mockLeaveSvc.EXPECT().ListAccountRequests(gomock.Any(), gomock.Eq(employeeIDStr)).Return(mockLeaveRequests, nil).Times(1)
			},
			expectedStatusCode:   http.StatusOK,
			expectedResponseCode: http.StatusOK,
			expectedMessage:      "Success",
			expectedDataLength:   2,                          // 預期返回 2 筆 DTO
			expectedFirstStatus:  models.LeaveStatusApproved, // 驗證映射和排序（假設 Service 返回是按 RequestedAt desc）
		},
		{
			name:         "Success - Employee has no requests",
			callerClaims: employeeClaims,
			setupMocks: func(mockLeaveSvc *mocks.MockLeaveRequestService) {
				mockLeaveSvc.EXPECT().ListAccountRequests(gomock.Any(), gomock.Eq(employeeIDStr)).Return([]models.LeaveRequest{}, nil).Times(1) // 返回空列表
			},
			expectedStatusCode:   http.StatusOK,
			expectedResponseCode: http.StatusOK,
			expectedMessage:      "Success",
			expectedDataLength:   0, // 預期返回空列表
		},
		{
			name:                 "Forbidden - HR tries to view employee status via this endpoint",
			callerClaims:         hrClaims, // HR 身份
			setupMocks:           nil,      // Service 不應被調用
			expectedStatusCode:   http.StatusForbidden,
			expectedResponseCode: http.StatusForbidden,
			expectedMessage:      "Permission denied: Only employees can view their leave status",
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
			callerClaims:         &struct{ Name string }{"fake"}, // 錯誤類型
			setupMocks:           nil,
			expectedStatusCode:   http.StatusInternalServerError,
			expectedResponseCode: http.StatusInternalServerError,
			expectedMessage:      "Internal error processing user identity",
			expectedDataLength:   -1,
		},
		{
			name:         "Service Error - Account Not Found",
			callerClaims: employeeClaims, // Claims 有效，但 Service 找不到帳戶
			setupMocks: func(mockLeaveSvc *mocks.MockLeaveRequestService) {
				mockLeaveSvc.EXPECT().ListAccountRequests(gomock.Any(), gomock.Eq(employeeIDStr)).Return(nil, services.ErrAccountNotFound).Times(1)
			},
			expectedStatusCode:   http.StatusNotFound, // Handler 應處理此錯誤
			expectedResponseCode: http.StatusNotFound,
			expectedMessage:      "User account not found",
			expectedDataLength:   -1,
		},
		{
			name:         "Service Error - Other DB Error",
			callerClaims: employeeClaims,
			setupMocks: func(mockLeaveSvc *mocks.MockLeaveRequestService) {
				dbError := errors.New("connection refused")
				mockLeaveSvc.EXPECT().ListAccountRequests(gomock.Any(), gomock.Eq(employeeIDStr)).Return(nil, dbError).Times(1)
			},
			expectedStatusCode:   http.StatusInternalServerError,
			expectedResponseCode: http.StatusInternalServerError,
			expectedMessage:      "Failed to retrieve leave records",
			expectedDataLength:   -1,
		},
	}

	// --- 執行測試迴圈 ---
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLeaveSvc := mocks.NewMockLeaveRequestService(ctrl)
			// 假設 Handler 只需要 LeaveRequestService
			viewHandler := NewViewLeaveStatusHandler(mockLeaveSvc)

			if tc.setupMocks != nil {
				tc.setupMocks(mockLeaveSvc)
			}

			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/employee/leave-status", nil) // 假設路由
			c.Request = req

			if tc.callerClaims != nil {
				c.Set("claims", tc.callerClaims)
			}

			viewHandler.ViewLeaveStatus(c)

			require.Equal(t, tc.expectedStatusCode, recorder.Code, "HTTP status code mismatch")

			var actualResponse common.Response
			err := json.Unmarshal(recorder.Body.Bytes(), &actualResponse)
			require.NoError(t, err, "Response body should be valid JSON for case: %s", tc.name)

			assert.Equal(t, tc.expectedResponseCode, actualResponse.Code, "Response code mismatch")
			assert.Equal(t, tc.expectedMessage, actualResponse.Message, "Response message mismatch")

			// 驗證 Data 部分
			if tc.expectedDataLength >= 0 {
				require.NotNil(t, actualResponse.Data, "Response data should not be nil for success case")
				// 將 actualResponse.Data 斷言回 DTO 切片
				var responseData []LeaveRequestStatusDTO
				dataBytes, _ := json.Marshal(actualResponse.Data)
				err = json.Unmarshal(dataBytes, &responseData)
				require.NoError(t, err, "Failed to unmarshal response data into []LeaveRequestStatusDTO")

				assert.Len(t, responseData, tc.expectedDataLength, "Data slice length mismatch")
				if tc.expectedDataLength > 0 && tc.expectedFirstStatus != "" {
					// 可以加入更多對 DTO 內容的檢查
					assert.Equal(t, tc.expectedFirstStatus, responseData[0].Status, "First item status mismatch")
					// 檢查日期格式是否正確 (例如，檢查是否符合 YYYY-MM-DD)
					_, dateErr := time.Parse("2006-01-02", responseData[0].StartDate)
					assert.NoError(t, dateErr, "StartDate format should be YYYY-MM-DD")
				}
			} else {
				assert.Nil(t, actualResponse.Data, "Response data should be nil for failure case")
			}
		})
	}
}
