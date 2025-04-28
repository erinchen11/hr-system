// 檔案路徑: internal/config/init.go
package config

import (
	"log"
	"os"
	"strconv" 

	"github.com/erinchen11/hr-system/environment" // 導入 environment 以便填充變數
	"github.com/joho/godotenv"
)

// LoadConfig 加載來自 .env 檔案和操作系統環境變數的配置。
// 它會確定運行環境，優先加載特定環境的 .env 檔案 (例如 .env.docker)，
// 然後讀取 OS 環境變數（會覆蓋 .env 的值），並為未設置的變數提供預設值，
// 最後填充 environment 包中的導出變數。
// 這個函數應該在 main.go 的最開始被調用一次。
func LoadConfig() {
	// 1. 確定環境
	//    優先讀取 OS 的環境變數 ENVIRONMENT。
	//    如果 OS 沒有，嘗試從基礎 .env 檔案讀取。
	//    如果都沒有，預設為 "local"。
	_ = godotenv.Load(".env") // 嘗試加載基礎 .env
	env := getEnv("ENVIRONMENT", "local")
	environment.Environment = env // 存儲確定的環境
	log.Printf("Loading configuration for environment: %s", env)

	// 2. 加載特定環境的 .env 檔案 (例如 .env.local, .env.docker)
	//    這允許特定環境的配置覆蓋基礎 .env 或 OS 環境變數(如果 godotenv 這樣設置)
	//    注意：godotenv 預設不會覆蓋已存在的 OS 環境變數。
	//    如果需要 .env 覆蓋 OS 變數，需要使用 godotenv.Overload()
	envFileName := ".env." + env
	if err := godotenv.Load(envFileName); err != nil && !os.IsNotExist(err) {
		log.Printf("Warning: Could not load '%s': %v", envFileName, err)
	} else if err == nil {
		log.Printf("Loaded environment-specific config from '%s'", envFileName)
	}

	// 3. 解析環境變數 (OS > .env) 並填充 environment 包
	//    getEnv 函數會優先讀取 OS 環境變數，如果不存在則使用 fallback
	environment.ServerPort = getEnv("SERVER_PORT", "8080")
	environment.GinMode = getEnv("GIN_MODE", "debug")
	environment.Domain = getEnv("DOMAIN", "localhost") // 添加 Domain 配置

	environment.DatabaseUser = getEnv("MYSQL_USER", "")
	environment.DatabasePassword = getEnv("MYSQL_PASSWORD", "")
	environment.DatabaseName = getEnv("MYSQL_DATABASE", "")
	environment.DatabaseHost = getEnv("MYSQL_HOST", "127.0.0.1")
	environment.DatabasePort = getEnv("MYSQL_PORT", "3306")                     // 存儲為 string
	environment.DatabaseLogLevel = getEnv("DB_LOG_LEVEL", "warn")               // 添加日誌級別
	environment.DatabaseMaxIdleConns = getEnv("DB_MAX_IDLE_CONNS", "10")        // 存儲為 string
	environment.DatabaseMaxOpenConns = getEnv("DB_MAX_OPEN_CONNS", "100")       // 存儲為 string
	environment.DatabaseMaxLifetime = getEnv("DB_MAX_LIFETIME_SECONDS", "3600") // 存儲為 string

	environment.RedisAddr = getEnv("REDIS_ADDR", "127.0.0.1:6379")
	environment.RedisPassword = getEnv("REDIS_PASSWORD", "")
	environment.RedisDB = getEnv("REDIS_DB", "0") // 存儲為 string

	environment.JwtSecret = getEnv("JWT_SECRET", "change-this-in-production-env-file")
	environment.JwtExpireHours = getEnv("JWT_EXPIRE_HOURS", "24") // 存儲為 string
	environment.DefaultPassword = getEnv("DEFAULT_PASSWORD", "")  // 預設密碼可能為空

	// 4. 添加關鍵配置的檢查和警告
	checkCriticalConfigs()

	log.Println("Configuration loading complete.")
}

// getEnv 輔助函數：優先讀取 OS 環境變數，若無則返回 fallback
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value // OS 環境變數優先
	}
	// 注意：godotenv 加載的值也會被 os.LookupEnv 讀取到
	// 所以如果 OS 沒設置，但 .env 設置了，這裡會返回 .env 的值
	// 只有當 OS 和 .env 都沒設置時，才使用 fallback
	return fallback
}


func getEnvInt(key string, fallback int) int {
	sVal := getEnv(key, "")
	if sVal != "" {
		if v, err := strconv.Atoi(sVal); err == nil {
			return v
		}
		log.Printf("Warning: Cannot parse integer for env var %s, value '%s'. Using fallback %d.", key, sVal, fallback)
	}
	return fallback
}

// checkCriticalConfigs 檢查關鍵配置是否缺失或使用不安全的預設值
func checkCriticalConfigs() {
	if environment.DatabaseUser == "" {
		log.Println("Warning: MYSQL_USER environment variable not set.")
	}
	
	if environment.JwtSecret == "change-this-in-production-env-file" {
		log.Println("Warning: JWT_SECRET is using the default insecure value. Set the JWT_SECRET environment variable.")
	}
	if environment.DefaultPassword == "" {
		log.Println("Warning: DEFAULT_PASSWORD environment variable not set. Default password for new users will be empty.")
	}
}
