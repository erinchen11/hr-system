// 檔案路徑: internal/utils/password_test.go
package utils_test // 使用 _test 包後綴，表示黑盒測試

import (
	"testing"

	"github.com/erinchen11/hr-system/internal/utils" // 導入被測包
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require" // 使用 require 確保關鍵步驟成功
)

// TestPasswordHashingAndChecking 測試密碼 Hashing 和檢查的完整流程
func TestPasswordHashingAndChecking(t *testing.T) {
	// 1. 獲取 Hasher 和 Checker 的實例
	// 因為 New... 返回的是介面，而底層是同一個 struct *bcryptPasswordChecker，
	// 所以我們可以分別獲取或只獲取一個然後進行類型斷言，這裡分別獲取更簡單。
	hasher := utils.NewBcryptPasswordHasher()
	checker := utils.NewBcryptPasswordChecker()

	// 使用 require 確保實例創建成功 (雖然不太可能失敗)
	require.NotNil(t, hasher)
	require.NotNil(t, checker)

	// 2. 定義測試用的密碼
	testPasswords := []struct {
		name     string
		password string
	}{
		{"NormalPassword", "mysecretpassword"},
		{"PasswordWithSymbols", "P@$$wOrd!_123"},
		{"ShortPassword", "short"},
		{"EmptyPassword", ""}, // 測試空密碼
	}

	for _, tc := range testPasswords {
		t.Run(tc.name, func(t *testing.T) { // 使用 t.Run 為每個密碼創建子測試
			password := tc.password

			// 3. Hashing 密碼
			hashedPassword, err := hasher.HashPassword(password)
			require.NoError(t, err, "Hashing password '%s' should not produce an error", password)
			require.NotEmpty(t, hashedPassword, "Hashed password for '%s' should not be empty", password)

			// 4. 驗證 CheckPassword
			// 4.1) 正確的密碼應該匹配
			matchCorrect := checker.CheckPassword(hashedPassword, password)
			assert.True(t, matchCorrect, "CheckPassword should return true for correct password '%s'", password)

			// 4.2) 錯誤的密碼不應匹配
			wrongPassword := password + "_invalid"
			matchWrong := checker.CheckPassword(hashedPassword, wrongPassword)
			assert.False(t, matchWrong, "CheckPassword should return false for wrong password '%s'", wrongPassword)

			// 4.3) 正確的密碼和錯誤的 Hash 不應匹配
			invalidHash := "$2a$10$invalidhashplaceholder" // 一個格式可能類似但無效的 hash
			matchInvalidHash := checker.CheckPassword(invalidHash, password)
			assert.False(t, matchInvalidHash, "CheckPassword should return false for correct password '%s' against invalid hash", password)

			// 5. (可選) 驗證同一密碼每次 Hashing 結果不同 (因為 salt 不同)
			// 對於空密碼，bcrypt 的行為可能不同，可以跳過此檢查
			if password != "" {
				hashedPassword2, err2 := hasher.HashPassword(password)
				require.NoError(t, err2)
				assert.NotEqual(t, hashedPassword, hashedPassword2, "Hashing the same password '%s' twice should produce different hashes due to salt", password)
				// 可以再驗證第二個 hash 也能匹配成功
				matchCorrect2 := checker.CheckPassword(hashedPassword2, password)
				assert.True(t, matchCorrect2, "Second hash for '%s' should also match the correct password", password)
			}
		})
	}
}
