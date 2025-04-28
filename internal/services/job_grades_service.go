package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/erinchen11/hr-system/internal/interfaces"
	"github.com/erinchen11/hr-system/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// --- Service 層錯誤 ---
var (
	ErrJobGradeNotFound      = errors.New("job grade not found")
	ErrJobGradeCodeExists    = errors.New("job grade code already exists")
	ErrJobGradeInUse         = errors.New("job grade is currently in use by employees and cannot be deleted")
	ErrJobGradeUpdateFailed  = errors.New("failed to update job grade")
	ErrJobGradeCreateFailed  = errors.New("failed to create job grade")
	ErrJobGradeDeleteFailed  = errors.New("failed to delete job grade")
	ErrInvalidInput          = errors.New("invalid input data")
	ErrEmploymentCheckFailed = errors.New("failed to check employment usage for job grade")
)

// jobGradeServiceImpl 實現了 JobGradeService 介面
type jobGradeServiceImpl struct {
	jobGradeRepo   interfaces.JobGradeRepository
	employmentRepo interfaces.EmploymentRepository // *** 用於檢查職等是否被使用 ***
}

// NewJobGradeServiceImpl 構造函數
func NewJobGradeServiceImpl(
	jobGradeRepo interfaces.JobGradeRepository,
	employmentRepo interfaces.EmploymentRepository,
) interfaces.JobGradeService { // 返回 JobGradeService 介面
	return &jobGradeServiceImpl{
		jobGradeRepo:   jobGradeRepo,
		employmentRepo: employmentRepo,
	}
}

// CreateJobGrade 創建新的職等記錄
func (s *jobGradeServiceImpl) CreateJobGrade(ctx context.Context, jobGrade *models.JobGrade) (*models.JobGrade, error) {
	if jobGrade == nil || strings.TrimSpace(jobGrade.Code) == "" || strings.TrimSpace(jobGrade.Name) == "" {
		return nil, fmt.Errorf("%w: job grade code and name cannot be empty", ErrInvalidInput)
	}

	existing, err := s.jobGradeRepo.GetJobGradeByCode(ctx, jobGrade.Code)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("Error checking job grade code existence for %s: %v", jobGrade.Code, err)
		return nil, fmt.Errorf("database error checking job grade code: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("%w: code %s", ErrJobGradeCodeExists, jobGrade.Code)
	}

	jobGrade.ID = uuid.Nil
	err = s.jobGradeRepo.CreateJobGrade(ctx, jobGrade)
	if err != nil {
		log.Printf("Error creating job grade with code %s: %v", jobGrade.Code, err)
		return nil, fmt.Errorf("%w: %w", ErrJobGradeCreateFailed, err)
	}
	return jobGrade, nil
}

// GetJobGradeByID 根據 ID 獲取職等記錄
func (s *jobGradeServiceImpl) GetJobGradeByID(ctx context.Context, id uuid.UUID) (*models.JobGrade, error) {
	jobGrade, err := s.jobGradeRepo.GetJobGradeByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrJobGradeNotFound
		}
		log.Printf("Error getting job grade by ID %s: %v", id, err)
		return nil, fmt.Errorf("database error fetching job grade by ID: %w", err)
	}
	return jobGrade, nil
}

// GetJobGradeByCode 根據職等代碼 (Code) 獲取職等記錄
func (s *jobGradeServiceImpl) GetJobGradeByCode(ctx context.Context, code string) (*models.JobGrade, error) {
	if strings.TrimSpace(code) == "" {
		return nil, fmt.Errorf("%w: job grade code cannot be empty", ErrInvalidInput)
	}
	jobGrade, err := s.jobGradeRepo.GetJobGradeByCode(ctx, code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrJobGradeNotFound
		}
		log.Printf("Error getting job grade by Code %s: %v", code, err)
		return nil, fmt.Errorf("database error fetching job grade by code: %w", err)
	}
	return jobGrade, nil
}

