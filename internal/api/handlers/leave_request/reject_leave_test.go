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
	"github.com/stretchr/testify/require"
)

func TestRejectLeaveRequestHandler_RejectLeaveRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 準備 Claims 數據
	hrUserID := uuid.New().String()
	hrClaims := &models.Claims{UserID: hrUserID, Email: "hr@example.com", Role: 1}                   // HR
	employeeClaims := &models.Claims{UserID: uuid.New().String(), Email: "emp@example.com", Role: 2} // Employee

	// 準備測試用的 Leave Request ID
	testLeaveID := uuid.New().String()
	testReason := "Project deadline approaching"

	testCases := []struct {
		name             string
		claimsToSet      interface{}
		leaveIDParam     string
		requestBody      string // JSON 請求體字串
		setupMocks       func(leaveSvc *mocks.MockLeaveRequestService)
		expectedStatus   int
		expectedResponse common.Response
	}{
		{
			name:         "Success - Reject without Reason",
			claimsToSet:  hrClaims,
			leaveIDParam: testLeaveID,
			requestBody:  `{}`, // 可以是空 JSON 或完全沒有 Body
			setupMocks: func(leaveSvc *mocks.MockLeaveRequestService) {
				// 期望 RejectRequest 被調用，Reason 為空字串
				leaveSvc.EXPECT().RejectRequest(gomock.Any(), testLeaveID, hrUserID, "").Return(nil).Times(1)
			},
			expectedStatus:   http.StatusOK,
			expectedResponse: common.Response{Code: http.StatusOK, Message: "Leave request rejected successfully", Data: nil},
		},
		{
			name:         "Success - Reject with Reason",
			claimsToSet:  hrClaims,
			leaveIDParam: testLeaveID,
			requestBody:  `{"reason": "` + testReason + `"}`, // 包含 Reason 的 JSON
			setupMocks: func(leaveSvc *mocks.MockLeaveRequestService) {
				// 期望 RejectRequest 被調用，Reason 為 testReason
				leaveSvc.EXPECT().RejectRequest(gomock.Any(), testLeaveID, hrUserID, testReason).Return(nil).Times(1)
			},
			expectedStatus:   http.StatusOK,
			expectedResponse: common.Response{Code: http.StatusOK, Message: "Leave request rejected successfully", Data: nil},
		},
		{
			name:             "Forbidden - Employee Tries to Reject",
			claimsToSet:      employeeClaims,
			leaveIDParam:     testLeaveID,
			requestBody:      `{}`,
			setupMocks:       nil,
			expectedStatus:   http.StatusForbidden,
			expectedResponse: common.Response{Code: http.StatusForbidden, Message: "Permission denied: Only HR can reject leave requests"},
		},
		{
			name:           "Unauthorized - Missing Claims",
			claimsToSet:    nil,
			leaveIDParam:   testLeaveID,
			requestBody:    `{}`,
			setupMocks:     nil,
			expectedStatus: http.StatusUnauthorized,
			expectedResponse: common.Response{
				Code:    http.StatusUnauthorized,
				Message: "Unauthorized: Missing user claims",
				Data:    nil,
			},
		},
		{
			name:             "Bad Request - Missing Leave ID Param",
			claimsToSet:      hrClaims,
			leaveIDParam:     "",
			requestBody:      `{}`,
			setupMocks:       nil,
			expectedStatus:   http.StatusBadRequest,
			expectedResponse: common.Response{Code: http.StatusBadRequest, Message: "Missing leave request ID in URL path"},
		},
		{
			name:             "Bad Request - Invalid JSON Body",
			claimsToSet:      hrClaims,
			leaveIDParam:     testLeaveID,
			requestBody:      `{"reason":`, // 無效 JSON
			setupMocks:       nil,
			expectedStatus:   http.StatusBadRequest,
			expectedResponse: common.Response{Code: http.StatusBadRequest, Message: "Invalid request body format"},
		},
		{
			name:         "Not Found - Service Returns Not Found Error",
			claimsToSet:  hrClaims,
			leaveIDParam: testLeaveID,
			requestBody:  `{}`,
			setupMocks: func(leaveSvc *mocks.MockLeaveRequestService) {
				leaveSvc.EXPECT().RejectRequest(gomock.Any(), testLeaveID, hrUserID, "").Return(services.ErrLeaveRequestNotFound).Times(1)
			},
			expectedStatus:   http.StatusNotFound,
			expectedResponse: common.Response{Code: http.StatusNotFound, Message: "Leave request not found"},
		},
		{
			name:         "Bad Request - Service Returns Invalid State Error",
			claimsToSet:  hrClaims,
			leaveIDParam: testLeaveID,
			requestBody:  `{}`,
			setupMocks: func(leaveSvc *mocks.MockLeaveRequestService) {
				leaveSvc.EXPECT().RejectRequest(gomock.Any(), testLeaveID, hrUserID, "").Return(services.ErrInvalidLeaveRequestState).Times(1)
			},
			expectedStatus:   http.StatusBadRequest,
			expectedResponse: common.Response{Code: http.StatusBadRequest, Message: "Leave request cannot be rejected (state is not pending)"},
		},
		{
			name:         "Internal Server Error - Service Update Failed",
			claimsToSet:  hrClaims,
			leaveIDParam: testLeaveID,
			requestBody:  `{}`,
			setupMocks: func(leaveSvc *mocks.MockLeaveRequestService) {
				leaveSvc.EXPECT().RejectRequest(gomock.Any(), testLeaveID, hrUserID, "").Return(services.ErrLeaveRequestUpdateFailed).Times(1)
			},
			expectedStatus:   http.StatusInternalServerError,
			expectedResponse: common.Response{Code: http.StatusInternalServerError, Message: "Failed to update leave request status"},
		},
		{
			name:         "Internal Server Error - Unexpected Service Error",
			claimsToSet:  hrClaims,
			leaveIDParam: testLeaveID,
			requestBody:  `{}`,
			setupMocks: func(leaveSvc *mocks.MockLeaveRequestService) {
				unexpectedError := errors.New("some random error")
				leaveSvc.EXPECT().RejectRequest(gomock.Any(), testLeaveID, hrUserID, "").Return(unexpectedError).Times(1)
			},
			expectedStatus:   http.StatusInternalServerError,
			expectedResponse: common.Response{Code: http.StatusInternalServerError, Message: "An unexpected error occurred"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			// defer ctrl.Finish()

			mockLeaveReqSvc := mocks.NewMockLeaveRequestService(ctrl) // Mock Service

			// 創建 Handler
			rejectHandler := NewRejectLeaveRequestHandler(mockLeaveReqSvc)

			// 設置 Mock 預期
			if tc.setupMocks != nil {
				tc.setupMocks(mockLeaveReqSvc)
			}

			// 模擬 HTTP 環境
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)

			// 模擬請求 (方法 POST，URL 包含參數，Body 根據測試案例)
			req, _ := http.NewRequest(http.MethodPost, "/hr/leave-requests/"+tc.leaveIDParam+"/reject", bytes.NewBufferString(tc.requestBody))
			
			req.Header.Set("Content-Type", "application/json") // 即使 Body 為空，也模擬 Content-Type
			c.Request = req

			// 模擬 Context
			if tc.claimsToSet != nil {
				c.Set("claims", tc.claimsToSet)
				if claims, ok := tc.claimsToSet.(*models.Claims); ok {
					c.Set("user_id", claims.UserID)
				}
			}

			// 模擬 URL 參數
			if tc.leaveIDParam != "" {
				c.Params = gin.Params{gin.Param{Key: "id", Value: tc.leaveIDParam}}
			} else {
				c.Params = gin.Params{}
			}

			// 調用 Handler 方法
			rejectHandler.RejectLeaveRequest(c)

			// 斷言
			assert.Equal(t, tc.expectedStatus, recorder.Code)

			var actualResponse common.Response
			err := json.Unmarshal(recorder.Body.Bytes(), &actualResponse)
			require.NoError(t, err, "Response body should be valid JSON")

			// 比較 Code 和 Message
			assert.Equal(t, tc.expectedResponse.Code, actualResponse.Code, "Response code mismatch")
			// 對於 Bad Request (Invalid JSON)，錯誤消息可能包含細節，需要調整比較方式
			if tc.expectedStatus == http.StatusBadRequest && tc.name == "Bad Request - Invalid JSON Body" {
				assert.Contains(t, actualResponse.Message, tc.expectedResponse.Message, "Response message mismatch for Bad Request (JSON)")
			} else {
				assert.Equal(t, tc.expectedResponse.Message, actualResponse.Message, "Response message mismatch")
			}
			assert.Nil(t, actualResponse.Data) // 這個 API 通常不返回 Data
		})
	}
}

