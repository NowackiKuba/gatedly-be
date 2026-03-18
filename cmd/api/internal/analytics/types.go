package analytics

// ProjectAnalyticsResponse matches the frontend contract in docs/backend-frontend-analytics-integration.mdc.
type ProjectAnalyticsResponse struct {
	ProjectId      string                            `json:"projectId"`
	RangeDays      int                               `json:"rangeDays"`
	Summary        ProjectAnalyticsSummary           `json:"summary"`
	ApiCallsSeries []ApiCallsSeriesPoint            `json:"apiCallsSeries"`
	EvaluationsByEnvironmentSeries []EvaluationsByEnvironmentSeriesPoint `json:"evaluationsByEnvironmentSeries"`
	TopFlags       []TopFlag                        `json:"topFlags"`
	RecentActivity []RecentActivityItem            `json:"recentActivity"`
}

type ProjectAnalyticsSummary struct {
	TotalCalls            int     `json:"totalCalls"`
	FlagEvaluations       int     `json:"flagEvaluations"`
	ErrorRatePct          float64 `json:"errorRatePct"`
	ActiveFlags           int     `json:"activeFlags"`

	TotalCallsDeltaPct      float64 `json:"totalCallsDeltaPct"`
	FlagEvaluationsDeltaPct float64 `json:"flagEvaluationsDeltaPct"`
	ErrorRateDeltaPct       float64 `json:"errorRateDeltaPct"`
	ActiveFlagsDeltaPct     float64 `json:"activeFlagsDeltaPct"`
}

type ApiCallsSeriesPoint struct {
	Date    string `json:"date"`
	DateKey string `json:"dateKey"`
	Calls   int    `json:"calls"`
	Errors  int    `json:"errors"`
}

type EvaluationsByEnvironmentSeriesPoint struct {
	Date             string             `json:"date"`
	DateKey          string             `json:"dateKey"`
	ByEnvironmentId map[string]int     `json:"byEnvironmentId"`
}

type TopFlag struct {
	FlagId             string  `json:"flagId"`
	Key                string  `json:"key"`
	Name               string  `json:"name"`
	Evaluations       int     `json:"evaluations"`
	Enabled           bool    `json:"enabled"`
	EvaluationsDeltaPct float64 `json:"evaluationsDeltaPct"`
}

type RecentActivityItem struct {
	EventId          string `json:"eventId"`
	OccurredAt       string `json:"occurredAt"`
	EventType        string `json:"eventType"`
	Who               string `json:"who"`

	EnvironmentId    string `json:"environmentId"`
	EnvironmentName  string `json:"environmentName"`

	FlagId           string `json:"flagId"`
	FlagKey          string `json:"flagKey"`

	Action           string `json:"action"`
}

