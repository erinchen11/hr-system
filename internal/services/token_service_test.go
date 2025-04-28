package services

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/erinchen11/hr-system/internal/interfaces"
	"github.com/erinchen11/hr-system/internal/interfaces/mocks"
	"github.com/erinchen11/hr-system/internal/models"
	"github.com/google/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gomock "github.com/golang/mock/gomock"
)

func TestTokenServiceImpl_GenerateAndCacheToken(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	userEmail := "test@user.com"
	userRole := models.RoleEmployee
	cacheTTLHours := 24 // Example TTL in hours
	expectedTTL := time.Hour * time.Duration(cacheTTLHours)
	expectedCacheKey := "login_token:" + userID.String()
	generatedToken := "valid.jwt.token"

	mockUser := &models.Account{
		ID:    userID,
		Email: userEmail,
		Role:  userRole,
	}

	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockCacheRepo := mocks.NewMockCacheRepository(ctrl)
		mockGenerator := mocks.NewMockTokenGenerator(ctrl)
		// Parser not needed for this method
		service := NewTokenServiceImpl(mockCacheRepo, mockGenerator, nil, cacheTTLHours)

		// 1. Expect GenerateJWT to be called
		mockGenerator.EXPECT().
			GenerateJWT(gomock.Eq(userID.String()), gomock.Eq(userEmail), gomock.Eq(userRole)).
			Return(generatedToken, nil). // Return success
			Times(1)

		// 2. Expect Set to be called on cache
		mockCacheRepo.EXPECT().
			Set(gomock.Any(), gomock.Eq(expectedCacheKey), gomock.Eq(generatedToken), gomock.Eq(expectedTTL)).
			Return(nil). // Return success
			Times(1)

		// Execute
		token, err := service.GenerateAndCacheToken(ctx, mockUser)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, generatedToken, token)
	})

	t.Run("Failure - Generator Error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockCacheRepo := mocks.NewMockCacheRepository(ctrl)
		mockGenerator := mocks.NewMockTokenGenerator(ctrl)
		service := NewTokenServiceImpl(mockCacheRepo, mockGenerator, nil, cacheTTLHours)

		genError := errors.New("jwt signing failed")
		// 1. Expect GenerateJWT to fail
		mockGenerator.EXPECT().
			GenerateJWT(gomock.Any(), gomock.Any(), gomock.Any()). // Match any args if specific ones don't matter for failure
			Return("", genError).                                  // Return error
			Times(1)
		// 2. Cache Set should NOT be called

		// Execute
		token, err := service.GenerateAndCacheToken(ctx, mockUser)

		// Assert
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrTokenGenerationFailed) // Check service level error
		assert.ErrorIs(t, err, genError)                 // Check underlying error
		assert.Empty(t, token)
	})

	t.Run("Failure - Cache Set Error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockCacheRepo := mocks.NewMockCacheRepository(ctrl)
		mockGenerator := mocks.NewMockTokenGenerator(ctrl)
		service := NewTokenServiceImpl(mockCacheRepo, mockGenerator, nil, cacheTTLHours)

		cacheError := errors.New("redis connection failed")
		// 1. Expect GenerateJWT to succeed
		mockGenerator.EXPECT().
			GenerateJWT(gomock.Eq(userID.String()), gomock.Eq(userEmail), gomock.Eq(userRole)).
			Return(generatedToken, nil).
			Times(1)
		// 2. Expect Set to fail
		mockCacheRepo.EXPECT().
			Set(gomock.Any(), gomock.Eq(expectedCacheKey), gomock.Eq(generatedToken), gomock.Eq(expectedTTL)).
			Return(cacheError). // Return cache error
			Times(1)

		// Execute
		token, err := service.GenerateAndCacheToken(ctx, mockUser)

		// Assert
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrTokenCacheFailed) // Check service level error
		assert.ErrorIs(t, err, cacheError)          // Check underlying error
		assert.Empty(t, token)                      // Token should not be returned on cache failure
	})
}

