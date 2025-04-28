// 檔案路徑: internal/utils/password.go
package utils

import (
	"log"

	"github.com/erinchen11/hr-system/internal/interfaces" // 導入介面定義
	"golang.org/x/crypto/bcrypt"
)

// bcryptPasswordChecker 同時實現了 PasswordChecker 和 PasswordHasher 介面
type bcryptPasswordChecker struct{} // 結構體保持不變

// --- PasswordChecker 實現 ---

// NewBcryptPasswordChecker 構造函數返回 PasswordChecker 介面
func NewBcryptPasswordChecker() interfaces.PasswordChecker {
	return &bcryptPasswordChecker{}
}

// CheckPassword 使用 bcrypt 比較密碼 (實現不變)
func (pc *bcryptPasswordChecker) CheckPassword(hashedPassword, plainPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
	if err != nil {
		if err != bcrypt.ErrMismatchedHashAndPassword {
			log.Printf("Warning: bcrypt comparison error: %v", err)
		}
		return false
	}
	return true
}

// --- PasswordHasher 實現 ---

// HashPassword 實現 PasswordHasher 介面的方法
func (pc *bcryptPasswordChecker) HashPassword(plainPassword string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	return string(bytes), err
}

// NewBcryptPasswordHasher 構造函數返回 PasswordHasher 介面
func NewBcryptPasswordHasher() interfaces.PasswordHasher {
	return &bcryptPasswordChecker{}
}

// --- 或者，如果不想讓同一個 struct 實現兩個介面，可以分開 ---
/*
type bcryptPasswordHasher struct{}

func NewBcryptPasswordHasher() interfaces.PasswordHasher {
    return &bcryptPasswordHasher{}
}

func (ph *bcryptPasswordHasher) HashPassword(plainPassword string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	return string(bytes), err
}
*/