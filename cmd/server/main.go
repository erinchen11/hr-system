// 檔案路徑: cmd/server/main.go
package main

import (
	"flag"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/erinchen11/hr-system/environment"  // 訪問配置變數
	"github.com/erinchen11/hr-system/internal/api" // 路由註冊
	"github.com/gin-contrib/cors"

	"github.com/erinchen11/hr-system/internal/api/handlers"                            // 頂層 handlers (如果 CheckLive 在這裡)
	acchandler "github.com/erinchen11/hr-system/internal/api/handlers/account"         // 使用別名 account handler
	authhandler "github.com/erinchen11/hr-system/internal/api/handlers/auth"           // 使用別名 auth handler
	jobgradehandler "github.com/erinchen11/hr-system/internal/api/handlers/job_grade"  // 導入 jobgrade
	leavehandler "github.com/erinchen11/hr-system/internal/api/handlers/leave_request" // 使用別名 leave handler

	"github.com/erinchen11/hr-system/internal/api/middleware" // Middleware 實現
	"github.com/erinchen11/hr-system/internal/config"         // 調用 LoadConfig
	"github.com/erinchen11/hr-system/internal/infra/cache"    // Cache 初始化和 Repository
	"github.com/erinchen11/hr-system/internal/infra/database" // DB 初始化和 Repository

	// 導入 interfaces
	"github.com/erinchen11/hr-system/internal/seeds"    // Seeds
	"github.com/erinchen11/hr-system/internal/services" // Service 實現
	"github.com/erinchen11/hr-system/internal/utils"    // Utilities 實現
	"github.com/gin-gonic/gin"                          // 導入 Gin
	"github.com/redis/go-redis/v9"                      // 導入 Redis Client

	"gorm.io/gorm" // 導入 GORM
)