func TestTokenServiceImpl_ValidateToken(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	userEmail := "validate@test.com"
	userRole := models.RoleHR
	tokenStr := "valid.jwt.token.string"
	cacheTTLHours := 1 // TTL doesn't directly affect validation logic itself, only GenerateAndCache
	cacheKey := "login_token:" + userID.String()

	mockClaims := &models.Claims{
		UserID: userID.String(), // Ensure UserID matches cacheKey logic
		Email:  userEmail,
		Role:   userRole,
	}

	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockCacheRepo := mocks.NewMockCacheRepository(ctrl)
		mockParser := mocks.NewMockTokenParser(ctrl)
		service := NewTokenServiceImpl(mockCacheRepo, nil, mockParser, cacheTTLHours)

		// 1. Expect ParseJWT to succeed
		mockParser.EXPECT().ParseJWT(gomock.Eq(tokenStr)).Return(mockClaims, nil).Times(1)

		// 2. Expect Cache Get to succeed AND return the matching token
		mockCacheRepo.EXPECT().
			Get(gomock.Any(), gomock.Eq(cacheKey), gomock.Any()). // Match key, accept any destination
			DoAndReturn(func(ctx context.Context, key string, dest interface{}) error {
				// Simulate finding the token in cache by setting the dest pointer
				// dest is interface{}, need type assertion
				if tokenPtr, ok := dest.(*string); ok {
					*tokenPtr = tokenStr // Set the destination to the original token string
				} else {
					return fmt.Errorf("mock Get: dest is not *string") // Should not happen if service code is correct
				}
				return nil // Return nil error (cache hit)
			}).Times(1)

		// Execute
		claims, err := service.ValidateToken(ctx, tokenStr)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, claims)
		assert.Equal(t, mockClaims.UserID, claims.UserID)
		assert.Equal(t, mockClaims.Email, claims.Email)
		assert.Equal(t, mockClaims.Role, claims.Role)
	})

	t.Run("Failure - Parse Error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockCacheRepo := mocks.NewMockCacheRepository(ctrl)
		mockParser := mocks.NewMockTokenParser(ctrl)
		service := NewTokenServiceImpl(mockCacheRepo, nil, mockParser, cacheTTLHours)

		parseError := errors.New("invalid signature")
		// 1. Expect ParseJWT to fail
		mockParser.EXPECT().ParseJWT(gomock.Eq(tokenStr)).Return(nil, parseError).Times(1)
		// 2. Cache Get should NOT be called

		// Execute
		claims, err := service.ValidateToken(ctx, tokenStr)

		// Assert
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrTokenInvalid) // Check service level error
		assert.ErrorIs(t, err, parseError)      // Check underlying error
		assert.Nil(t, claims)
	})

	t.Run("Failure - Cache Miss", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockCacheRepo := mocks.NewMockCacheRepository(ctrl)
		mockParser := mocks.NewMockTokenParser(ctrl)
		service := NewTokenServiceImpl(mockCacheRepo, nil, mockParser, cacheTTLHours)

		// 1. Expect ParseJWT to succeed
		mockParser.EXPECT().ParseJWT(gomock.Eq(tokenStr)).Return(mockClaims, nil).Times(1)
		// 2. Expect Cache Get to return ErrCacheMiss
		mockCacheRepo.EXPECT().
			Get(gomock.Any(), gomock.Eq(cacheKey), gomock.Any()).
			Return(interfaces.ErrCacheMiss). // Return cache miss error
			Times(1)

		// Execute
		claims, err := service.ValidateToken(ctx, tokenStr)

		// Assert
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrTokenExpiredOrRevoked) // Check specific error for cache miss
		assert.Nil(t, claims)
	})

	t.Run("Failure - Cache Get Error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockCacheRepo := mocks.NewMockCacheRepository(ctrl)
		mockParser := mocks.NewMockTokenParser(ctrl)
		service := NewTokenServiceImpl(mockCacheRepo, nil, mockParser, cacheTTLHours)

		cacheError := errors.New("redis timeout")
		// 1. Expect ParseJWT to succeed
		mockParser.EXPECT().ParseJWT(gomock.Eq(tokenStr)).Return(mockClaims, nil).Times(1)
		// 2. Expect Cache Get to return a different error
		mockCacheRepo.EXPECT().
			Get(gomock.Any(), gomock.Eq(cacheKey), gomock.Any()).
			Return(cacheError). // Return other cache error
			Times(1)

		// Execute
		claims, err := service.ValidateToken(ctx, tokenStr)

		// Assert
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrTokenCacheCheckFailed) // Check specific error for cache issues
		assert.ErrorIs(t, err, cacheError)               // Check underlying error
		assert.Nil(t, claims)
	})

	t.Run("Failure - Token Mismatch", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockCacheRepo := mocks.NewMockCacheRepository(ctrl)
		mockParser := mocks.NewMockTokenParser(ctrl)
		service := NewTokenServiceImpl(mockCacheRepo, nil, mockParser, cacheTTLHours)

		differentToken := "different.jwt.token"
		// 1. Expect ParseJWT to succeed
		mockParser.EXPECT().ParseJWT(gomock.Eq(tokenStr)).Return(mockClaims, nil).Times(1)
		// 2. Expect Cache Get to succeed BUT return a different token
		mockCacheRepo.EXPECT().
			Get(gomock.Any(), gomock.Eq(cacheKey), gomock.Any()).
			DoAndReturn(func(ctx context.Context, key string, dest interface{}) error {
				// Simulate finding a DIFFERENT token in cache
				if tokenPtr, ok := dest.(*string); ok {
					*tokenPtr = differentToken // Set the destination to the different token
				} else {
					return fmt.Errorf("mock Get: dest is not *string")
				}
				return nil // Cache hit, but wrong token
			}).Times(1)

		// Execute
		claims, err := service.ValidateToken(ctx, tokenStr)

		// Assert
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrTokenMismatch) // Check specific error for mismatch
		assert.Nil(t, claims)
	})
}
