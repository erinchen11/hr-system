package handlers

import (
	"log"
	"net/http"

	"github.com/erinchen11/hr-system/internal/interfaces"
	common "github.com/erinchen11/hr-system/internal/models/common"
	"github.com/gin-gonic/gin"
)

// LoginHandler 包含依賴
type LoginHandler struct {
	AuthSvc  interfaces.AuthService
	TokenSvc interfaces.TokenService
}

// NewLoginHandler 構造函數
func NewLoginHandler(authSvc interfaces.AuthService, tokenSvc interfaces.TokenService) *LoginHandler {
	return &LoginHandler{
		AuthSvc:  authSvc,
		TokenSvc: tokenSvc,
	}
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Login 方法處理登入邏輯
func (h *LoginHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 建議填充完整的 Response 內容
		c.JSON(http.StatusBadRequest, common.Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid request format: " + err.Error(), // 可以包含錯誤細節
			Data:    nil,
		})
		return
	}

	// 使用注入的 AuthService (現在是 h.AuthSvc，類型是 interfaces.AuthService)
	user, err := h.AuthSvc.Authenticate(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		// 根據 AuthService 返回的錯誤類型判斷
		// 建議在 Service 層返回定義好的錯誤變量 (如 services.ErrInvalidCredentials)
		statusCode := http.StatusUnauthorized
		errMsg := "Invalid email or password" // 預設錯誤訊息

		c.JSON(statusCode, common.Response{
			Code:    statusCode,
			Message: errMsg,
			Data:    nil,
		})
		return
	}

	// 使用注入的 TokenService (現在是 h.TokenSvc，類型是 interfaces.TokenService)
	token, err := h.TokenSvc.GenerateAndCacheToken(c.Request.Context(), user) // user 類型是 *models.Employee
	if err != nil {
		// 記錄內部錯誤
		log.Printf("Token processing error: %v", err)
		c.JSON(http.StatusInternalServerError, common.Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to process token", // 對外隱藏內部錯誤細節
			Data:    nil,
		})
		return
	}

	c.JSON(http.StatusOK, common.Response{
		Code:    http.StatusOK, 
		Message: "Login success",
		Data: gin.H{
			"token": token,
			"user": gin.H{ 
				"email":      user.Email,
				"role":       user.Role,
				"first_name": user.FirstName, 
				"last_name":  user.LastName,
			},
		},
	})
}
