package evaluation

// Reason describes why a flag resolved to its value.
type Reason string

const (
	ReasonDisabled   Reason = "flag_disabled"
	ReasonDenyList   Reason = "deny_list"
	ReasonAllowList  Reason = "allow_list"
	ReasonConditions Reason = "conditions_not_met"
	ReasonRollout    Reason = "rollout"
	ReasonEnabled    Reason = "flag_enabled"
	ReasonExperiment Reason = "experiment"
)

// EvaluationResult is returned by the engine and passed up through service → handler.
type EvaluationResult struct {
	FlagKey      string `json:"flagKey"`
	Enabled      bool   `json:"enabled"`
	Reason       string `json:"reason"`
	Variant      string `json:"variant,omitempty"`
	ExperimentID string `json:"experimentId,omitempty"`
}
