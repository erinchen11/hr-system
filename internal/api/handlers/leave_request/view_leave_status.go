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
	"github.com/google/uuid"
)

// ViewLeaveStatusHandler 包含依賴 (結構體名保持不變)
type ViewLeaveStatusHandler struct {
	leaveRequestSvc interfaces.LeaveRequestService
}

// NewViewLeaveStatusHandler 構造函數 (保持不變)
func NewViewLeaveStatusHandler(leaveRequestSvc interfaces.LeaveRequestService) *ViewLeaveStatusHandler {
	return &ViewLeaveStatusHandler{leaveRequestSvc: leaveRequestSvc}
}

// LeaveRequestStatusDTO 定義返回給客戶端的請假記錄資料結構
// 只包含必要欄位，避免洩漏過多內部細節
type LeaveRequestStatusDTO struct {
	ID          uuid.UUID  `json:"id"`
	LeaveType   string     `json:"leave_type"`
	StartDate   string     `json:"start_date"` // 返回 YYYY-MM-DD 格式字串
	EndDate     string     `json:"end_date"`   // 返回 YYYY-MM-DD 格式字串
	Reason      string     `json:"reason,omitempty"`
	Status      string     `json:"status"`
	RequestedAt time.Time  `json:"requested_at"`
	ApprovedAt  *time.Time `json:"approved_at,omitempty"`
	// 不返回 ApproverID 或完整的 Approver/Account 資訊
}

// ViewLeaveStatus 方法處理員工查看自己請假狀態的 HTTP 請求
func (h *ViewLeaveStatusHandler) ViewLeaveStatus(c *gin.Context) {
	// 1. 授權檢查: 確保是普通員工 (Role 2)
	claimsRaw, exists := c.Get("claims")
	if !exists {
		log.Println("ViewLeaveStatus: Claims not found in context")
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
	if claims.Role != models.RoleEmployee {
		c.AbortWithStatusJSON(http.StatusForbidden, common.Response{Code: http.StatusForbidden, Message: "Permission denied: Only employees can view their leave status"})
		return
	}
	// 從 Context 獲取員工本人的 Account ID String

	accountIDStr := claims.UserID

	// 2. 調用 Service 層獲取該員工的請假列表
	// *** 調用 ListAccountRequests ***
	leaveRequests, err := h.leaveRequestSvc.ListAccountRequests(c.Request.Context(), accountIDStr)

	// 3. 處理 Service 層返回的錯誤
	if err != nil {
		// *** 處理 AccountNotFound 錯誤 ***
		switch {
		case errors.Is(err, services.ErrAccountNotFound):
			log.Printf("Account %s not found when fetching leave requests", accountIDStr)
			c.JSON(http.StatusNotFound, common.Response{Code: http.StatusNotFound, Message: "User account not found"})
		default:
			log.Printf("Error fetching leave status for user %s via service: %v", accountIDStr, err)
			c.JSON(http.StatusInternalServerError, common.Response{
				Code:    http.StatusInternalServerError,
				Message: "Failed to retrieve leave records",
			})
		}
		return // *** 確保處理錯誤後返回 ***
	}

	// 4. 將 Service 返回的 []models.LeaveRequest 轉換為 []LeaveRequestStatusDTO
	responseDTOs := make([]LeaveRequestStatusDTO, 0, len(leaveRequests))
	for _, req := range leaveRequests {
		dto := LeaveRequestStatusDTO{
			ID:          req.ID,
			LeaveType:   req.LeaveType,
			StartDate:   req.StartDate.Format("2006-01-02"), // 格式化日期
			EndDate:     req.EndDate.Format("2006-01-02"),   // 格式化日期
			Reason:      req.Reason,
			Status:      req.Status,
			RequestedAt: req.RequestedAt,
			ApprovedAt:  req.ApprovedAt,
		}
		responseDTOs = append(responseDTOs, dto)
	}

	// 5. 返回成功響應
	c.JSON(http.StatusOK, common.Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    responseDTOs, // ***  返回 DTO slice ***
	})
}
