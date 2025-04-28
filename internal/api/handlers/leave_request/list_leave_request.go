package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/erinchen11/hr-system/internal/interfaces" 
	common "github.com/erinchen11/hr-system/internal/models/common"

	"github.com/erinchen11/hr-system/internal/models" 
	"github.com/gin-gonic/gin"
)

// ListLeaveRequestsHandler 包含依賴
type ListLeaveRequestsHandler struct {
	leaveRequestSvc interfaces.LeaveRequestService 
}

// NewListLeaveRequestsHandler 構造函數
func NewListLeaveRequestsHandler(leaveRequestSvc interfaces.LeaveRequestService) *ListLeaveRequestsHandler {
	return &ListLeaveRequestsHandler{leaveRequestSvc: leaveRequestSvc}
}

// LeaveRequestResponse 是對前端乾淨的回傳結構
type LeaveRequestResponse struct {
	Id          string     `json:"id"`
	LeaveType   string     `json:"leave_type"`
	StartDate   time.Time  `json:"start_date"`
	EndDate     time.Time  `json:"end_date"`
	Reason      string     `json:"reason,omitempty"`
	Status      string     `json:"status"`
	RequestedAt time.Time  `json:"requested_at"`
	ApprovedAt  *time.Time `json:"approved_at,omitempty"`

	Applicant struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Email     string `json:"email"`
	} `json:"applicant"`

	Approver *struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Email     string `json:"email"`
	} `json:"approver,omitempty"`
}

func (h *ListLeaveRequestsHandler) ListLeaveRequests(c *gin.Context) {
	claimsRaw, exists := c.Get("claims")
	if !exists {
		c.AbortWithStatusJSON(http.StatusUnauthorized, common.Response{Code: http.StatusUnauthorized, Message: "Unauthorized: Missing user claims"})
		return
	}
	claims, ok := claimsRaw.(*models.Claims)
	if !ok || claims == nil {
		log.Printf("Error: Invalid claims type in context: %T", claimsRaw)
		c.AbortWithStatusJSON(http.StatusInternalServerError, common.Response{Code: http.StatusInternalServerError, Message: "Internal error processing user identity"})
		return
	}
	if claims.Role != 1 {
		c.AbortWithStatusJSON(http.StatusForbidden, common.Response{Code: http.StatusForbidden, Message: "Permission denied: Only HR can view leave requests"})
		return
	}

	// 調用 Service
	requests, err := h.leaveRequestSvc.ListAllRequests(c.Request.Context())
	if err != nil {
		log.Printf("Error fetching leave requests via service: %v", err)
		c.JSON(http.StatusInternalServerError, common.Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to fetch leave requests",
			Data:    nil,
		})
		return
	}

	// --- 將原始模型轉換成乾淨的回傳結構 ---
	response := make([]LeaveRequestResponse, 0, len(requests))
	for _, r := range requests {
		item := LeaveRequestResponse{
			Id:          r.ID.String(),
			LeaveType:   r.LeaveType,
			StartDate:   r.StartDate,
			EndDate:     r.EndDate,
			Reason:      r.Reason,
			Status:      r.Status,
			RequestedAt: r.RequestedAt,
			ApprovedAt:  r.ApprovedAt,
			Applicant: struct {
				FirstName string `json:"first_name"`
				LastName  string `json:"last_name"`
				Email     string `json:"email"`
			}{
				FirstName: r.Account.FirstName,
				LastName:  r.Account.LastName,
				Email:     r.Account.Email,
			},
		}

		if r.Approver != nil {
			item.Approver = &struct {
				FirstName string `json:"first_name"`
				LastName  string `json:"last_name"`
				Email     string `json:"email"`
			}{
				FirstName: r.Approver.FirstName,
				LastName:  r.Approver.LastName,
				Email:     r.Approver.Email,
			}
		}

		response = append(response, item)
	}

	// --- 成功回傳 ---
	c.JSON(http.StatusOK, common.Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    response,
	})
}
