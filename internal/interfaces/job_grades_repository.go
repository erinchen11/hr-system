package interfaces

import (
	"context"

	"github.com/erinchen11/hr-system/internal/models"
	"github.com/google/uuid"
)

type JobGradeRepository interface {
	CreateJobGrade(ctx context.Context, jobGrade *models.JobGrade) error
	GetJobGradeByID(ctx context.Context, id uuid.UUID) (*models.JobGrade, error)
	GetJobGradeByCode(ctx context.Context, code string) (*models.JobGrade, error)
	UpdateJobGrade(ctx context.Context, jobGrade *models.JobGrade) error
	ListJobGrades(ctx context.Context) ([]models.JobGrade, error)
	DeleteJobGrade(ctx context.Context, id uuid.UUID) error
}
