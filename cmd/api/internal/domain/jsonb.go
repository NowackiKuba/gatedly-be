package domain

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// JSONMap is a small helper type for storing arbitrary JSONB payloads in Postgres.
// It implements both driver.Valuer and sql.Scanner semantics via GORM.
type JSONMap map[string]any

func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return []byte(`{}`), nil
	}
	b, err := json.Marshal(j)
	if err != nil {
		return nil, fmt.Errorf("JSONMap marshal: %w", err)
	}
	return b, nil
}

func (j *JSONMap) Scan(value any) error {
	if value == nil {
		*j = JSONMap{}
		return nil
	}

	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("JSONMap scan: unsupported type %T", value)
	}

	if len(b) == 0 {
		*j = JSONMap{}
		return nil
	}

	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return fmt.Errorf("JSONMap unmarshal: %w", err)
	}
	*j = out
	return nil
}

