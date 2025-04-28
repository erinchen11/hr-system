package environment

// --- Configuration Variables ---
// 這些變數將由 config.LoadConfig() 根據 .env 檔案和 OS 環境變數來填充。
var (
	Environment string // local, docker, production
	GinMode     string // debug, release

	// 伺服器
	ServerPort string
	Domain     string

	// 資料庫 (保持為 string，在需要的地方轉換)
	DatabaseUser         string
	DatabasePassword     string
	DatabaseName         string
	DatabaseHost         string
	DatabasePort         string
	DatabaseLogLevel     string
	DatabaseMaxIdleConns string
	DatabaseMaxOpenConns string
	DatabaseMaxLifetime  string // second

	// Redis (保持為 string)
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
