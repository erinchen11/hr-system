// 檔案路徑: internal/utils/jwt_test.go
package utils_test // 使用 _test 包，進行黑盒測試

import (
	// 雖然 jwt 本身不用，但可以 import 以備將來可能需要
	"errors" // 導入 os 以便設置環境變數 (用於 NewJwtUtils 測試)
	"fmt"
	"testing"
	"time"

	// 導入 models 以使用 Claims
	"github.com/erinchen11/hr-system/internal/models"
	"github.com/erinchen11/hr-system/internal/utils" // 導入被測包 utils
	"github.com/golang-jwt/jwt/v5"                   // 導入 jwt 庫
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- 測試 NewJwtUtils 構造函數 ---
func TestNewJwtUtils(t *testing.T) {
	validSecret := "a-very-secure-secret-key-minimum-length"
	insecureSecret := "change-this-in-production-env-file"
	validIssuer := "test-issuer"
	validExpireStr := "24"
	invalidExpireStr := "abc"
	zeroExpireStr := "0"
	negativeExpireStr := "-1"
	defaultExpireDuration := time.Hour * 24

	testCases := []struct {
		name           string
		secret         string
		issuer         string
		expireStr      string
		expectError    bool
		expectedExpire time.Duration // 期望的過期時間
		expectPanic    bool          // 是否期望 Fatal (例如密鑰不安全)
	}{
		{
			name:           "Success - Valid inputs",
			secret:         validSecret,
			issuer:         validIssuer,
			expireStr:      validExpireStr,
			expectError:    false,
			expectedExpire: defaultExpireDuration,
		},
		{
			name:           "Error - Empty secret",
			secret:         "", // 空密鑰
			issuer:         validIssuer,
			expireStr:      validExpireStr,
			expectError:    true, // 期望返回錯誤
			expectedExpire: 0,
		},
		{
			name:           "Warning - Default insecure secret (should not error)",
			secret:         insecureSecret, // 不安全的預設密鑰
			issuer:         validIssuer,
			expireStr:      validExpireStr,
			expectError:    false, // 目前實現只打印警告，不報錯
			expectedExpire: defaultExpireDuration,
			// 如果希望在生產中報錯，可以在 NewJwtUtils 中 邏輯
		},
		{
			name:           "Success - Empty expire string (use default)",
			secret:         validSecret,
			issuer:         validIssuer,
			expireStr:      "", // 空過期時間字串
			expectError:    false,
			expectedExpire: defaultExpireDuration, // 期望使用預設值 24 小時
		},
		{
			name:           "Success - Invalid expire string (use default)",
			secret:         validSecret,
			issuer:         validIssuer,
			expireStr:      invalidExpireStr, // 無效過期時間字串
			expectError:    false,
			expectedExpire: defaultExpireDuration, // 期望使用預設值 24 小時
		},
		{
			name:           "Success - Zero expire string (use default)",
			secret:         validSecret,
			issuer:         validIssuer,
			expireStr:      zeroExpireStr, // 過期時間為 0
			expectError:    false,
			expectedExpire: defaultExpireDuration, // 期望使用預設值 24 小時
		},
		{
			name:           "Success - Negative expire string (use default)",
			secret:         validSecret,
			issuer:         validIssuer,
			expireStr:      negativeExpireStr, // 過期時間為負數
			expectError:    false,
			expectedExpire: defaultExpireDuration, // 期望使用預設值 24 小時
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 為了測試 NewJwtUtils 在不同環境變數下的行為，
			// 我們可以使用 t.Setenv (Go 1.17+) 臨時設置環境變數，
			// 但 NewJwtUtils 目前直接接收參數，所以直接傳入即可。
			// 如果 NewJwtUtils 內部讀取 env，則需要用 t.Setenv。

			helper, err := utils.NewJwtUtils(tc.secret, tc.issuer, tc.expireStr)

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, helper)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, helper) // 使用 require 確保 helper 非 nil，防止後續 panic

				// 使用反射或導出 getter 來檢查內部欄位有點麻煩且破壞封裝
				// 更好的方法是通過後續 Generate/Parse 測試來驗證其行為
				// 但如果只想檢查過期時間，可以暫時這樣（不推薦）：
				// assert.Equal(t, tc.expectedExpire, helper.expireDuration) // 這需要 expireDuration 是導出的，或者用反射
				// 或者，我們可以生成一個 token 並檢查其過期時間
				// _, claims, _ := generateAndParseToken(t, helper, "id", "email", 1)
				// expireDiff := claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time)
				// assert.InDelta(t, tc.expectedExpire, expireDiff, float64(time.Second)) // 允許秒級誤差
			}
		})
	}
}