// UpdateJobGrade 更新現有的職等記錄
func (s *jobGradeServiceImpl) UpdateJobGrade(ctx context.Context, id uuid.UUID, updates *models.JobGrade) (*models.JobGrade, error) {
	existingGrade, err := s.jobGradeRepo.GetJobGradeByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrJobGradeNotFound
		}
		log.Printf("Error fetching job grade %s for update: %v", id, err)
		return nil, fmt.Errorf("failed to retrieve job grade for update: %w", err)
	}

	needsUpdate := false
	if updates.Code != "" && updates.Code != existingGrade.Code {
		_, checkErr := s.jobGradeRepo.GetJobGradeByCode(ctx, updates.Code)
		if checkErr == nil {
			return nil, fmt.Errorf("%w: code %s", ErrJobGradeCodeExists, updates.Code)
		}
		if !errors.Is(checkErr, gorm.ErrRecordNotFound) {
			log.Printf("Error checking new job grade code %s during update: %v", updates.Code, checkErr)
			return nil, fmt.Errorf("database error checking new job grade code: %w", checkErr)
		}
		existingGrade.Code = updates.Code
		needsUpdate = true
	}
	if updates.Name != "" && updates.Name != existingGrade.Name {
		existingGrade.Name = updates.Name
		needsUpdate = true
	}
	if updates.Description != "" && (existingGrade.Description == "" || updates.Description != existingGrade.Description) {
		existingGrade.Description = updates.Description
		needsUpdate = true
	}
	if !updates.MinSalary.IsZero() && !updates.MinSalary.Equals(existingGrade.MinSalary) {
		existingGrade.MinSalary = updates.MinSalary
		needsUpdate = true
	}
	if !updates.MaxSalary.IsZero() && !updates.MaxSalary.Equals(existingGrade.MaxSalary) {
		existingGrade.MaxSalary = updates.MaxSalary
		needsUpdate = true
	}

	if !needsUpdate {
		return existingGrade, nil
	}

	err = s.jobGradeRepo.UpdateJobGrade(ctx, existingGrade)
	if err != nil {
		log.Printf("Error updating job grade %s in repository: %v", id, err)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrJobGradeNotFound
		}
		return nil, fmt.Errorf("%w: %w", ErrJobGradeUpdateFailed, err)
	}
	return existingGrade, nil
}

// ListJobGrades 列出所有職等記錄
func (s *jobGradeServiceImpl) ListJobGrades(ctx context.Context) ([]models.JobGrade, error) {
	// 依賴 jobGradeRepo 提供了 ListJobGrades 方法
	jobGrades, err := s.jobGradeRepo.ListJobGrades(ctx)
	if err != nil {
		log.Printf("Error listing job grades from repository: %v", err)
		return nil, fmt.Errorf("failed to list job grades: %w", err)
	}
	return jobGrades, nil
}

// DeleteJobGrade 刪除職等記錄 (*** 確保這個方法存在 ***)
func (s *jobGradeServiceImpl) DeleteJobGrade(ctx context.Context, id uuid.UUID) error {
	// 1. 檢查是否有員工正在使用此職等
	// *** 假設 EmploymentRepository 提供了 GetEmploymentCountByJobGradeID 方法 ***
	count, err := s.employmentRepo.GetEmploymentCountByJobGradeID(ctx, id)
	if err != nil {
		log.Printf("Error checking employment usage for job grade %s: %v", id, err)
		return fmt.Errorf("%w: %w", ErrEmploymentCheckFailed, err)
	}
	if count > 0 {
		log.Printf("Attempted to delete job grade %s which is still in use by %d employment records", id, count)
		return ErrJobGradeInUse
	}

	// 2. 調用 Repository 刪除
	// *** 依賴 jobGradeRepo 提供了 DeleteJobGrade 方法 ***
	err = s.jobGradeRepo.DeleteJobGrade(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrJobGradeNotFound
		}
		log.Printf("Error deleting job grade %s from repository: %v", id, err)
		return fmt.Errorf("%w: %w", ErrJobGradeDeleteFailed, err)
	}

	log.Printf("Job grade %s deleted successfully.", id)
	return nil
}

