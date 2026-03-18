package analytics

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"toggly.com/m/cmd/api/internal/domain"
)

type Service struct {
	repo Repository
}

func NewService(db *gorm.DB) *Service {
	return &Service{repo: NewRepository(db)}
}

func NewServiceWithRepo(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetProjectIDByEnvironmentID(ctx context.Context, envID uuid.UUID) (uuid.UUID, error) {
	return s.repo.GetProjectIDByEnvironmentID(ctx, envID)
}

func (s *Service) IncrementApiUsageDaily(ctx context.Context, projectID uuid.UUID, date time.Time, callsDelta int, errorsDelta int) error {
	return s.repo.IncrementApiUsageDaily(ctx, projectID, date, callsDelta, errorsDelta)
}

func (s *Service) IncrementEnvEvaluationsDaily(ctx context.Context, projectID uuid.UUID, environmentID uuid.UUID, date time.Time, evaluationsDelta int) error {
	return s.repo.IncrementEnvEvaluationsDaily(ctx, projectID, environmentID, date, evaluationsDelta)
}

func (s *Service) IncrementFlagEvaluationsDaily(ctx context.Context, projectID uuid.UUID, flagID uuid.UUID, date time.Time, evaluationsDelta int) error {
	return s.repo.IncrementFlagEvaluationsDaily(ctx, projectID, flagID, date, evaluationsDelta)
}

func (s *Service) CreateActivityEvent(ctx context.Context, evt *domain.AnalyticsActivityEvent) error {
	return s.repo.CreateActivityEvent(ctx, evt)
}

func (s *Service) GetProjectAnalytics(ctx context.Context, projectID uuid.UUID, rangeDays int) (*ProjectAnalyticsResponse, error) {
	return s.repo.GetProjectAnalytics(ctx, projectID, rangeDays)
}

