package domain

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// StringArray is a []string that maps to postgres text[].
type StringArray []string

// Scan implements sql.Scanner for postgres text[].
func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	var source []byte
	switch v := value.(type) {
	case []byte:
		source = v
	case string:
		source = []byte(v)
	default:
		return fmt.Errorf("unsupported type for StringArray: %T", value)
	}
	if len(source) == 0 {
		*s = nil
		return nil
	}
	// Postgres text[] format: {"a","b","c"} or {a,b,c}
	str := string(source)
	str = strings.TrimPrefix(str, "{")
	str = strings.TrimSuffix(str, "}")
	if str == "" {
		*s = []string{}
		return nil
	}
	var out []string
	for _, part := range strings.Split(str, ",") {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, `"`)
		out = append(out, part)
	}
	*s = out
	return nil
}

// Value implements driver.Valuer for postgres text[].
func (s StringArray) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	if len(s) == 0 {
		return "{}", nil
	}
	quoted := make([]string, len(s))
	for i, v := range s {
		quoted[i] = `"` + strings.ReplaceAll(v, `"`, `\"`) + `"`
	}
	return "{" + strings.Join(quoted, ",") + "}", nil
}

// MarshalJSON implements json.Marshaler.
func (s StringArray) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("null"), nil
	}
	return json.Marshal([]string(s))
}

// UnmarshalJSON implements json.Unmarshaler.
func (s *StringArray) UnmarshalJSON(data []byte) error {
	var slice []string
	if err := json.Unmarshal(data, &slice); err != nil {
		return fmt.Errorf("StringArray unmarshal: %w", err)
	}
	*s = slice
	return nil
}

// ConditionOperator is the logical operator for combining conditions (AND / OR).
type ConditionOperator string

const (
	OperatorAND ConditionOperator = "AND"
	OperatorOR  ConditionOperator = "OR"
)

// CompareOperator is the comparison operator for a single condition.
type CompareOperator string

const (
	CompareEq       CompareOperator = "eq"
	CompareNeq      CompareOperator = "neq"
	CompareGt       CompareOperator = "gt"
	CompareLt       CompareOperator = "lt"
	CompareGte      CompareOperator = "gte"
	CompareLte      CompareOperator = "lte"
	CompareIn       CompareOperator = "in"
	CompareNotIn    CompareOperator = "not_in"
	CompareContains CompareOperator = "contains"
)

// Condition represents a single attribute comparison.
type Condition struct {
	Attribute string         `json:"attribute"`
	Operator  CompareOperator `json:"operator"`
	Value     any            `json:"value"`
}

// ConditionGroup is a group of conditions combined by AND or OR (stored as JSONB).
type ConditionGroup struct {
	Operator   ConditionOperator `json:"operator"`
	Conditions []Condition       `json:"conditions"`
}

// Empty returns true when there are no conditions.
func (g ConditionGroup) Empty() bool {
	return len(g.Conditions) == 0
}

// Scan implements sql.Scanner for postgres JSONB.
func (g *ConditionGroup) Scan(value interface{}) error {
	if value == nil {
		*g = ConditionGroup{Conditions: []Condition{}}
		return nil
	}
	var source []byte
	switch v := value.(type) {
	case []byte:
		source = v
	case string:
		source = []byte(v)
	default:
		return fmt.Errorf("unsupported type for ConditionGroup: %T", value)
	}
	if len(source) == 0 {
		*g = ConditionGroup{Conditions: []Condition{}}
		return nil
	}
	if err := json.Unmarshal(source, g); err != nil {
		return fmt.Errorf("ConditionGroup scan: %w", err)
	}
	if g.Conditions == nil {
		g.Conditions = []Condition{}
	}
	return nil
}

// Value implements driver.Valuer for postgres JSONB.
func (g ConditionGroup) Value() (driver.Value, error) {
	if len(g.Conditions) == 0 && g.Operator == "" {
		return "{}", nil
	}
	return json.Marshal(g)
}

