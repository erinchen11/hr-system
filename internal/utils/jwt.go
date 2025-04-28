// 檔案路徑: internal/utils/jwt.go
package utils

import (
	"errors"
	"fmt"
	"log" // 用於記錄警告或錯誤
	"net/http"
	"net/http/httptest"
	"strconv" // 用於轉換 expireHoursStr
	"time"

	"github.com/erinchen11/hr-system/internal/interfaces"
	"github.com/erinchen11/hr-system/internal/models" // 假設 Claims 在 models
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// jwtHelper 結構體 (保持不變)
type jwtHelper struct {
	secretKey      []byte
	issuer         string
	expireDuration time.Duration
}

// --- 完整的 NewJwtUtils 函數 ---

// NewJwtUtils 是 jwtHelper 的構造函數
// 它接收配置字串，進行驗證和轉換，並返回一個 *jwtHelper 實例或錯誤
func NewJwtUtils(secret string, issuer string, expireHoursStr string) (*jwtHelper, error) {
	// 1. 驗證 Secret 是否有效
	//    (不應為空或使用已知的、不安全的預設值)
	if secret == "" {
		return nil, errors.New("JWT secret cannot be empty")
	}
	// 你可以在這裡加入更多對 secret 強度的檢查，或者檢查是否為預設值
	if secret == "change-this-in-production-env-file" {
		// 在生產環境中，如果檢測到這個值，應該報錯或 Fatal
		log.Println("Warning: JWT_SECRET is using the default insecure value. Please set a strong secret.")
		// return nil, errors.New("insecure default JWT secret used") // 可以選擇報錯退出
	}

	// 2. 解析過期時間 (小時)
	var expireHours int
	var err error
	if expireHoursStr == "" {
		log.Println("Warning: JWT_EXPIRE_HOURS not set, using default 24 hours.")
		expireHours = 24 // 提供預設值
	} else {
		expireHours, err = strconv.Atoi(expireHoursStr)
		if err != nil {
			log.Printf("Warning: Invalid JWT_EXPIRE_HOURS '%s', using default 24 hours. Error: %v", expireHoursStr, err)
			expireHours = 24 // 解析失敗也使用預設值
		} else if expireHours <= 0 {
			log.Printf("Warning: Invalid JWT_EXPIRE_HOURS '%d', must be positive. Using default 24 hours.", expireHours)
			expireHours = 24 // 過期時間必須是正數
		}
	}

	// 3. 創建並返回 jwtHelper 實例
	helper := &jwtHelper{
		secretKey:      []byte(secret),
		issuer:         issuer, // 可以考慮檢查 issuer 是否為空
		expireDuration: time.Hour * time.Duration(expireHours),
	}

	// 4. 返回實例和 nil 錯誤表示成功
	return helper, nil
}

// --- GenerateJWT 方法 (不變) ---
func (j *jwtHelper) GenerateJWT(userID string, email string, role uint8) (string, error) {
	expirationTime := time.Now().Add(j.expireDuration)
	claims := &models.Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    j.issuer,
			Subject:   userID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secretKey)
}

// --- ParseJWT 方法 (包含安全檢查，不變) ---
func (j *jwtHelper) ParseJWT(tokenString string, opts ...jwt.ParserOption) (*models.Claims, error) {
	claims := &models.Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.secretKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}
	parsedClaims, ok := token.Claims.(*models.Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return parsedClaims, nil
}

// GenerateTestContext 創建一個模擬的 Gin Context，
// 其中包含模擬 AuthMiddleware 設置的用戶 Claims 和其他相關鍵。
func GenerateTestContext(req *http.Request, userID, email string, role uint8) *gin.Context {
	// 1. 創建一個 ResponseRecorder (儘管在這個函數中我們不直接檢查響應)
	w := httptest.NewRecorder()
	// 2. 創建一個測試用的 Gin Context
	c, _ := gin.CreateTestContext(w)
	// 3. 將傳入的 http.Request 附加到 Context
	// 如果 req 是 nil，可以創建一個預設的
	if req == nil {
		req = httptest.NewRequest(http.MethodGet, "/", nil) // 使用一個簡單的預設請求
	}
	c.Request = req

	// 4. 創建要放入 Context 的 Claims 物件
	//    這裡假設 Claims 結構體定義在 models 包中
	testClaims := &models.Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		// 可以根據需要填充 RegisteredClaims，但對於模擬 Context 通常不是必須的
		RegisteredClaims: jwt.RegisteredClaims{
			// ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			// IssuedAt:  jwt.NewNumericDate(time.Now()),
			// Issuer:    "test-issuer",
			// Subject:   userID,
		},
	}

	// 5. 模擬 AuthMiddleware 的行為，將資訊設置到 Context 中
	c.Set("claims", testClaims) // 設置完整的 Claims 物件
	c.Set("user_id", userID)    // 單獨設置 user_id
	c.Set("email", email)       // 單獨設置 email
	c.Set("role", role)         // 單獨設置 role

	// 6. 返回配置好的 Context
	return c
}

// --- 介面符合性檢查 (可選) ---
var _ interfaces.TokenGenerator = (*jwtHelper)(nil)
var _ interfaces.TokenParser = (*jwtHelper)(nil)

// var _ interfaces.TokenGeneratorParser = (*jwtHelper)(nil) // 如果你沒有組合介面，就不需要這行
