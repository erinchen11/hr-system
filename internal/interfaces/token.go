package interfaces

import (
	"context"

	"github.com/erinchen11/hr-system/internal/models"
	"github.com/golang-jwt/jwt/v5"
)

type TokenGenerator interface {
	GenerateJWT(userID string, email string, role uint8) (string, error)
}

type TokenParser interface {
	ParseJWT(tokenStr string, opts ...jwt.ParserOption) (*models.Claims, error)
}

// TokenService 處理 Token 的生成、驗證和緩存邏輯
type TokenService interface {
	GenerateAndCacheToken(ctx context.Context, user *models.Account) (string, error)
	ValidateToken(ctx context.Context, tokenStr string) (*models.Claims, error)
}
