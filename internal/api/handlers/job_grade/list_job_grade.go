package handlers

import (
	"log"
	"net/http"

	"github.com/erinchen11/hr-system/internal/interfaces"
	"github.com/erinchen11/hr-system/internal/models"
	common "github.com/erinchen11/hr-system/internal/models/common"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type ListJobGradesHandler struct {
	jobGradeSvc interfaces.JobGradeService
}

// NewListJobGradesHandler 構造函數
func NewListJobGradesHandler(jobGradeSvc interfaces.JobGradeService) *ListJobGradesHandler {
	return &ListJobGradesHandler{jobGradeSvc: jobGradeSvc}
}

// JobGradeDTO 定義返回給客戶端的職等資料結構
type JobGradeDTO struct {
	Code        string          `json:"code"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	MinSalary   decimal.Decimal `json:"min_salary"`
	MaxSalary   decimal.Decimal `json:"max_salary"`
}

// ListJobGrades 方法處理獲取所有職等列表的 HTTP 請求
func (h *ListJobGradesHandler) ListJobGrades(c *gin.Context) {
	// 1. 授權檢查: 確保是 HR 或 Super User
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

	// 檢查角色權限
	if claims.Role != models.RoleHR && claims.Role != models.RoleSuperAdmin {
		c.AbortWithStatusJSON(http.StatusForbidden, common.Response{Code: http.StatusForbidden, Message: "Permission denied: Only HR or Super Admin can list job grades"})
		return
	}

	// 2. 調用 Service 層獲取所有職等
	jobGrades, err := h.jobGradeSvc.ListJobGrades(c.Request.Context())
	if err != nil {
		// 處理 Service 層返回的錯誤
		log.Printf("Error fetching job grades via service: %v", err)
		c.JSON(http.StatusInternalServerError, common.Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to retrieve job grades",
		})
		return
	}

	// 3. 將 []models.JobGrade 轉換為 []JobGradeDTO
	responseDTOs := make([]JobGradeDTO, 0, len(jobGrades))
	for _, grade := range jobGrades {
		dto := JobGradeDTO{
			Code:        grade.Code,
			Name:        grade.Name,
			Description: grade.Description,
			MinSalary:   grade.MinSalary, 
			MaxSalary:   grade.MaxSalary, 
		}
		responseDTOs = append(responseDTOs, dto)
	}

	// 4. 返回成功響應
	c.JSON(http.StatusOK, common.Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    responseDTOs,
	})
}