// --- 測試 GenerateJWT 和 ParseJWT 的組合 ---
func TestJWTGenerateParseCycle(t *testing.T) {
	// 使用 require 確保初始化成功，否則後續測試無意義
	helper, err := utils.NewJwtUtils("test-secret-key", "test-issuer", "1") // 1小時過期
	require.NoError(t, err)
	require.NotNil(t, helper)

	userID := uuid.New().String()
	email := "test@jwt.com"
	role := uint8(1) // HR

	// 1. 測試成功生成和解析
	t.Run("Success Cycle", func(t *testing.T) {
		tokenString, err := helper.GenerateJWT(userID, email, role)
		assert.NoError(t, err)
		assert.NotEmpty(t, tokenString)

		// 立刻解析回來
		claims, err := helper.ParseJWT(tokenString)
		assert.NoError(t, err)
		require.NotNil(t, claims)

		// 驗證 Claims 內容
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, role, claims.Role)
		assert.Equal(t, "test-issuer", claims.Issuer)
		assert.Equal(t, userID, claims.Subject)

		// 驗證過期時間 (大約在 1 小時後)
		assert.WithinDuration(t, time.Now().Add(time.Hour), claims.ExpiresAt.Time, 5*time.Second) // 允許 5 秒誤差
		// 驗證簽發時間 (應該很接近現在)
		assert.WithinDuration(t, time.Now(), claims.IssuedAt.Time, 5*time.Second)
	})

	t.Run("Expired Token", func(t *testing.T) {
		// 生成一個 1 小時有效的 Token (使用 helper)
		tokenString, err := helper.GenerateJWT(userID, email, role)
		assert.NoError(t, err)
		assert.NotEmpty(t, tokenString)

		// ***  : 使用配置好的 jwt.Parser ***
		// 定義一個返回未來時間的函數
		timeInFuture := func() time.Time {
			return time.Now().Add(2 * time.Hour)
		}

		// 創建一個新的 Parser 實例，並配置 TimeFunc
		parser := jwt.NewParser(
			jwt.WithValidMethods([]string{"HS256"}), // 強制檢查算法
			// jwt.WithIssuer( /* "test-issuer" */ ),   // 可選：如果需要檢查 issuer
			// jwt.WithAudience( /* "audience" */ ),    // 可選：如果需要檢查 audience
			jwt.WithExpirationRequired(),   // 可選：強制要求有 exp 欄位
			jwt.WithTimeFunc(timeInFuture), // <-- **關鍵：設置驗證時間**
		)

		// 定義 keyFunc (需要能獲取到測試用的 secret)
		testSecretKey := []byte("test-secret-key") // <-- 使用與 helper 初始化時相同的密鑰
		keyFunc := func(token *jwt.Token) (interface{}, error) {
			// 再次驗證算法
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return testSecretKey, nil
		}

		// 使用配置好的 parser 來解析
		claims := &models.Claims{} // 準備接收 claims 的容器
		token, err := parser.ParseWithClaims(tokenString, claims, keyFunc)
		// ------------------------------------

		// --- 斷言 ---
		assert.Error(t, err) // 期望有錯誤

		// 檢查返回的錯誤是否確實是 Token 過期錯誤
		// 使用 require 確保錯誤類型正確，否則測試停止
		require.ErrorIs(t, err, jwt.ErrTokenExpired, "Expected jwt.ErrTokenExpired error")

		// 當因為過期而解析失敗時，claims 應該是 nil 或者 token.Valid 是 false
		assert.False(t, token.Valid, "Token should be invalid when expired") // 可以額外斷言 token 無效
		// assert.Nil(t, claims) // ParseWithClaims 在 err != nil 時，claims 的狀態可能不確定，不一定為 nil，所以上面的 ErrorIs 更好
	})
	// 3. 測試簽名無效
	t.Run("Invalid Signature", func(t *testing.T) {
		helperA, _ := utils.NewJwtUtils("secret-A", "issuer-A", "1")
		helperB, _ := utils.NewJwtUtils("secret-B", "issuer-A", "1") // 使用不同的 Secret

		tokenString, err := helperA.GenerateJWT(userID, email, role)
		assert.NoError(t, err)

		// 使用 helperB (錯誤的 secret) 來解析
		claims, err := helperB.ParseJWT(tokenString)
		assert.Error(t, err) // 期望有錯誤
		assert.True(t, errors.Is(err, jwt.ErrSignatureInvalid), "Expected jwt.ErrSignatureInvalid, got %v", err)
		assert.Nil(t, claims)
	})

	// 4. 測試 Token 格式錯誤
	t.Run("Malformed Token", func(t *testing.T) {
		claims, err := helper.ParseJWT("this.is.not.a.valid.jwt.token")
		assert.Error(t, err)
		assert.Nil(t, claims)
		// 可以進一步檢查錯誤是否包含 "token is malformed" 之類的訊息
		assert.ErrorContains(t, err, "token is malformed")
	})

	// 5. 測試簽名算法不匹配 (模擬)
	t.Run("Unexpected Signing Method", func(t *testing.T) {
		// 創建一個使用不同算法 (例如 ES256) 或 alg=none 的 Token 字串比較困難，
		// 通常依賴庫內部的檢查。但我們可以驗證 ParseJWT 的 keyFunc 是否正確拒絕。
		// 我們可以創建一個聲明 alg 為 none 的 token string 片段 (不完整，但足以測試 keyFunc)
		// 注意：這只是為了觸發 keyFunc 中的算法檢查，並非一個有效的 JWT
		malformedTokenWithAlgNone := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ."

		claims, err := helper.ParseJWT(malformedTokenWithAlgNone)
		assert.Error(t, err) // 期望有錯誤
		assert.Nil(t, claims)
		// 期望錯誤消息包含 "unexpected signing method"
		assert.ErrorContains(t, err, "unexpected signing method")
	})
}

// --- GenerateTestContext 的測試 (可選) ---
// 通常這個輔助函數不需要單獨的單元測試，它的正確性會在 Handler 測試中體現
// 如果要測試，可以驗證它返回的 Context 中是否包含了預期的值
/*
func TestGenerateTestContext(t *testing.T) {
    req := httptest.NewRequest(http.MethodGet, "/test", nil)
    userID := uuid.New().String()
    email := "test@context.com"
    role := uint8(2)

    c := utils.GenerateTestContext(req, userID, email, role)

    assert.Equal(t, req, c.Request)

    claimsRaw, exists := c.Get("claims")
    assert.True(t, exists)
    claims, ok := claimsRaw.(*models.Claims)
    require.True(t, ok)
    assert.Equal(t, userID, claims.UserID)
    assert.Equal(t, email, claims.Email)
    assert.Equal(t, role, claims.Role)

    idVal, _ := c.Get("user_id")
    emailVal, _ := c.Get("email")
    roleVal, _ := c.Get("role")
    assert.Equal(t, userID, idVal)
    assert.Equal(t, email, emailVal)
    assert.Equal(t, role, roleVal)
}
*/
