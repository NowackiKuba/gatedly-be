package domain

import (
	"time"

	"gorm.io/gorm"

	"github.com/google/uuid"
)

// TableName is required so we can control exact Postgres table names.
type AnalyticsAPIUsageDaily struct {
	Base
	ProjectID  uuid.UUID `json:"projectId" gorm:"type:uuid;not null;index;uniqueIndex:idx_analytics_api_usage_daily_project_date"`
	Date       time.Time `json:"date" gorm:"type:date;not null;uniqueIndex:idx_analytics_api_usage_daily_project_date"`
	CallsTotal int       `json:"callsTotal" gorm:"not null;default:0"`
	ErrorsTotal int      `json:"errorsTotal" gorm:"not null;default:0"`
}

func (AnalyticsAPIUsageDaily) TableName() string { return "analytics_api_usage_daily" }

type AnalyticsEnvEvaluationsDaily struct {
	Base
	ProjectID       uuid.UUID `json:"projectId" gorm:"type:uuid;not null;index;uniqueIndex:idx_analytics_env_evaluations_daily_project_env_date"`
	EnvironmentID   uuid.UUID `json:"environmentId" gorm:"type:uuid;not null;uniqueIndex:idx_analytics_env_evaluations_daily_project_env_date"`
	Date            time.Time `json:"date" gorm:"type:date;not null;uniqueIndex:idx_analytics_env_evaluations_daily_project_env_date"`
	EvaluationsTotal int      `json:"evaluationsTotal" gorm:"not null;default:0"`
}

func (AnalyticsEnvEvaluationsDaily) TableName() string { return "analytics_env_evaluations_daily" }

type AnalyticsFlagEvaluationsDaily struct {
	Base
	ProjectID       uuid.UUID `json:"projectId" gorm:"type:uuid;not null;index;uniqueIndex:idx_analytics_flag_evaluations_daily_project_flag_date"`
	FlagID          uuid.UUID `json:"flagId" gorm:"type:uuid;not null;uniqueIndex:idx_analytics_flag_evaluations_daily_project_flag_date"`
	Date            time.Time `json:"date" gorm:"type:date;not null;uniqueIndex:idx_analytics_flag_evaluations_daily_project_flag_date"`
	EvaluationsTotal int      `json:"evaluationsTotal" gorm:"not null;default:0"`
}

func (AnalyticsFlagEvaluationsDaily) TableName() string { return "analytics_flag_evaluations_daily" }

// AnalyticsActivityEvent stores recent activity entries for the frontend "recent activity" feed.
// Notes:
// - We use OccurredAt as the canonical event timestamp (stored in DB as occurred_at).
// - The JSON payload is intentionally flexible to support different PATCH fields.
type AnalyticsActivityEvent struct {
	ID uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`

	ProjectID     uuid.UUID `json:"projectId" gorm:"type:uuid;not null;index"`
	EnvironmentID *uuid.UUID `json:"environmentId,omitempty" gorm:"type:uuid"`
	FlagID        *uuid.UUID `json:"flagId,omitempty" gorm:"type:uuid"`
	RuleID        *uuid.UUID `json:"ruleId,omitempty" gorm:"type:uuid"`

	EventType string    `json:"eventType" gorm:"size:64;not null;index"`
	ActorID   *uuid.UUID `json:"actorId,omitempty" gorm:"type:uuid"`

	OccurredAt time.Time `json:"occurredAt" gorm:"column:occurred_at;not null;index"`
	Payload    JSONMap   `json:"payload" gorm:"type:jsonb;default:'{}'"`
}

func (AnalyticsActivityEvent) TableName() string { return "analytics_activity_events" }

func (e *AnalyticsActivityEvent) BeforeCreate(tx *gorm.DB) error {
	// We intentionally do not rely on domain.Base here (we don't want extra created_at/updated_at columns).
	if e.ID == uuid.Nil {
		e.ID = uuid.Must(uuid.NewV7())
	}
	return nil
}

