// 檔案路徑: internal/api/handlers/apply_leave_handler_test.go
package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/erinchen11/hr-system/internal/interfaces/mocks"
	"github.com/erinchen11/hr-system/internal/models"
	common "github.com/erinchen11/hr-system/internal/models/common"
	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyLeaveHandler_ApplyLeave(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// --- 通用模擬資料 ---
	employeeID := uuid.New()
	employeeIDStr := employeeID.String()
	employeeClaims := &models.Claims{UserID: employeeIDStr, Role: models.RoleEmployee, Email: "emp@test.com"}
	hrClaims := &models.Claims{UserID: uuid.New().String(), Role: models.RoleHR, Email: "hr@test.com"}

	testLeaveType := "personal"
	testStartDateStr := "2025-07-10"
	testEndDateStr := "2025-07-11"
	testReason := "Family emergency"
	// --- 合法的請假申請 JSON ---
	validRequestBody := fmt.Sprintf(`{"start_date": "%s", "end_date": "%s", "leave_type": "%s", "reason": "%s"}`, testStartDateStr, testEndDateStr, testLeaveType, testReason)
	validRequestBodyNoReason := fmt.Sprintf(`{"start_date": "%s", "end_date": "%s", "leave_type": "%s"}`, testStartDateStr, testEndDateStr, testLeaveType)

	testCases := []struct {
		name                 string
		callerClaims         interface{}
		requestBody          string
		setupMocks           func(mockLeaveSvc *mocks.MockLeaveRequestService)
		expectedStatusCode   int
		expectedResponseCode int
		expectedMessage      string
	}{
		{
			name:         "Success - Apply with reason",
			callerClaims: employeeClaims,
			requestBody:  validRequestBody,
			setupMocks: func(mockLeaveSvc *mocks.MockLeaveRequestService) {
				mockLeaveSvc.EXPECT().ApplyForLeave(gomock.Any(), employeeIDStr, testLeaveType, testReason, gomock.Any(), gomock.Any()).Return(&models.LeaveRequest{ID: uuid.New()}, nil)
			},
			expectedStatusCode:   http.StatusCreated,
			expectedResponseCode: http.StatusCreated,
			expectedMessage:      "Leave application submitted successfully",
		},
		{
			name:         "Success - Apply without reason",
			callerClaims: employeeClaims,
			requestBody:  validRequestBodyNoReason,
			setupMocks: func(mockLeaveSvc *mocks.MockLeaveRequestService) {
				mockLeaveSvc.EXPECT().ApplyForLeave(gomock.Any(), employeeIDStr, testLeaveType, "", gomock.Any(), gomock.Any()).Return(&models.LeaveRequest{ID: uuid.New()}, nil)
			},
			expectedStatusCode:   http.StatusCreated,
			expectedResponseCode: http.StatusCreated,
			expectedMessage:      "Leave application submitted successfully",
		},
		{
			name:                 "Forbidden - HR tries to apply",
			callerClaims:         hrClaims,
			requestBody:          validRequestBody,
			expectedStatusCode:   http.StatusForbidden,
			expectedResponseCode: http.StatusForbidden,
			expectedMessage:      "Permission denied: Only employees can apply for leave",
		},
		{
			name:                 "Unauthorized - Missing Claims",
			callerClaims:         nil,
			requestBody:          validRequestBody,
			expectedStatusCode:   http.StatusUnauthorized,
			expectedResponseCode: http.StatusUnauthorized,
			expectedMessage:      "Unauthorized: Missing user claims",
		},
		{
			name:                 "Internal Server Error - Invalid Claims Type",
			callerClaims:         &struct{ UserID string }{UserID: "fake"},
			requestBody:          validRequestBody,
			expectedStatusCode:   http.StatusInternalServerError,
			expectedResponseCode: http.StatusInternalServerError,
			expectedMessage:      "Internal error processing user identity",
		},
		{
			name:                 "Bad Request - Invalid JSON",
			callerClaims:         employeeClaims,
			requestBody:          `{"start_date": "bad", "end_date": "json", "leave_type": "sick",}`,
			expectedStatusCode:   http.StatusBadRequest,
			expectedResponseCode: http.StatusBadRequest,
			expectedMessage:      "Invalid request format",
		},
		{
			name:                 "Bad Request - Missing Leave Type",
			callerClaims:         employeeClaims,
			requestBody:          `{"start_date": "2025-07-10", "end_date": "2025-07-11"}`,
			expectedStatusCode:   http.StatusBadRequest,
			expectedResponseCode: http.StatusBadRequest,
			expectedMessage:      "Invalid request format",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLeaveSvc := mocks.NewMockLeaveRequestService(ctrl)
			handler := NewApplyLeaveHandler(mockLeaveSvc)

			if tc.setupMocks != nil {
				tc.setupMocks(mockLeaveSvc)
			}

			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/employee/apply-leave", bytes.NewBufferString(tc.requestBody))
			req.Header.Set("Content-Type", "application/json")
			c.Request = req

			if tc.callerClaims != nil {
				c.Set("claims", tc.callerClaims)
			}

			handler.ApplyLeave(c)

			require.Equal(t, tc.expectedStatusCode, recorder.Code)

			var resp common.Response
			err := json.Unmarshal(recorder.Body.Bytes(), &resp)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedResponseCode, resp.Code)
			assert.Contains(t, resp.Message, tc.expectedMessage)
		})
	}
}
