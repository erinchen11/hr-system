package environment

// --- Configuration Variables ---
var (
	Environment string // local, docker, production
	GinMode     string // debug, release

	// 伺服器
	ServerPort string
	Domain     string

	// 資料庫
	DatabaseUser         string
	DatabasePassword     string
	DatabaseName         string
	DatabaseHost         string
	DatabasePort         string
	DatabaseLogLevel     string
	DatabaseMaxIdleConns string
	DatabaseMaxOpenConns string
	DatabaseMaxLifetime  string // second

	// Redis
	RedisAddr     string
	RedisPassword string
	RedisDB       string

	// Secrets / App Specific
	JwtSecret       string
	JwtExpireHours  string // 小時 (保持為 string)
	DefaultPassword string // 新用戶的預設密碼
)

// API 的基礎路徑
const BasePath = "/hr-system-api"

// --- Default fallback values ---
const (
	DefaultEnv        = "local"
	DefaultGinMode    = "debug"
	DefaultServerPort = "8080"
	DefaultDomain     = "localhost"

	DefaultDatabaseHost         = "127.0.0.1"
	DefaultDatabasePort         = "3306"
	DefaultDatabaseLogLevel     = "warn"
	DefaultDatabaseMaxIdleConns = "10"
	DefaultDatabaseMaxOpenConns = "100"
	DefaultDatabaseMaxLifetime  = "3600"

	DefaultRedisAddr = "127.0.0.1:6379"
	DefaultRedisDB   = "0"

	DefaultJwtSecret      = "change-this-in-production-env-file"
	DefaultJwtExpireHours = "24"
	DefaultPasswordValue  = ""
)
