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

type ApproveLeaveRequestHandler struct {
	leaveRequestSvc interfaces.LeaveRequestService 
}

// NewApproveLeaveRequestHandler 構造函數
func NewApproveLeaveRequestHandler(leaveRequestSvc interfaces.LeaveRequestService) *ApproveLeaveRequestHandler {
	return &ApproveLeaveRequestHandler{leaveRequestSvc: leaveRequestSvc}
}

// ApproveLeaveRequest 方法處理批准請假的 HTTP 請求
func (h *ApproveLeaveRequestHandler) ApproveLeaveRequest(c *gin.Context) {
	claimsRaw, exists := c.Get("claims")
	if !exists {
		c.AbortWithStatusJSON(http.StatusUnauthorized, common.Response{Code: http.StatusUnauthorized, Message: "Unauthorized: Missing claims"})
		return
	}
	claims, ok := claimsRaw.(*models.Claims)
	if !ok || claims == nil {
		log.Printf("Error: Invalid claims type in context: %T", claimsRaw)
		c.AbortWithStatusJSON(http.StatusInternalServerError, common.Response{Code: http.StatusInternalServerError, Message: "Internal error processing user identity"})
		return
	}

	// 檢查角色是否為 HR (Role 1)
	if claims.Role != 1 {
		c.AbortWithStatusJSON(http.StatusForbidden, common.Response{Code: http.StatusForbidden, Message: "Permission denied: Only HR can approve leave requests"})
		return
	}
	// 從 Context 獲取批准人 (HR) 的 ID
	approverIDStr := claims.UserID // 直接使用 claims 中的 UserID

	// 2. 從 URL 路徑參數獲取要批准的請假單 ID
	leaveRequestIDStr := c.Param("id")
	if leaveRequestIDStr == "" {
		c.JSON(http.StatusBadRequest, common.Response{Code: http.StatusBadRequest, Message: "Missing leave request ID in URL path"})
		return
	}

	// 3. 調用 Service 層處理批准邏輯
	err := h.leaveRequestSvc.ApproveRequest(c.Request.Context(), leaveRequestIDStr, approverIDStr)

	// 4. 處理 Service 層返回的錯誤
	if err != nil {
		switch {
		case errors.Is(err, services.ErrLeaveRequestNotFound):
			c.JSON(http.StatusNotFound, common.Response{Code: http.StatusNotFound, Message: "Leave request not found"})
		case errors.Is(err, services.ErrInvalidLeaveRequestState):
			c.JSON(http.StatusBadRequest, common.Response{Code: http.StatusBadRequest, Message: "Leave request cannot be approved (state is not pending)"})
		case errors.Is(err, services.ErrLeaveRequestUpdateFailed):
			log.Printf("Internal error approving leave request %s: %v", leaveRequestIDStr, err)
			c.JSON(http.StatusInternalServerError, common.Response{Code: http.StatusInternalServerError, Message: "Failed to update leave request status"})
		default:
			log.Printf("Unexpected error approving leave request %s: %v", leaveRequestIDStr, err)
			c.JSON(http.StatusInternalServerError, common.Response{Code: http.StatusInternalServerError, Message: "An unexpected error occurred"})
		}
		return
	}

	// 5. 返回成功響應
	c.JSON(http.StatusOK, common.Response{
		Code:    http.StatusOK, // 成功用 200 OK
		Message: "Leave request approved successfully",
		Data:    nil, // 通常批准操作不需要返回數據體
	})
}
