package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/erinchen11/hr-system/internal/interfaces"
	"github.com/erinchen11/hr-system/internal/models" 
	common "github.com/erinchen11/hr-system/internal/models/common"
	"github.com/erinchen11/hr-system/internal/services" 
	"github.com/gin-gonic/gin"
)

// RejectLeaveRequestHandler 包含依賴
type RejectLeaveRequestHandler struct {
	leaveRequestSvc interfaces.LeaveRequestService
}

// NewRejectLeaveRequestHandler 構造函數
func NewRejectLeaveRequestHandler(leaveRequestSvc interfaces.LeaveRequestService) *RejectLeaveRequestHandler {
	return &RejectLeaveRequestHandler{leaveRequestSvc: leaveRequestSvc}
}

// RejectLeaveRequestRequest 請求體
type RejectLeaveRequestRequest struct {
	Reason string `json:"reason"`
}

func (h *RejectLeaveRequestHandler) RejectLeaveRequest(c *gin.Context) {
	// 1. 授權檢查
	claimsRaw, exists := c.Get("claims")
	// *** 確保處理 claims 不存在的情況 ***
	if !exists {
		log.Println("RejectLeaveRequest: Claims not found in context") 
		c.AbortWithStatusJSON(http.StatusUnauthorized, common.Response{
			Code:    http.StatusUnauthorized,
			Message: "Unauthorized: Missing user claims", 
		})
		return // <-- 必須 return
	}

	// ***  點：確保處理 claims 類型錯誤或為 nil 的情況 ***
	claims, ok := claimsRaw.(*models.Claims) // 假設 Claims 在 models 包
	if !ok || claims == nil {
		log.Printf("Error: Invalid claims type in context: %T", claimsRaw)
		c.AbortWithStatusJSON(http.StatusInternalServerError, common.Response{
			Code:    http.StatusInternalServerError,
			Message: "Internal error processing user identity",
		})
		return // <-- 必須 return
	}

	// 檢查角色是否為 HR (Role 1)
	if claims.Role != 1 {
		c.AbortWithStatusJSON(http.StatusForbidden, common.Response{Code: http.StatusForbidden, Message: "Permission denied: Only HR can reject leave requests"})
		return
	}
	// 從 Context 獲取拒絕人 (HR) 的 ID
	rejectorIDStr := claims.UserID

	// 2. 從 URL 路徑參數獲取要拒絕的請假單 ID
	leaveRequestIDStr := c.Param("id")
	if leaveRequestIDStr == "" {
		c.JSON(http.StatusBadRequest, common.Response{Code: http.StatusBadRequest, Message: "Missing leave request ID in URL path"})
		return
	}

	// 3. 解析請求體獲取拒絕理由 (可選)
	var req RejectLeaveRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		c.JSON(http.StatusBadRequest, common.Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid request body format: " + err.Error(),
		})
		return
	}

	// 4. 調用 Service 層處理拒絕邏輯
	err := h.leaveRequestSvc.RejectRequest(c.Request.Context(), leaveRequestIDStr, rejectorIDStr, req.Reason)

	// 5. 處理 Service 層返回的錯誤 (與 Approve 類似)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrLeaveRequestNotFound):
			c.JSON(http.StatusNotFound, common.Response{Code: http.StatusNotFound, Message: "Leave request not found"})
		case errors.Is(err, services.ErrInvalidLeaveRequestState):
			c.JSON(http.StatusBadRequest, common.Response{Code: http.StatusBadRequest, Message: "Leave request cannot be rejected (state is not pending)"})
		case errors.Is(err, services.ErrLeaveRequestUpdateFailed):
			log.Printf("Internal error rejecting leave request %s: %v", leaveRequestIDStr, err)
			c.JSON(http.StatusInternalServerError, common.Response{Code: http.StatusInternalServerError, Message: "Failed to update leave request status"})
		default:
			log.Printf("Unexpected error rejecting leave request %s: %v", leaveRequestIDStr, err)
			c.JSON(http.StatusInternalServerError, common.Response{Code: http.StatusInternalServerError, Message: "An unexpected error occurred"})
		}
		return
	}

	// 6. 返回成功響應
	c.JSON(http.StatusOK, common.Response{
		Code:    http.StatusOK,
		Message: "Leave request rejected successfully",
		Data:    nil,
	})
}
