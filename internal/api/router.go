package api

import (
	handlers "github.com/erinchen11/hr-system/internal/api/handlers"
	account "github.com/erinchen11/hr-system/internal/api/handlers/account"
	auth "github.com/erinchen11/hr-system/internal/api/handlers/auth"
	jobgradehandler "github.com/erinchen11/hr-system/internal/api/handlers/job_grade"
	leaverequest "github.com/erinchen11/hr-system/internal/api/handlers/leave_request"

	"github.com/erinchen11/hr-system/internal/api/middleware"
	"github.com/gin-gonic/gin"
)

// *** RegisterRoutesV1 函數簽名已更新 ***
func RegisterRoutesV1(
	rg *gin.RouterGroup,

	// Handlers - 注入來自不同子套件的 Handler 實例
	checkLiveHandler *handlers.CheckLiveHandler, // 假設仍在 handlers 層級
	loginHandler *auth.LoginHandler, // <--- 使用 auth.
	authMiddleware *middleware.AuthMiddleware,
	accountPasswordHandler *account.AccountPasswordHandler, // <--- 使用 account. (假設 Handler 內結構體名未改)
	userCreationHandler *account.AccountCreationHandler, // <--- 使用 account. (假設 Handler 內結構體名未改)
	listLeaveRequestsHandler *leaverequest.ListLeaveRequestsHandler, // <--- 使用 leave_request.
	approveLeaveRequestHandler *leaverequest.ApproveLeaveRequestHandler, // <--- 使用 leave_request.
	rejectLeaveRequestHandler *leaverequest.RejectLeaveRequestHandler, // <--- 使用 leave_request.
	userProfileHandler *account.UserProfileHandler, // <--- 使用 account. (假設 Handler 內結構體名未改)
	applyLeaveHandler *leaverequest.ApplyLeaveHandler, // <--- 使用 leave_request.
	viewLeaveStatusHandler *leaverequest.ViewLeaveStatusHandler, // <--- 使用 leave_request.
	listJobGradesHandler *jobgradehandler.ListJobGradesHandler,

) {
	// --- 路由註冊邏輯保持不變 ---

	// 無需登入的
	rg.GET("/check-live", checkLiveHandler.CheckLive)
	rg.POST("/login", loginHandler.Login)

	// 需要登入後的
	protected := rg.Group("/")
	protected.Use(authMiddleware.Authenticate())
	{
		// --- 通用功能 ---
		protected.POST("/change-password", accountPasswordHandler.ChangePassword)
		protected.POST("/account/create", userCreationHandler.CreateUser) // 統一用戶創建入口

		// --- 特定角色 API ---

		// HR APIs
		hr := protected.Group("/hr")
		{
			hr.GET("/job-grades", listJobGradesHandler.ListJobGrades) 

			hr.GET("/leave-requests", listLeaveRequestsHandler.ListLeaveRequests)
			hr.POST("/leave-requests/:id/approve", approveLeaveRequestHandler.ApproveLeaveRequest)
			hr.POST("/leave-requests/:id/reject", rejectLeaveRequestHandler.RejectLeaveRequest)
		}

		// Employee APIs
		employee := protected.Group("/employee")
		{
			employee.GET("/profile", userProfileHandler.GetProfile)
			employee.POST("/apply-leave", applyLeaveHandler.ApplyLeave)
			employee.GET("/leave-status", viewLeaveStatusHandler.ViewLeaveStatus)
		}

		// Super User APIs (可選)
		// super := protected.Group("/super")
		// super.Use(middleware.AuthorizeRole(models.RoleSuperAdmin))
		// {
		// 可能有其他 Super Admin 專屬 API
		// }
	}
}
