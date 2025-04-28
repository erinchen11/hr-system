// 檔案路徑: internal/api/handlers/change_password_handler.go
package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/erinchen11/hr-system/internal/interfaces"
	common "github.com/erinchen11/hr-system/internal/models/common"
	"github.com/erinchen11/hr-system/internal/services" // 導入 services 以判斷錯誤類型
	"github.com/gin-gonic/gin"
	"github.com/google/uuid" // *** 新增: 導入 uuid ***
)

// AccountPasswordHandler 包含依賴 (*** 建議: 重命名結構體 ***)
type AccountPasswordHandler struct {
	accountSvc interfaces.AccountService // ***  : 注入 AccountService ***
}

// NewAccountPasswordHandler 構造函數 (*** 建議: 重命名函數 ***)
func NewAccountPasswordHandler(accountSvc interfaces.AccountService) *AccountPasswordHandler {
	return &AccountPasswordHandler{accountSvc: accountSvc}
}

// ChangePasswordRequest 請求體結構 (保持不變)
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// ChangePassword 方法處理 密碼的 HTTP 請求
func (h *AccountPasswordHandler) ChangePassword(c *gin.Context) {
	// 1. 從 Context 獲取用戶 ID (應為 Account ID)
	userIDRaw, exists := c.Get("user_id") // 假設 Middleware 設置的是 Account ID string
	if !exists {
		c.AbortWithStatusJSON(http.StatusUnauthorized, common.Response{
			Code: http.StatusUnauthorized, Message: "Unauthorized: Missing user identity",
		})
		return
	}
	userIDStr, ok := userIDRaw.(string)
	if !ok || userIDStr == "" {
		log.Printf("Error: Invalid user ID type in context: %T", userIDRaw)
		c.AbortWithStatusJSON(http.StatusInternalServerError, common.Response{
			Code: http.StatusInternalServerError, Message: "Internal error processing user identity",
		})
		return
	}

	accountUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.Printf("Error parsing account ID '%s' from context: %v", userIDStr, err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, common.Response{
			Code: http.StatusInternalServerError, Message: "Internal error processing user identity format",
		})
		return
	}

	// 2. 解析請求體
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.Response{
			Code: http.StatusBadRequest, Message: "Invalid request body", 
		})
		return
	}

	// 3. 基礎業務驗證：新舊密碼不能相同
	if req.OldPassword == req.NewPassword {
		c.JSON(http.StatusBadRequest, common.Response{
			Code: http.StatusBadRequest, Message: "New password cannot be the same as the old password",
		})
		return
	}

	// 4. 調用 Service 層處理業務邏輯
	err = h.accountSvc.ChangePassword(c.Request.Context(), accountUUID, req.OldPassword, req.NewPassword)

	// 5. 處理 Service 層返回的錯誤
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, common.Response{Code: http.StatusUnauthorized, Message: "Old password is incorrect"})
		case errors.Is(err, services.ErrAccountNotFound):
			// 正常情況下，如果 ID 來自 context，這個錯誤不應輕易觸發，除非帳戶剛好被刪除
			log.Printf("Account not found during password change for ID %s (potentially deleted?)", accountUUID)
			c.JSON(http.StatusNotFound, common.Response{Code: http.StatusNotFound, Message: "User account not found"})
		case errors.Is(err, services.ErrPasswordHashingFailed):
			// 記錄內部錯誤，對外模糊
			log.Printf("Internal error during ChangePassword for account %s: %v", accountUUID, err)
			c.JSON(http.StatusInternalServerError, common.Response{Code: http.StatusInternalServerError, Message: "Failed to process new password"})
		case errors.Is(err, services.ErrPasswordUpdateFailed):
			// 記錄內部錯誤，對外模糊
			log.Printf("Internal error during ChangePassword for account %s: %v", accountUUID, err)
			c.JSON(http.StatusInternalServerError, common.Response{Code: http.StatusInternalServerError, Message: "Failed to update password"})
		default:
			// 記錄未知錯誤
			log.Printf("Unexpected error during ChangePassword for account %s: %v", accountUUID, err)
			c.JSON(http.StatusInternalServerError, common.Response{Code: http.StatusInternalServerError, Message: "An unexpected error occurred"})
		}
		return
	}

	// 6. 返回成功響應
	c.JSON(http.StatusOK, common.Response{
		Code:    http.StatusOK,
		Message: "Password updated successfully.", 
		Data:    nil,
	})
}