func main() {
	// --- flag 參數解析 ---
	migrate := flag.Bool("migrate", false, "Run database migrations")
	seed := flag.Bool("seed", false, "Seed the database with initial data")
	flag.Parse()

	// --- 1. 加載配置 ---
	config.LoadConfig()

	// --- 2. 初始化基礎設施 ---
	log.Println("Initializing infrastructure...")
	db := initializeDatabase()
	redisClient := initializeRedis()
	engine := initializeGin()
	log.Println("Infrastructure initialized.")

	// --- Migration & Seed ---
	if *migrate {
		log.Println("Running database migrations...")
		if err := database.RunMigrations(db); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		log.Println("Migrations completed.")
		os.Exit(0)
	}
	if *seed {
		log.Println("Running database seeding...")
		if err := seeds.Run(db); err != nil {
			log.Fatalf("Seeding failed: %v", err)
		}
		log.Println("Seeding completed.")
		os.Exit(0)
	}

	// --- 3. 依賴注入設置 ---
	log.Println("Initializing dependencies...")

	// 3.1 實例化 Repositories
	log.Println("Initializing repositories...")
	accountRepo := database.NewGormAccountRepository(db)
	employmentRepo := database.NewGormEmploymentRepository(db)
	leaveRequestRepo := database.NewGormLeaveRequestRepository(db)
	jobGradeRepo := database.NewGormJobGradeRepository(db)
	cacheRepo := cache.NewRedisCacheRepository(redisClient)
	log.Println("Repositories initialized.")

	// 3.2 實例化 Utilities / Helpers
	log.Println("Initializing utilities...")
	pwChecker := utils.NewBcryptPasswordChecker()
	pwHasher := utils.NewBcryptPasswordHasher()
	jwtSecret := environment.JwtSecret
	jwtIssuer := "hr-system-api"
	jwtExpireHoursStr := environment.JwtExpireHours
	jwtExpireHours, err := strconv.Atoi(jwtExpireHoursStr)
	if err != nil {
		log.Printf("Warning: Invalid JWT_EXPIRE_HOURS '%s', using default 24 hours. Error: %v", jwtExpireHoursStr, err)
		jwtExpireHours = 24
	}
	jwtHelper, err := utils.NewJwtUtils(jwtSecret, jwtIssuer, jwtExpireHoursStr)
	if err != nil {
		log.Fatalf("Failed to initialize JWT Utils: %v", err)
	}
	defaultPassword := environment.DefaultPassword
	log.Println("Utilities initialized.")

	// 3.3 實例化 Services
	log.Println("Initializing services...")
	authService := services.NewAuthServiceImpl(accountRepo, pwChecker)
	tokenService := services.NewTokenServiceImpl(cacheRepo, jwtHelper, jwtHelper, jwtExpireHours)
	accountService := services.NewAccountServiceImpl(
		accountRepo, employmentRepo, pwChecker, pwHasher, cacheRepo, defaultPassword, db,
	)
	employmentService := services.NewEmploymentServiceImpl(
		employmentRepo, accountRepo,
	)
	leaveRequestService := services.NewLeaveRequestServiceImpl(
		leaveRequestRepo, accountRepo,
	)
	jobGradeService := services.NewJobGradeServiceImpl(jobGradeRepo, employmentRepo) // 實例化 JobGradeService

	log.Println("Services initialized.")

	// 3.4 實例化 Handlers
	log.Println("Initializing handlers...")
	checkLiveHandler := handlers.NewCheckLiveHandler()
	loginHandler := authhandler.NewLoginHandler(authService, tokenService)
	accountPasswordHandler := acchandler.NewAccountPasswordHandler(accountService)            // 使用 accountService
	userCreationHandler := acchandler.NewAccountCreationHandler(accountService)               // 使用 accountService
	userProfileHandler := acchandler.NewUserProfileHandler(accountService, employmentService) // 使用 accountService 和 employmentService
	listLeaveRequestsHandler := leavehandler.NewListLeaveRequestsHandler(leaveRequestService)
	approveLeaveRequestHandler := leavehandler.NewApproveLeaveRequestHandler(leaveRequestService)
	rejectLeaveRequestHandler := leavehandler.NewRejectLeaveRequestHandler(leaveRequestService)
	applyLeaveHandler := leavehandler.NewApplyLeaveHandler(leaveRequestService)
	viewLeaveStatusHandler := leavehandler.NewViewLeaveStatusHandler(leaveRequestService)
	listJobGradesHandler := jobgradehandler.NewListJobGradesHandler(jobGradeService) // 新增: 創建 ListJobGradesHandler
	log.Println("Handlers initialized.")

	// 3.5 實例化 Middleware
	authMiddleware := middleware.NewAuthMiddleware(tokenService)
	log.Println("Middleware initialized.")

	log.Println("Dependencies initialized.")

	// --- 4. 設置路由 ---
	log.Println("Registering routes...")
	v1 := engine.Group(environment.BasePath + "/v1")

	//傳遞更新後的 Handler 實例列表
	api.RegisterRoutesV1(
		v1,
		checkLiveHandler,
		loginHandler,               // auth.LoginHandler
		authMiddleware,             // middleware
		accountPasswordHandler,     // account.AccountPasswordHandler
		userCreationHandler,        // account.UserCreationHandler
		listLeaveRequestsHandler,   // leave_request.ListLeaveRequestsHandler
		approveLeaveRequestHandler, // leave_request.ApproveLeaveRequestHandler
		rejectLeaveRequestHandler,  // leave_request.RejectLeaveRequestHandler
		userProfileHandler,         // account.UserProfileHandler
		applyLeaveHandler,          // leave_request.ApplyLeaveHandler
		viewLeaveStatusHandler,     // leave_request.ViewLeaveStatusHandler
		listJobGradesHandler,
	)
	log.Println("Routes registered.")

	// --- 5. 啟動 HTTP Server ---
	serverPort := environment.ServerPort
	log.Printf("🚀 Starting server on port %s...", serverPort)
	if err := engine.Run(":" + serverPort); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// --- 輔助函數：初始化資料庫, Redis, Gin---
func initializeDatabase() *gorm.DB {
	log.Println("Initializing database...")
	dbPort, err := strconv.Atoi(environment.DatabasePort)
	if err != nil {
		log.Printf("Warning: Invalid MYSQL_PORT '%s', using default 3306. Error: %v", environment.DatabasePort, err)
		dbPort = 3306
	}
	maxIdleConns, _ := strconv.Atoi(environment.DatabaseMaxIdleConns)
	maxOpenConns, _ := strconv.Atoi(environment.DatabaseMaxOpenConns)
	maxLifetimeSeconds, _ := strconv.Atoi(environment.DatabaseMaxLifetime)

	dbCfg := database.DatabaseConfig{User: environment.DatabaseUser, Passwd: environment.DatabasePassword, Host: environment.DatabaseHost, Port: strconv.Itoa(dbPort), DbName: environment.DatabaseName, MaxIdleConns: maxIdleConns, MaxOpenConns: maxOpenConns, MaxLifetime: maxLifetimeSeconds}
	db, err := database.InitializeDB(dbCfg)
	if err != nil {
		log.Fatalf("DB Initialization Failed: %v", err)
	}
	log.Println("Database initialized successfully.")
	return db
}
func initializeRedis() *redis.Client { /* ... */
	log.Println("Initializing Redis...")
	redisDB, err := strconv.Atoi(environment.RedisDB)
	if err != nil {
		log.Printf("Warning: Invalid REDIS_DB '%s', using default 0. Error: %v", environment.RedisDB, err)
		redisDB = 0
	}
	redisCfg := cache.RedisConfig{Addr: environment.RedisAddr, Passwd: environment.RedisPassword, DB: redisDB}
	rdb, err := cache.InitializeCache(redisCfg)
	if err != nil {
		log.Fatalf("Redis Initialization Failed: %v", err)
	}
	log.Println("Redis initialized successfully.")
	return rdb
}
func initializeGin() *gin.Engine { /* ... */
	log.Println("Initializing Gin engine...")
	if environment.GinMode == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}
	engine := gin.Default()
	// --- CORS ---
	engine.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // 允許所有來源, 也可以改成 []string{"http://localhost:3000"} 指定
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	log.Printf("Gin engine initialized in %s mode.", gin.Mode())
	return engine
}
