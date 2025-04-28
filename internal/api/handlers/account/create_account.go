// internal/api/handlers/user_creation_handler.go
package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/erinchen11/hr-system/internal/interfaces"
	"github.com/erinchen11/hr-system/internal/models"
	common "github.com/erinchen11/hr-system/internal/models/common"
	"github.com/erinchen11/hr-system/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// AccountCreationHandler 負責處理帳號建立請求
type AccountCreationHandler struct {
	accountSvc interfaces.AccountService
}

// NewAccountCreationHandler 建構函數
func NewAccountCreationHandler(accountSvc interfaces.AccountService) *AccountCreationHandler {
	return &AccountCreationHandler{accountSvc: accountSvc}
}

// CreateUserRequest 建立使用者請求體
// 建立 HR 或 Employee 帳號，SuperAdmin 可以建立 HR/Employee，HR 只能建立 Employee
// 預設 role=2 (Employee)
type CreateUserRequest struct {
	FirstName     string  `json:"first_name" binding:"required"`
	LastName      string  `json:"last_name" binding:"required"`
	Email         string  `json:"email" binding:"required,email"`
	Role          uint8   `json:"role" binding:"omitempty,oneof=1 2" example:"2" default:"2"`
	JobGradeCode  string  `json:"job_grade_code,omitempty" binding:"omitempty"`
	PositionTitle string  `json:"position_title,omitempty"`
	Salary        *string `json:"salary,omitempty" binding:"omitempty,numeric"`
	HireDate      *string `json:"hire_date,omitempty" binding:"omitempty,datetime=2006-01-02"`
}

// CreateUserResponse 建立使用者成功回傳
type CreateUserResponse struct {
	Email string `json:"email"`
}

func (h *AccountCreationHandler) CreateUser(c *gin.Context) {
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

	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.Response{Code: http.StatusBadRequest, Message: "Invalid request format: " + err.Error()})
		return
	}

	// 如果 Role 沒帶，預設為 Employee
	if req.Role == 0 {
		req.Role = models.RoleEmployee
	}

	// 權限驗證
	if claims.Role == models.RoleSuperAdmin {
		if req.Role != models.RoleHR && req.Role != models.RoleEmployee {
			c.AbortWithStatusJSON(http.StatusForbidden, common.Response{Code: http.StatusForbidden, Message: "Permission denied: Invalid role specified for creation"})
			return
		}
	} else if claims.Role == models.RoleHR {
		if req.Role != models.RoleEmployee {
			c.AbortWithStatusJSON(http.StatusForbidden, common.Response{Code: http.StatusForbidden, Message: "Permission denied: HR can only create Employee roles"})
			return
		}
	} else {
		c.AbortWithStatusJSON(http.StatusForbidden, common.Response{Code: http.StatusForbidden, Message: "Permission denied: Insufficient privileges to create users"})
		return
	}

	newAccount := &models.Account{
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		Email:       req.Email,
		Role:        req.Role,
		PhoneNumber: "",
	}

	employment := &models.Employment{
		Status:        models.EmploymentStatusActive,
		PositionTitle: req.PositionTitle,
	}

	if req.JobGradeCode != "" {
		jobGrade, err := h.accountSvc.GetJobGradeByCode(c.Request.Context(), req.JobGradeCode)
		if err != nil {
			c.JSON(http.StatusBadRequest, common.Response{
				Code:    http.StatusBadRequest,
				Message: fmt.Sprintf("Invalid JobGradeCode: %v", err),
			})
			return
		}
		employment.JobGradeID = &jobGrade.ID
	}

	if req.Salary != nil && *req.Salary != "" {
		parsedSalary, err := decimal.NewFromString(*req.Salary)
		if err != nil {
			c.JSON(http.StatusBadRequest, common.Response{Code: http.StatusBadRequest, Message: fmt.Sprintf("Invalid Salary format: %v", err)})
			return
		}
		employment.Salary = &parsedSalary
	}

	if req.HireDate != nil && *req.HireDate != "" {
		parsedDate, err := time.Parse("2006-01-02", *req.HireDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, common.Response{Code: http.StatusBadRequest, Message: fmt.Sprintf("Invalid HireDate format: %v", err)})
			return
		}
		employment.HireDate = &parsedDate
	} else {
		now := time.Now()
		employment.HireDate = &now
	}

	createdAccount, err := h.accountSvc.CreateAccountWithEmployment(c.Request.Context(), newAccount, employment)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrEmailExists):
			c.JSON(http.StatusConflict, common.Response{Code: http.StatusConflict, Message: "Email already exists"})
		case errors.Is(err, services.ErrPasswordHashingFailed),
			errors.Is(err, services.ErrAccountCreationFailed),
			errors.Is(err, services.ErrEmploymentCreationFailed):
			log.Printf("Internal error creating user: %v", err)
			c.JSON(http.StatusInternalServerError, common.Response{Code: http.StatusInternalServerError, Message: "Failed to create user due to internal error"})
		default:
			log.Printf("Unexpected error creating user: %v", err)
			c.JSON(http.StatusInternalServerError, common.Response{Code: http.StatusInternalServerError, Message: "An unexpected error occurred"})
		}
		return
	}

	resp := CreateUserResponse{
		Email: createdAccount.Email,
	}
	msg := "Employee created successfully"
	if req.Role == models.RoleHR {
		msg = "HR user created successfully"
	}
	c.JSON(http.StatusCreated, common.Response{Code: http.StatusCreated, Message: msg, Data: resp})
}