// Base is embedded in all models: UUID PK, timestamps, soft delete.
type Base struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	CreatedAt time.Time      `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updatedAt" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// BeforeCreate generates UUID if ID is nil (zero value).
func (b *Base) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.Must(uuid.NewV7())
	}
	return nil
}

// User is the account model.
type User struct {
	Base
	Email        string `json:"email" gorm:"size:255;uniqueIndex;not null"`
	PasswordHash string `json:"-" gorm:"size:255;not null"`
	Name         string `json:"name" gorm:"size:255;not null"`
}

// TableName returns the table name for User.
func (User) TableName() string { return "users" }

// Project belongs to a User and has many Environments and Flags.
type Project struct {
	Base
	OwnerID     uuid.UUID `json:"ownerId" gorm:"type:uuid;not null;index"`
	Name        string    `json:"name" gorm:"size:255;not null"`
	Slug        string    `json:"slug" gorm:"size:255;uniqueIndex;not null"`
	Description string    `json:"description" gorm:"type:text"`

	Environments []Environment `json:"environments,omitempty" gorm:"foreignKey:ProjectID"`
	Flags        []Flag        `json:"flags,omitempty" gorm:"foreignKey:ProjectID"`
}

// TableName returns the table name for Project.
func (Project) TableName() string { return "projects" }

// Environment belongs to a Project.
type Environment struct {
	Base
	ProjectID uuid.UUID `json:"projectId" gorm:"type:uuid;not null;uniqueIndex:idx_env_project_slug"`
	Name      string    `json:"name" gorm:"size:255;not null"`
	Slug      string    `json:"slug" gorm:"size:255;not null;uniqueIndex:idx_env_project_slug"`
	Color     string    `json:"color" gorm:"size:7;not null;default:#6366f1"`
}

// TableName returns the table name for Environment.
func (Environment) TableName() string { return "environments" }

// Flag belongs to a Project and has many FlagRules.
type Flag struct {
	Base
	ProjectID   uuid.UUID `json:"projectId" gorm:"type:uuid;not null;uniqueIndex:idx_flag_project_key"`
	Key         string    `json:"key" gorm:"size:255;not null;uniqueIndex:idx_flag_project_key"`
	Name        string    `json:"name" gorm:"size:255;not null"`
	Description string    `json:"description" gorm:"type:text"`

	Rules []FlagRule `json:"rules,omitempty" gorm:"foreignKey:FlagID"`
}

// TableName returns the table name for Flag.
func (Flag) TableName() string { return "flags" }

// FlagRule defines per-environment rule for a flag.
type FlagRule struct {
	Base
	FlagID        uuid.UUID       `json:"flagId" gorm:"type:uuid;not null;uniqueIndex:idx_flag_rule_flag_env"`
	EnvironmentID uuid.UUID       `json:"environmentId" gorm:"type:uuid;not null;uniqueIndex:idx_flag_rule_flag_env"`
	Enabled       bool            `json:"enabled" gorm:"not null;default:false"`
	RolloutPct    int             `json:"rolloutPct" gorm:"not null;default:0;check:rollout_pct >= 0 AND rollout_pct <= 100"`
	AllowList     StringArray     `json:"allowList" gorm:"type:text[]"`
	DenyList      StringArray     `json:"denyList" gorm:"type:text[]"`
	Conditions    ConditionGroup   `json:"conditions" gorm:"type:jsonb;default:'{}'"`
	UpdatedBy     uuid.UUID       `json:"updatedBy" gorm:"type:uuid;not null"`

	// Flag is populated when preloading (e.g. in evaluation cache). Not stored as a column.
	Flag *Flag `json:"flag,omitempty" gorm:"foreignKey:FlagID"`
}

// TableName returns the table name for FlagRule.
func (FlagRule) TableName() string { return "flag_rules" }

// APIKey is a secret key scoped to an environment (e.g. for SDK / evaluation).
type APIKey struct {
	Base
	EnvironmentID uuid.UUID   `json:"environmentId" gorm:"type:uuid;not null;index"`
	Environment   Environment `json:"environment,omitempty" gorm:"foreignKey:EnvironmentID"`
	Name          string      `json:"name" gorm:"size:255;not null"`
	Prefix        string      `json:"prefix" gorm:"size:16;not null;uniqueIndex:idx_api_key_prefix"`
	KeyHash       string      `json:"-" gorm:"size:255;not null"`
	LastUsedAt    *time.Time  `json:"lastUsedAt" gorm:"type:timestamptz"`
}

// TableName returns the table name for APIKey.
func (APIKey) TableName() string { return "api_keys" }
