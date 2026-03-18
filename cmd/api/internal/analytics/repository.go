package analytics

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"toggly.com/m/cmd/api/internal/domain"
)

type Repository interface {
	GetProjectIDByEnvironmentID(ctx context.Context, envID uuid.UUID) (uuid.UUID, error)

	IncrementApiUsageDaily(ctx context.Context, projectID uuid.UUID, date time.Time, callsDelta int, errorsDelta int) error
	IncrementEnvEvaluationsDaily(ctx context.Context, projectID uuid.UUID, environmentID uuid.UUID, date time.Time, evaluationsDelta int) error
	IncrementFlagEvaluationsDaily(ctx context.Context, projectID uuid.UUID, flagID uuid.UUID, date time.Time, evaluationsDelta int) error

	CreateActivityEvent(ctx context.Context, evt *domain.AnalyticsActivityEvent) error

	GetProjectAnalytics(ctx context.Context, projectID uuid.UUID, rangeDays int) (*ProjectAnalyticsResponse, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) GetProjectIDByEnvironmentID(ctx context.Context, envID uuid.UUID) (uuid.UUID, error) {
	var env domain.Environment
	if err := r.db.WithContext(ctx).
		Select("project_id").
		Where("id = ?", envID).
		First(&env).Error; err != nil {
		return uuid.Nil, err
	}
	return env.ProjectID, nil
}

func (r *repository) IncrementApiUsageDaily(ctx context.Context, projectID uuid.UUID, date time.Time, callsDelta int, errorsDelta int) error {
	row := &domain.AnalyticsAPIUsageDaily{
		ProjectID:   projectID,
		Date:        date,
		CallsTotal:  callsDelta,
		ErrorsTotal: errorsDelta,
	}

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "project_id"},
			{Name: "date"},
		},
		DoUpdates: clause.Assignments(map[string]any{
			"calls_total":  gorm.Expr("calls_total + ?", callsDelta),
			"errors_total": gorm.Expr("errors_total + ?", errorsDelta),
		}),
	}).Create(row).Error
}

func (r *repository) IncrementEnvEvaluationsDaily(ctx context.Context, projectID uuid.UUID, environmentID uuid.UUID, date time.Time, evaluationsDelta int) error {
	row := &domain.AnalyticsEnvEvaluationsDaily{
		ProjectID:        projectID,
		EnvironmentID:    environmentID,
		Date:             date,
		EvaluationsTotal: evaluationsDelta,
	}

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "project_id"},
			{Name: "environment_id"},
			{Name: "date"},
		},
		DoUpdates: clause.Assignments(map[string]any{
			"evaluations_total": gorm.Expr("evaluations_total + ?", evaluationsDelta),
		}),
	}).Create(row).Error
}

func (r *repository) IncrementFlagEvaluationsDaily(ctx context.Context, projectID uuid.UUID, flagID uuid.UUID, date time.Time, evaluationsDelta int) error {
	row := &domain.AnalyticsFlagEvaluationsDaily{
		ProjectID:        projectID,
		FlagID:           flagID,
		Date:             date,
		EvaluationsTotal: evaluationsDelta,
	}

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "project_id"},
			{Name: "flag_id"},
			{Name: "date"},
		},
		DoUpdates: clause.Assignments(map[string]any{
			"evaluations_total": gorm.Expr("evaluations_total + ?", evaluationsDelta),
		}),
	}).Create(row).Error
}

func (r *repository) CreateActivityEvent(ctx context.Context, evt *domain.AnalyticsActivityEvent) error {
	return r.db.WithContext(ctx).Create(evt).Error
}

