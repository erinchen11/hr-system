package interfaces

import (
	"context"

	"github.com/erinchen11/hr-system/internal/models"
	"github.com/google/uuid"
)

// JobGradeService 定義了與職等相關的業務邏輯操作
type JobGradeService interface {
	CreateJobGrade(ctx context.Context, jobGrade *models.JobGrade) (*models.JobGrade, error)
	GetJobGradeByID(ctx context.Context, id uuid.UUID) (*models.JobGrade, error)
	GetJobGradeByCode(ctx context.Context, code string) (*models.JobGrade, error)
	UpdateJobGrade(ctx context.Context, id uuid.UUID, updates *models.JobGrade) (*models.JobGrade, error) // 接收 ID 和更新資料
	ListJobGrades(ctx context.Context) ([]models.JobGrade, error)
	DeleteJobGrade(ctx context.Context, id uuid.UUID) error // Service 層應包含刪除前的業務檢查
}
