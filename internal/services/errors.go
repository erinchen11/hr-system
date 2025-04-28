package services

import "errors"

// ==================== Account Service 錯誤 ====================

var (
	ErrAccountNotFound           = errors.New("account not found")
	ErrPasswordHashingFailed     = errors.New("failed to hash new password")
	ErrPasswordUpdateFailed      = errors.New("failed to update password")
	ErrInvalidCredentials        = errors.New("invalid credentials")
	ErrEmailExists               = errors.New("email already exists")
	ErrAccountCreationFailed     = errors.New("failed to create account")
	ErrEmploymentCreationFailed  = errors.New("failed to create employment record for account")
)

// ==================== Employment Service 錯誤 ====================

var (
	ErrEmploymentNotFound  = errors.New("employment record not found")
	ErrUpdateFailed        = errors.New("failed to update employment details")
	ErrTerminationFailed   = errors.New("failed to terminate employment")
	ErrAlreadyTerminated   = errors.New("employment record is already terminated")
)

// ==================== Leave Request 錯誤 ====================

var (
	ErrLeaveRequestNotFound     = errors.New("leave request not found")
	ErrInvalidLeaveRequestState = errors.New("leave request is not in pending state for this operation")
	ErrLeaveRequestUpdateFailed = errors.New("failed to update leave request")
	ErrInvalidDateRange         = errors.New("invalid date range: end date cannot be before start date")
	ErrLeaveApplyFailed         = errors.New("failed to apply for leave")
	ErrInvalidProcessor         = errors.New("invalid processor account or insufficient permissions")
	// 可以未來新增 ErrForbidden 等權限不足錯誤
)

// ==================== Token Service 錯誤 ====================

var (
	ErrTokenGenerationFailed = errors.New("failed to generate token")
	ErrTokenCacheFailed      = errors.New("failed to cache token")
	ErrTokenInvalid          = errors.New("invalid token")
	ErrTokenExpiredOrRevoked = errors.New("token expired or revoked")
	ErrTokenCacheCheckFailed = errors.New("cache error validating token")
	ErrTokenMismatch         = errors.New("token mismatch")
)