func (r *repository) GetProjectAnalytics(ctx context.Context, projectID uuid.UUID, rangeDays int) (*ProjectAnalyticsResponse, error) {
	if rangeDays <= 0 {
		return nil, fmt.Errorf("rangeDays must be > 0")
	}

	// Use UTC dates so rollups + frontend charts align.
	now := time.Now().UTC()
	todayUTC := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	currentStart := todayUTC.AddDate(0, 0, -rangeDays)
	currentEnd := todayUTC // exclusive

	previousStart := currentStart.AddDate(0, 0, -rangeDays)
	previousEnd := currentStart // exclusive

	dateKey := func(d time.Time) string { return d.Format("2006-01-02") }
	dateLabel := func(d time.Time) string { return d.Format("Jan 2") }

	deltaPct := func(current, previous float64) float64 {
		if previous == 0 {
			// Safe frontend default.
			return 0
		}
		return ((current - previous) / previous) * 100
	}

	// Fetch environment ids for consistent chart maps.
	var envs []domain.Environment
	if err := r.db.WithContext(ctx).
		Select("id").
		Where("project_id = ?", projectID).
		Find(&envs).Error; err != nil {
		return nil, fmt.Errorf("analytics: env ids: %w", err)
	}
	envIDs := make([]uuid.UUID, 0, len(envs))
	for _, e := range envs {
		envIDs = append(envIDs, e.ID)
	}

	// API usage daily series (calls + errors).
	type apiSeriesRow struct {
		Date        time.Time `gorm:"column:date"`
		CallsTotal  int       `gorm:"column:calls_total"`
		ErrorsTotal int       `gorm:"column:errors_total"`
	}
	var apiRows []apiSeriesRow
	if err := r.db.WithContext(ctx).
		Table(domain.AnalyticsAPIUsageDaily{}.TableName()).
		Select("date, calls_total, errors_total").
		Where("project_id = ? AND date >= ? AND date < ?", projectID, currentStart, currentEnd).
		Order("date ASC").
		Scan(&apiRows).Error; err != nil {
		return nil, fmt.Errorf("analytics: api usage series: %w", err)
	}
	apiByDateKey := make(map[string]apiSeriesRow, len(apiRows))
	for _, row := range apiRows {
		apiByDateKey[dateKey(row.Date)] = row
	}

	// API totals for deltas + error rate.
	type sumRow struct {
		Calls  int `gorm:"column:calls"`
		Errors int `gorm:"column:errors"`
	}
	var currSums sumRow
	if err := r.db.WithContext(ctx).
		Table(domain.AnalyticsAPIUsageDaily{}.TableName()).
		Select("COALESCE(SUM(calls_total), 0) AS calls, COALESCE(SUM(errors_total), 0) AS errors").
		Where("project_id = ? AND date >= ? AND date < ?", projectID, currentStart, currentEnd).
		Scan(&currSums).Error; err != nil {
		return nil, fmt.Errorf("analytics: api sums current: %w", err)
	}

	var prevSums sumRow
	if err := r.db.WithContext(ctx).
		Table(domain.AnalyticsAPIUsageDaily{}.TableName()).
		Select("COALESCE(SUM(calls_total), 0) AS calls, COALESCE(SUM(errors_total), 0) AS errors").
		Where("project_id = ? AND date >= ? AND date < ?", projectID, previousStart, previousEnd).
		Scan(&prevSums).Error; err != nil {
		return nil, fmt.Errorf("analytics: api sums previous: %w", err)
	}

	// Error rates.
	var currErrorRatePct float64
	if currSums.Calls > 0 {
		currErrorRatePct = (float64(currSums.Errors) / float64(currSums.Calls)) * 100
	}
	var prevErrorRatePct float64
	if prevSums.Calls > 0 {
		prevErrorRatePct = (float64(prevSums.Errors) / float64(prevSums.Calls)) * 100
	}

	// Env evaluations series.
	type envSeriesRow struct {
		Date             time.Time `gorm:"column:date"`
		EnvironmentID   uuid.UUID `gorm:"column:environment_id"`
		EvaluationsTotal int      `gorm:"column:evaluations_total"`
	}
	var envRows []envSeriesRow
	if err := r.db.WithContext(ctx).
		Table(domain.AnalyticsEnvEvaluationsDaily{}.TableName()).
		Select("date, environment_id, evaluations_total").
		Where("project_id = ? AND date >= ? AND date < ?", projectID, currentStart, currentEnd).
		Order("date ASC").
		Scan(&envRows).Error; err != nil {
		return nil, fmt.Errorf("analytics: env evaluations series: %w", err)
	}

	envByDateKey := make(map[string]map[uuid.UUID]int, len(apiRows))
	for _, row := range envRows {
		k := dateKey(row.Date)
		if envByDateKey[k] == nil {
			envByDateKey[k] = make(map[uuid.UUID]int)
		}
		envByDateKey[k][row.EnvironmentID] = row.EvaluationsTotal
	}

	// Flag evaluations totals + active flags + top flags.
	var currFlagEvalTotal int
	if err := r.db.WithContext(ctx).
		Table(domain.AnalyticsFlagEvaluationsDaily{}.TableName()).
		Select("COALESCE(SUM(evaluations_total), 0)").
		Where("project_id = ? AND date >= ? AND date < ?", projectID, currentStart, currentEnd).
		Scan(&currFlagEvalTotal).Error; err != nil {
		return nil, fmt.Errorf("analytics: flag evaluations total current: %w", err)
	}

	var prevFlagEvalTotal int
	if err := r.db.WithContext(ctx).
		Table(domain.AnalyticsFlagEvaluationsDaily{}.TableName()).
		Select("COALESCE(SUM(evaluations_total), 0)").
		Where("project_id = ? AND date >= ? AND date < ?", projectID, previousStart, previousEnd).
		Scan(&prevFlagEvalTotal).Error; err != nil {
		return nil, fmt.Errorf("analytics: flag evaluations total previous: %w", err)
	}

	var currActiveFlags int
	if err := r.db.WithContext(ctx).
		Table(domain.AnalyticsFlagEvaluationsDaily{}.TableName()).
		Select("COALESCE(COUNT(DISTINCT flag_id), 0)").
		Where("project_id = ? AND date >= ? AND date < ?", projectID, currentStart, currentEnd).
		Scan(&currActiveFlags).Error; err != nil {
		return nil, fmt.Errorf("analytics: active flags current: %w", err)
	}

	var prevActiveFlags int
	if err := r.db.WithContext(ctx).
		Table(domain.AnalyticsFlagEvaluationsDaily{}.TableName()).
		Select("COALESCE(COUNT(DISTINCT flag_id), 0)").
		Where("project_id = ? AND date >= ? AND date < ?", projectID, previousStart, previousEnd).
		Scan(&prevActiveFlags).Error; err != nil {
		return nil, fmt.Errorf("analytics: active flags previous: %w", err)
	}

	// Top flags by evaluations_total in current window.
	type flagTopRow struct {
		FlagID           uuid.UUID `gorm:"column:flag_id"`
		EvaluationsTotal int      `gorm:"column:evaluations_total"`
	}
	var topRows []flagTopRow
	if err := r.db.WithContext(ctx).
		Table(domain.AnalyticsFlagEvaluationsDaily{}.TableName()).
		Select("flag_id, SUM(evaluations_total) AS evaluations_total").
		Where("project_id = ? AND date >= ? AND date < ?", projectID, currentStart, currentEnd).
		Group("flag_id").
		Order("SUM(evaluations_total) DESC").
		Limit(5).
		Scan(&topRows).Error; err != nil {
		return nil, fmt.Errorf("analytics: top flags: %w", err)
	}

	topFlagIDs := make([]uuid.UUID, 0, len(topRows))
	for _, r := range topRows {
		topFlagIDs = append(topFlagIDs, r.FlagID)
	}

	// Previous period sums for the top flags (for delta %).
	prevByFlagID := make(map[uuid.UUID]int, len(topFlagIDs))
	if len(topFlagIDs) > 0 {
		var prevTopRows []flagTopRow
		if err := r.db.WithContext(ctx).
			Table(domain.AnalyticsFlagEvaluationsDaily{}.TableName()).
			Select("flag_id, SUM(evaluations_total) AS evaluations_total").
			Where("project_id = ? AND date >= ? AND date < ? AND flag_id IN ?", projectID, previousStart, previousEnd, topFlagIDs).
			Group("flag_id").
			Scan(&prevTopRows).Error; err != nil {
			return nil, fmt.Errorf("analytics: prev top flags: %w", err)
		}
		for _, row := range prevTopRows {
			prevByFlagID[row.FlagID] = row.EvaluationsTotal
		}
	}

	// Enrich top flags with flag key/name and "enabled" status (derived from current rule table).
	flagsByID := make(map[uuid.UUID]domain.Flag, len(topFlagIDs))
	if len(topFlagIDs) > 0 {
		var flags []domain.Flag
		if err := r.db.WithContext(ctx).
			Where("project_id = ? AND id IN ?", projectID, topFlagIDs).
			Find(&flags).Error; err != nil {
			return nil, fmt.Errorf("analytics: flags lookup: %w", err)
		}
		for _, f := range flags {
			flagsByID[f.ID] = f
		}
	}

	enabledSet := make(map[uuid.UUID]bool, len(topFlagIDs))
	if len(topFlagIDs) > 0 {
		var enabledFlagRows []struct {
			FlagID uuid.UUID `gorm:"column:flag_id"`
		}
		if err := r.db.WithContext(ctx).
			Table(domain.FlagRule{}.TableName()).
			Select("DISTINCT flag_id").
			Where("flag_id IN ? AND enabled = true", topFlagIDs).
			Scan(&enabledFlagRows).Error; err != nil {
			return nil, fmt.Errorf("analytics: enabled set lookup: %w", err)
		}
		for _, row := range enabledFlagRows {
			enabledSet[row.FlagID] = true
		}
	}

	topFlags := make([]TopFlag, 0, len(topRows))
	for _, row := range topRows {
		f := flagsByID[row.FlagID]
		prev := prevByFlagID[row.FlagID]
		delta := deltaPct(float64(row.EvaluationsTotal), float64(prev))
		// Avoid "-0" and -0.0 output.
		delta = math.Round(delta*100) / 100

		topFlags = append(topFlags, TopFlag{
			FlagId:             row.FlagID.String(),
			Key:                f.Key,
			Name:               f.Name,
			Evaluations:       row.EvaluationsTotal,
			Enabled:           enabledSet[row.FlagID],
			EvaluationsDeltaPct: delta,
		})
	}

	// Build series arrays for current window days.
	apiCallsSeries := make([]ApiCallsSeriesPoint, 0, rangeDays)
	evalsByEnvSeries := make([]EvaluationsByEnvironmentSeriesPoint, 0, rangeDays)
	for i := 0; i < rangeDays; i++ {
		d := currentStart.AddDate(0, 0, i)
		dk := dateKey(d)

		seriesRow := apiByDateKey[dk]
		apiCallsSeries = append(apiCallsSeries, ApiCallsSeriesPoint{
			Date:    dateLabel(d),
			DateKey: dk,
			Calls:   seriesRow.CallsTotal,
			Errors:  seriesRow.ErrorsTotal,
		})

		byEnv := make(map[string]int, len(envIDs))
		for _, envID := range envIDs {
			if envByDateKey[dk] != nil {
				byEnv[envID.String()] = envByDateKey[dk][envID]
			} else {
				byEnv[envID.String()] = 0
			}
		}
		evalsByEnvSeries = append(evalsByEnvSeries, EvaluationsByEnvironmentSeriesPoint{
			Date:             dateLabel(d),
			DateKey:          dk,
			ByEnvironmentId: byEnv,
		})
	}

	// Summary metrics + deltas.
	totalCallsCurrent := float64(currSums.Calls)
	totalCallsPrevious := float64(prevSums.Calls)

	flagEvalsCurrent := float64(currFlagEvalTotal)
	flagEvalsPrevious := float64(prevFlagEvalTotal)

	activeFlagsCurrentF := float64(currActiveFlags)
	activeFlagsPreviousF := float64(prevActiveFlags)

	summary := ProjectAnalyticsSummary{
		TotalCalls:            currSums.Calls,
		FlagEvaluations:       currFlagEvalTotal,
		ErrorRatePct:          math.Round(currErrorRatePct*100) / 100,
		ActiveFlags:           currActiveFlags,
		TotalCallsDeltaPct:      math.Round(deltaPct(totalCallsCurrent, totalCallsPrevious)*100) / 100,
		FlagEvaluationsDeltaPct: math.Round(deltaPct(flagEvalsCurrent, flagEvalsPrevious)*100) / 100,
		ErrorRateDeltaPct:       math.Round(deltaPct(currErrorRatePct, prevErrorRatePct)*100) / 100,
		ActiveFlagsDeltaPct:     math.Round(deltaPct(activeFlagsCurrentF, activeFlagsPreviousF)*100) / 100,
	}

	// Recent activity from analytics_activity_events.
	events, err := func() ([]domain.AnalyticsActivityEvent, error) {
		var evts []domain.AnalyticsActivityEvent
		if err := r.db.WithContext(ctx).
			Where("project_id = ?", projectID).
			Order("occurred_at DESC").
			Limit(10).
			Find(&evts).Error; err != nil {
			return nil, err
		}
		return evts, nil
	}()
	if err != nil {
		return nil, fmt.Errorf("analytics: recent activity: %w", err)
	}

	formatOccurredAt := func(t time.Time) string {
		return t.UTC().Format("2006-01-02T15:04:05.000Z")
	}

	// Resolve names for actors/envs/flags.
	envIDsInEvents := make(map[uuid.UUID]struct{})
	flagIDsInEvents := make(map[uuid.UUID]struct{})
	actorIDsInEvents := make(map[uuid.UUID]struct{})
	for _, evt := range events {
		if evt.EnvironmentID != nil {
			envIDsInEvents[*evt.EnvironmentID] = struct{}{}
		}
		if evt.FlagID != nil {
			flagIDsInEvents[*evt.FlagID] = struct{}{}
		}
		if evt.ActorID != nil {
			actorIDsInEvents[*evt.ActorID] = struct{}{}
		}
	}

	envIDList := make([]uuid.UUID, 0, len(envIDsInEvents))
	for id := range envIDsInEvents {
		envIDList = append(envIDList, id)
	}
	flagIDList := make([]uuid.UUID, 0, len(flagIDsInEvents))
	for id := range flagIDsInEvents {
		flagIDList = append(flagIDList, id)
	}
	actorIDList := make([]uuid.UUID, 0, len(actorIDsInEvents))
	for id := range actorIDsInEvents {
		actorIDList = append(actorIDList, id)
	}

	envNameByID := make(map[uuid.UUID]string, len(envIDList))
	if len(envIDList) > 0 {
		type envNameRow struct {
			ID   uuid.UUID `gorm:"column:id"`
			Name string    `gorm:"column:name"`
		}
		var rows []envNameRow
		if err := r.db.WithContext(ctx).
			Table(domain.Environment{}.TableName()).
			Select("id, name").
			Where("id IN ?", envIDList).
			Scan(&rows).Error; err != nil {
			return nil, fmt.Errorf("analytics: env names: %w", err)
		}
		for _, row := range rows {
			envNameByID[row.ID] = row.Name
		}
	}

	flagByID := make(map[uuid.UUID]domain.Flag, len(flagIDList))
	if len(flagIDList) > 0 {
		type flagNameRow struct {
			ID   uuid.UUID `gorm:"column:id"`
			Key  string    `gorm:"column:key"`
			Name string    `gorm:"column:name"`
		}
		var rows []flagNameRow
		if err := r.db.WithContext(ctx).
			Table(domain.Flag{}.TableName()).
			Select("id, key, name").
			Where("id IN ?", flagIDList).
			Scan(&rows).Error; err != nil {
			return nil, fmt.Errorf("analytics: flag keys: %w", err)
		}
		for _, row := range rows {
			flagByID[row.ID] = domain.Flag{Base: domain.Base{}, Key: row.Key, Name: row.Name, ProjectID: projectID}
		}
	}

	actorNameByID := make(map[uuid.UUID]string, len(actorIDList))
	if len(actorIDList) > 0 {
		type actorNameRow struct {
			ID   uuid.UUID `gorm:"column:id"`
			Name string    `gorm:"column:name"`
		}
		var rows []actorNameRow
		if err := r.db.WithContext(ctx).
			Table(domain.User{}.TableName()).
			Select("id, name").
			Where("id IN ?", actorIDList).
			Scan(&rows).Error; err != nil {
			return nil, fmt.Errorf("analytics: actor names: %w", err)
		}
		for _, row := range rows {
			actorNameByID[row.ID] = row.Name
		}
	}

	actionByType := func(eventType string) string {
		switch eventType {
		case "RULE_CREATED":
			return "Rule created"
		case "RULE_DELETED":
			return "Rule deleted"
		case "RULE_ENABLED":
			return "Rule enabled"
		case "RULE_DISABLED":
			return "Rule disabled"
		case "ROLLOUT_CHANGED":
			return "Rollout changed"
		case "ALLOW_LIST_UPDATED":
			return "Allow list updated"
		case "DENY_LIST_UPDATED":
			return "Deny list updated"
		case "CONDITIONS_UPDATED":
			return "Conditions updated"
		default:
			return eventType
		}
	}

	recentActivity := make([]RecentActivityItem, 0, len(events))
	for _, evt := range events {
		var who string
		if evt.ActorID != nil {
			if name, ok := actorNameByID[*evt.ActorID]; ok && name != "" {
				who = name
			} else {
				who = evt.ActorID.String()
			}
		}

		var envIDStr, envName string
		if evt.EnvironmentID != nil {
			envIDStr = evt.EnvironmentID.String()
			envName = envNameByID[*evt.EnvironmentID]
		}

		var flagIDStr, flagKey string
		if evt.FlagID != nil {
			flagIDStr = evt.FlagID.String()
			flagKey = flagByID[*evt.FlagID].Key
		}

		recentActivity = append(recentActivity, RecentActivityItem{
			EventId:         evt.ID.String(),
			OccurredAt:      formatOccurredAt(evt.OccurredAt),
			EventType:       evt.EventType,
			Who:             who,
			EnvironmentId:   envIDStr,
			EnvironmentName: envName,
			FlagId:          flagIDStr,
			FlagKey:         flagKey,
			Action:          actionByType(evt.EventType),
		})
	}

	return &ProjectAnalyticsResponse{
		ProjectId:      projectID.String(),
		RangeDays:      rangeDays,
		Summary:        summary,
		ApiCallsSeries: apiCallsSeries,
		EvaluationsByEnvironmentSeries: evalsByEnvSeries,
		TopFlags:       topFlags,
		RecentActivity: recentActivity,
	}, nil
}

