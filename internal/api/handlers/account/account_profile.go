// 檔案路徑: internal/api/handlers/user_profile_handler.go (建議重命名)
package handlers

import (
	"errors"
	"log"
	"net/http"
	"time" // 保留 time

	"github.com/erinchen11/hr-system/internal/interfaces" // 導入 models
	common "github.com/erinchen11/hr-system/internal/models/common"
	"github.com/erinchen11/hr-system/internal/services" // 導入 services 以判斷錯誤類型
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"        // 導入 uuid
	"github.com/shopspring/decimal" // 導入 decimal
)

// UserProfileHandler 包含依賴 (*** 建議: 重命名 ***)
type UserProfileHandler struct {
	accountSvc    interfaces.AccountService
	employmentSvc interfaces.EmploymentService
}

// NewUserProfileHandler 構造函數
func NewUserProfileHandler(
	accountSvc interfaces.AccountService,
	employmentSvc interfaces.EmploymentService,
) *UserProfileHandler {
	return &UserProfileHandler{
		accountSvc:    accountSvc,
		employmentSvc: employmentSvc,
	}
}

// UserProfileResponse DTO 定義返回給客戶端的資料結構 (*** 建議: 重命名 ***)
// 欄位來自 Account 和 Employment 模型
type UserProfileResponse struct {
	FirstName     string           `json:"first_name"`               // From Account
	LastName      string           `json:"last_name"`                // From Account
	Email         string           `json:"email"`                    // From Account
	Role          uint8            `json:"role"`                     // From Account
	PhoneNumber   string           `json:"phone_number,omitempty"`   // From Account
	PositionTitle string           `json:"position_title,omitempty"` // From Employment
	Salary        *decimal.Decimal `json:"salary,omitempty"`         // From Employment (是否返回需謹慎)
	HireDate      *time.Time       `json:"hire_date,omitempty"`      // From Employment
	Status        string           `json:"status,omitempty"`         // From Employment
}

// GetProfile 方法處理獲取當前登入用戶個人資料的 HTTP 請求
func (h *UserProfileHandler) GetProfile(c *gin.Context) {
	// 1. 從 Context 獲取用戶 Account ID (由 AuthMiddleware 設置)
	userIDRaw, exists := c.Get("user_id")
	if !exists {
		c.AbortWithStatusJSON(http.StatusUnauthorized, common.Response{Code: http.StatusUnauthorized, Message: "Unauthorized: Missing user identity"})
		return
	}
	userIDStr, ok := userIDRaw.(string)
	if !ok || userIDStr == "" {
		log.Printf("Error: Invalid user ID type in context: %T", userIDRaw)
		c.AbortWithStatusJSON(http.StatusInternalServerError, common.Response{Code: http.StatusInternalServerError, Message: "Internal error processing user identity"})
		return
	}
	accountUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.Printf("Error parsing account ID '%s' from context: %v", userIDStr, err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, common.Response{Code: http.StatusInternalServerError, Message: "Internal error processing user identity format"})
		return
	}

	// 2. 調用 AccountService 獲取帳戶基本資訊
	account, err := h.accountSvc.GetAccount(c.Request.Context(), accountUUID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrAccountNotFound):
			c.JSON(http.StatusNotFound, common.Response{Code: http.StatusNotFound, Message: "User account not found"})
		default:
			log.Printf("Error fetching account profile for %s via service: %v", accountUUID, err)
			c.JSON(http.StatusInternalServerError, common.Response{Code: http.StatusInternalServerError, Message: "Failed to retrieve user profile"})
		}
		return
	}

	// 3. 調用 EmploymentService 獲取僱傭資訊
	employment, err := h.employmentSvc.GetEmploymentByAccountID(c.Request.Context(), accountUUID)
	if err != nil && !errors.Is(err, services.ErrEmploymentNotFound) {
		// 如果不是 "未找到" 錯誤，則記錄並返回內部錯誤
		// 如果是 ErrEmploymentNotFound，對於某些角色 (如 SuperAdmin) 可能是正常的，我們後面處理 DTO 映射時再判斷
		log.Printf("Error fetching employment details for account %s: %v", accountUUID, err)
		c.JSON(http.StatusInternalServerError, common.Response{Code: http.StatusInternalServerError, Message: "Failed to retrieve employment details"})
		return
	}
	// 4. 將 Account 和 Employment (如果存在) 的資料映射到 Response DTO
	responseDTO := UserProfileResponse{
		FirstName:   account.FirstName,
		LastName:    account.LastName,
		Email:       account.Email,
		Role:        account.Role,
		PhoneNumber: account.PhoneNumber,
	}

	if employment != nil { // 只有在找到僱傭記錄時才填充相關欄位
		responseDTO.PositionTitle = employment.PositionTitle
		responseDTO.Salary = employment.Salary // 再次考慮是否要在 Profile API 中返回薪資
		responseDTO.HireDate = employment.HireDate
		responseDTO.Status = employment.Status
	} else {
		responseDTO.Status = "N/A"
	}

	// 5. 返回成功響應
	c.JSON(http.StatusOK, common.Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    responseDTO,
	})
}
