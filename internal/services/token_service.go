// 檔案路徑: internal/services/token_service.go
package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/erinchen11/hr-system/internal/interfaces"
	"github.com/erinchen11/hr-system/internal/models"
)

// tokenServiceImpl 實現了 TokenService 介面
type tokenServiceImpl struct {
	cacheRepo interfaces.CacheRepository
	generator interfaces.TokenGenerator
	parser    interfaces.TokenParser
	cacheTTL  time.Duration
}

// NewTokenServiceImpl 構造函數
func NewTokenServiceImpl(
	cacheRepo interfaces.CacheRepository,
	generator interfaces.TokenGenerator,
	parser interfaces.TokenParser,
	cacheTTLHours int,
) interfaces.TokenService {
	return &tokenServiceImpl{
		cacheRepo: cacheRepo,
		generator: generator,
		parser:    parser,
		cacheTTL:  time.Hour * time.Duration(cacheTTLHours),
	}
}

func (s *tokenServiceImpl) GenerateAndCacheToken(ctx context.Context, user *models.Account) (string, error) {
	token, err := s.generator.GenerateJWT(user.ID.String(), user.Email, uint8(user.Role))
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrTokenGenerationFailed, err)
	}

	cacheKey := "login_token:" + user.ID.String()
	err = s.cacheRepo.Set(ctx, cacheKey, token, s.cacheTTL)
	if err != nil {
		log.Printf("Warning: Failed to cache token for user %s: %v", user.ID.String(), err)
		return "", fmt.Errorf("%w: %w", ErrTokenCacheFailed, err)
	}

	return token, nil
}
func (s *tokenServiceImpl) ValidateToken(ctx context.Context, tokenStr string) (*models.Claims, error) {
	// 1. 解析 Token
	claims, err := s.parser.ParseJWT(tokenStr)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTokenInvalid, err) // <<< 這裡改成 %w: %w
	}
	if claims == nil {
		log.Println("Error: ParseJWT returned nil claims with nil error")
		return nil, ErrTokenInvalid
	}

	// 2. 檢查 Cache
	cacheKey := "login_token:" + claims.UserID
	var cachedToken string
	err = s.cacheRepo.Get(ctx, cacheKey, &cachedToken)
	if err != nil {
		if errors.Is(err, interfaces.ErrCacheMiss) {
			return nil, ErrTokenExpiredOrRevoked
		}
		log.Printf("Cache error validating token for user %s: %v", claims.UserID, err)
		return nil, fmt.Errorf("%w: %w", ErrTokenCacheCheckFailed, err)
	}

	// 3. 比較 Token
	if cachedToken != tokenStr {
		log.Printf("Token mismatch for user %s.", claims.UserID)
		return nil, ErrTokenMismatch
	}

	// 4. 成功
	return claims, nil
}
