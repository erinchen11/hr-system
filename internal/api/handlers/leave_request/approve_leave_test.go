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
	"github.com/erinchen11/hr-system/internal/services" 
	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApproveLeaveRequestHandler_ApproveLeaveRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 準備 Claims 數據
	hrUserID := uuid.New().String()
	hrClaims := &models.Claims{UserID: hrUserID, Email: "hr@example.com", Role: 1}                   // HR
	employeeClaims := &models.Claims{UserID: uuid.New().String(), Email: "emp@example.com", Role: 2} // Employee

	// 準備測試用的 Leave Request ID
	testLeaveID := uuid.New().String()

	testCases := []struct {
		name             string
		claimsToSet      interface{} // 模擬 Context 中的 claims
		leaveIDParam     string      // 模擬 URL 中的 id 參數
		setupMocks       func(leaveSvc *mocks.MockLeaveRequestService)
		expectedStatus   int
		expectedResponse common.Response
	}{
		{
			name:         "Success - HR Approves Request",
			claimsToSet:  hrClaims,
			leaveIDParam: testLeaveID,
			setupMocks: func(leaveSvc *mocks.MockLeaveRequestService) {
				// 期望 Service 的 ApproveRequest 被正確調用且成功
				leaveSvc.EXPECT().ApproveRequest(gomock.Any(), testLeaveID, hrUserID).Return(nil).Times(1)
			},
			expectedStatus:   http.StatusOK,
			expectedResponse: common.Response{Code: http.StatusOK, Message: "Leave request approved successfully", Data: nil},
		},
		{
			name:             "Forbidden - Employee Tries to Approve",
			claimsToSet:      employeeClaims, // 使用 Employee 的 Claims
			leaveIDParam:     testLeaveID,
			setupMocks:       nil, // Service 不應被調用
			expectedStatus:   http.StatusForbidden,
			expectedResponse: common.Response{Code: http.StatusForbidden, Message: "Permission denied: Only HR can approve leave requests"},
		},
		{
			name:             "Unauthorized - Missing Claims",
			claimsToSet:      nil,
			leaveIDParam:     testLeaveID,
			setupMocks:       nil,
			expectedStatus:   http.StatusUnauthorized,
			expectedResponse: common.Response{Code: http.StatusUnauthorized, Message: "Unauthorized: Missing claims"},
		},
		{
			name:             "Bad Request - Missing Leave ID Param",
			claimsToSet:      hrClaims,
			leaveIDParam:     "",  // 模擬 URL 中缺少 ID 或為空
			setupMocks:       nil, // Service 不應被調用
			expectedStatus:   http.StatusBadRequest,
			expectedResponse: common.Response{Code: http.StatusBadRequest, Message: "Missing leave request ID in URL path"},
		},
		{
			name:         "Not Found - Service Returns Not Found Error",
			claimsToSet:  hrClaims,
			leaveIDParam: testLeaveID,
			setupMocks: func(leaveSvc *mocks.MockLeaveRequestService) {
				// 期望 Service 返回 ErrLeaveRequestNotFound
				leaveSvc.EXPECT().ApproveRequest(gomock.Any(), testLeaveID, hrUserID).Return(services.ErrLeaveRequestNotFound).Times(1)
			},
			expectedStatus:   http.StatusNotFound,
			expectedResponse: common.Response{Code: http.StatusNotFound, Message: "Leave request not found"},
		},
		{
			name:         "Bad Request - Service Returns Invalid State Error",
			claimsToSet:  hrClaims,
			leaveIDParam: testLeaveID,
			setupMocks: func(leaveSvc *mocks.MockLeaveRequestService) {
				// 期望 Service 返回 ErrInvalidLeaveRequestState
				leaveSvc.EXPECT().ApproveRequest(gomock.Any(), testLeaveID, hrUserID).Return(services.ErrInvalidLeaveRequestState).Times(1)
			},
			expectedStatus:   http.StatusBadRequest,
			expectedResponse: common.Response{Code: http.StatusBadRequest, Message: "Leave request cannot be approved (state is not pending)"},
		},
		{
			name:         "Internal Server Error - Service Update Failed",
			claimsToSet:  hrClaims,
			leaveIDParam: testLeaveID,
			setupMocks: func(leaveSvc *mocks.MockLeaveRequestService) {
				// 期望 Service 返回 ErrLeaveRequestUpdateFailed
				leaveSvc.EXPECT().ApproveRequest(gomock.Any(), testLeaveID, hrUserID).Return(services.ErrLeaveRequestUpdateFailed).Times(1)
			},
			expectedStatus:   http.StatusInternalServerError,
			expectedResponse: common.Response{Code: http.StatusInternalServerError, Message: "Failed to update leave request status"},
		},
		{
			name:         "Internal Server Error - Unexpected Service Error",
			claimsToSet:  hrClaims,
			leaveIDParam: testLeaveID,
			setupMocks: func(leaveSvc *mocks.MockLeaveRequestService) {
				// 期望 Service 返回一個通用錯誤
				unexpectedError := errors.New("some unexpected service error")
				leaveSvc.EXPECT().ApproveRequest(gomock.Any(), testLeaveID, hrUserID).Return(unexpectedError).Times(1)
			},
			expectedStatus:   http.StatusInternalServerError,
			expectedResponse: common.Response{Code: http.StatusInternalServerError, Message: "An unexpected error occurred"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			// defer ctrl.Finish()

			mockLeaveReqSvc := mocks.NewMockLeaveRequestService(ctrl) // <-- Mock Service

			// 創建 Handler
			approveHandler := NewApproveLeaveRequestHandler(mockLeaveReqSvc)

			// 設置 Mock 預期
			if tc.setupMocks != nil {
				tc.setupMocks(mockLeaveReqSvc)
			}

			// 模擬 HTTP 環境
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)

			// 模擬請求 (方法 POST，URL 包含參數，無 Body)
			// 注意: URL 需要匹配你在 router.go 中定義的實際路徑格式
			req, _ := http.NewRequest(http.MethodPost, "/hr/leave-requests/"+tc.leaveIDParam+"/approve", nil)
			c.Request = req

			// *** 模擬 Context (設置 claims 和 user_id) ***
			if tc.claimsToSet != nil {
				c.Set("claims", tc.claimsToSet)
				if claims, ok := tc.claimsToSet.(*models.Claims); ok {
					c.Set("user_id", claims.UserID) // <--- 確保 user_id 被設置
					// c.Set("email", claims.Email)
					// c.Set("role", claims.Role)
				}
			}

			// *** 模擬 URL 參數 ***
			if tc.leaveIDParam != "" {
				c.Params = gin.Params{gin.Param{Key: "id", Value: tc.leaveIDParam}}
			} else {
				c.Params = gin.Params{} // 確保在沒有 ID 的情況下 Params 是空的
			}

			// 調用 Handler 方法
			approveHandler.ApproveLeaveRequest(c)

			// 斷言
			assert.Equal(t, tc.expectedStatus, recorder.Code)

			var actualResponse common.Response
			err := json.Unmarshal(recorder.Body.Bytes(), &actualResponse)
			require.NoError(t, err, "Response body should be valid JSON")

			assert.Equal(t, tc.expectedResponse.Code, actualResponse.Code, "Response code mismatch")
			assert.Equal(t, tc.expectedResponse.Message, actualResponse.Message, "Response message mismatch")
			// 這個 API 通常不返回 Data，可以確認一下
			assert.Nil(t, actualResponse.Data)
		})
	}
}

