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
	_ = godotenv.Load(".env") // 嘗試加載基礎 .env
	env := getEnv("ENVIRONMENT", environment.Environment)
	environment.Environment = env
	log.Printf("Loading configuration for environment: %s", env)

	envFileName := ".env." + env
	if err := godotenv.Load(envFileName); err != nil && !os.IsNotExist(err) {
		log.Printf("Warning: Could not load '%s': %v", envFileName, err)
	} else if err == nil {
		log.Printf("Loaded environment-specific config from '%s'", envFileName)
	}

	environment.ServerPort = getEnv("SERVER_PORT", environment.DefaultServerPort)
	environment.GinMode = getEnv("GIN_MODE", environment.DefaultGinMode)
	environment.Domain = getEnv("DOMAIN", environment.DefaultDomain)

	environment.DatabaseUser = getEnv("MYSQL_USER", "")
	environment.DatabasePassword = getEnv("MYSQL_PASSWORD", "")
	environment.DatabaseName = getEnv("MYSQL_DATABASE", "")
	environment.DatabaseHost = getEnv("MYSQL_HOST", environment.DefaultDatabaseHost)
	environment.DatabasePort = getEnv("MYSQL_PORT", environment.DefaultDatabasePort)
	environment.DatabaseLogLevel = getEnv("DB_LOG_LEVEL", environment.DefaultDatabaseLogLevel)
	environment.DatabaseMaxIdleConns = getEnv("DB_MAX_IDLE_CONNS", environment.DefaultDatabaseMaxIdleConns)
	environment.DatabaseMaxOpenConns = getEnv("DB_MAX_OPEN_CONNS", environment.DefaultDatabaseMaxOpenConns)
	environment.DatabaseMaxLifetime = getEnv("DB_MAX_LIFETIME_SECONDS", environment.DefaultDatabaseMaxLifetime)

	environment.RedisAddr = getEnv("REDIS_ADDR", environment.DefaultRedisAddr)
	environment.RedisPassword = getEnv("REDIS_PASSWORD", "")
	environment.RedisDB = getEnv("REDIS_DB", environment.DefaultRedisDB)

	environment.JwtSecret = getEnv("JWT_SECRET", environment.DefaultJwtSecret)
	environment.JwtExpireHours = getEnv("JWT_EXPIRE_HOURS", environment.DefaultJwtExpireHours)
	environment.DefaultPassword = getEnv("DEFAULT_PASSWORD", "")

	checkCriticalConfigs()
	log.Println("Configuration loading complete.")
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
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

func checkCriticalConfigs() {
	if environment.DatabaseUser == "" {
		log.Println("Warning: MYSQL_USER environment variable not set.")
	}
	if environment.JwtSecret == environment.DefaultJwtSecret {
		log.Println("Warning: JWT_SECRET is using the default insecure value. Set the JWT_SECRET environment variable.")
	}
	if environment.DefaultPassword == "" {
		log.Println("Warning: DEFAULT_PASSWORD environment variable not set. Default password for new users will be empty.")
	}
}
