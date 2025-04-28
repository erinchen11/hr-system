package middleware

import (
	"log"
	"net/http"
	"strings"

	"github.com/erinchen11/hr-system/internal/interfaces" 
	"github.com/erinchen11/hr-system/internal/models/common"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware 結構體包含依賴
type AuthMiddleware struct {
	TokenSvc interfaces.TokenService
}

// NewAuthMiddleware 構造函數
func NewAuthMiddleware(tokenSvc interfaces.TokenService) *AuthMiddleware {
	return &AuthMiddleware{TokenSvc: tokenSvc}
}

// Authenticate 返回實際的 Middleware HandlerFunc
func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// *** 添加 Code 欄位 ***
			c.AbortWithStatusJSON(http.StatusUnauthorized, common.Response{
				Code:    http.StatusUnauthorized,        
				Message: "Authorization header required", 
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		// 使用 EqualFold 比較 "Bearer" (忽略大小寫)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			// ***  : 添加 Code 欄位, 調整 Message ***
			c.AbortWithStatusJSON(http.StatusUnauthorized, common.Response{
				Code:    http.StatusUnauthorized,                         
				Message: "Invalid Authorization format (Bearer <token>)", 
			})
			return
		}
		tokenStr := parts[1]

		// 使用注入的 TokenService
		claims, err := m.TokenSvc.ValidateToken(c.Request.Context(), tokenStr)
		if err != nil {
			log.Printf("Token validation failed during middleware: %v", err) // 添加日誌
			// ***  添加 Code 欄位 ***
			c.AbortWithStatusJSON(http.StatusUnauthorized, common.Response{
				Code:    http.StatusUnauthorized,    
				Message: "Invalid or expired token", 
			})
			return
		}

		// ---!!! 關鍵 ：移除類型斷言，增加 nil 檢查 !!!---
		// 因為 ValidateToken 成功時直接返回 *models.Claims，所以不需要斷言
		// 但要檢查 Service 是否可能在 err == nil 時返回 nil claims (雖然不應如此)
		if claims == nil {
			log.Printf("Error: ValidateToken returned nil claims with nil error")
			c.AbortWithStatusJSON(http.StatusInternalServerError, common.Response{
				Code:    http.StatusInternalServerError,
				Message: "Internal server error processing token claims",
			})
			return
		}
		// ----------------------------------------------------

		// 將驗證後的資訊放入 Context (直接使用 claims)
		c.Set("claims", claims) // <-- 直接使用 claims (類型已是 *models.Claims)
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)

		c.Next()
	}
}
