package handlers

import (
	"errors" 
	"log"
	"net/http"
	"time" 

	"github.com/erinchen11/hr-system/internal/interfaces"
	"github.com/erinchen11/hr-system/internal/models" 
	common "github.com/erinchen11/hr-system/internal/models/common"
	"github.com/erinchen11/hr-system/internal/services" 
	"github.com/gin-gonic/gin"
	
)

// ApplyLeaveHandler 包含依賴
type ApplyLeaveHandler struct {
	leaveRequestSvc interfaces.LeaveRequestService 
}

// NewApplyLeaveHandler 構造函數
func NewApplyLeaveHandler(leaveRequestSvc interfaces.LeaveRequestService) *ApplyLeaveHandler {
	return &ApplyLeaveHandler{leaveRequestSvc: leaveRequestSvc}
}

// ApplyLeaveRequest 請求體結構
type ApplyLeaveRequest struct {
	StartDate string `json:"start_date" binding:"required,datetime=2006-01-02"` 
	EndDate   string `json:"end_date" binding:"required,datetime=2006-01-02"`
	LeaveType string `json:"leave_type" binding:"required"`
	Reason    string `json:"reason"` // 允許 reason 為空
}

// ApplyLeave 方法處理員工提交請假申請的 HTTP 請求
func (h *ApplyLeaveHandler) ApplyLeave(c *gin.Context) {
	// 1. 授權檢查: 確保是普通員工 (Role 2)
	claimsRaw, exists := c.Get("claims")
	if !exists {
		log.Println("ApplyLeave: Claims not found in context")
		c.AbortWithStatusJSON(http.StatusUnauthorized, common.Response{Code: http.StatusUnauthorized, Message: "Unauthorized: Missing user claims"})
		return
	}
	claims, ok := claimsRaw.(*models.Claims)
	if !ok || claims == nil {
		log.Printf("Error: Invalid claims type in context: %T", claimsRaw)
		c.AbortWithStatusJSON(http.StatusInternalServerError, common.Response{Code: http.StatusInternalServerError, Message: "Internal error processing user identity"})
		return
	}

	// 檢查角色是否為 Employee (Role 2)
	// 注意：如果 HR 或 SuperAdmin 也被允許 "替自己" 請假，這裡的邏輯需要調整
	if claims.Role != models.RoleEmployee {
		c.AbortWithStatusJSON(http.StatusForbidden, common.Response{Code: http.StatusForbidden, Message: "Permission denied: Only employees can apply for leave"})
		return
	}
	// 從 Context 獲取申請人 (員工) 的 Account ID String
	accountIDStr := claims.UserID

	// 2. 解析請求體
	var req ApplyLeaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid request format: " + err.Error(), 
		})
		return
	}

	// 3. 將日期字串轉換為 time.Time
	startDate, _ := time.Parse("2006-01-02", req.StartDate)
	endDate, _ := time.Parse("2006-01-02", req.EndDate)

	// 4. 調用 Service 層處理請假申請邏輯
	_, err := h.leaveRequestSvc.ApplyForLeave(
		c.Request.Context(),
		accountIDStr,
		req.LeaveType,
		req.Reason,
		startDate,
		endDate,
	)

	// 5. 處理 Service 層返回的錯誤
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidDateRange):
			c.JSON(http.StatusBadRequest, common.Response{Code: http.StatusBadRequest, Message: err.Error()})
		// *** 新增: 處理帳戶未找到的錯誤 ***
		case errors.Is(err, services.ErrAccountNotFound):
			// 這表示 context 中的 UserID 無效或對應的帳戶不存在
			log.Printf("Error applying leave: Applicant account %s not found: %v", accountIDStr, err)
			// 可以返回 404 或 400，這裡用 404
			c.JSON(http.StatusNotFound, common.Response{Code: http.StatusNotFound, Message: "Applicant account not found"})
		case errors.Is(err, services.ErrLeaveApplyFailed):
			log.Printf("Internal error applying leave for user %s: %v", accountIDStr, err)
			c.JSON(http.StatusInternalServerError, common.Response{Code: http.StatusInternalServerError, Message: "Failed to submit leave application"})
		default:
			// 處理其他可能的 Service 層錯誤或 Repository 層錯誤
			log.Printf("Unexpected error applying leave for user %s: %v", accountIDStr, err)
			c.JSON(http.StatusInternalServerError, common.Response{Code: http.StatusInternalServerError, Message: "An unexpected error occurred"})
		}
		return
	}

	c.JSON(http.StatusCreated, common.Response{
		Code:    http.StatusCreated,
		Message: "Leave application submitted successfully",
	})
}
