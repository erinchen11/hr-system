package models

import "github.com/golang-jwt/jwt/v5"

// JWT Claims 自定義
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   uint8  `json:"role"`
	jwt.RegisteredClaims
}
