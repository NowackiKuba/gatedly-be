package evaluation

import (
	"fmt"
	"hash/fnv"
	"strings"

	"toggly.com/m/cmd/api/internal/domain"
)

func Evaluate(rule domain.FlagRule, userID string, attributes map[string]any) EvaluationResult {
	if !rule.Enabled {
		return EvaluationResult{Enabled: false, Reason: ReasonDisabled}
	}

	for _, id := range rule.DenyList {
		if id == userID {
			return EvaluationResult{Enabled: false, Reason: ReasonDenyList}
		}
	}

	for _, id := range rule.AllowList {
		if id == userID {
			return EvaluationResult{Enabled: true, Reason: ReasonAllowList}
		}
	}

	if !evaluateConditions(rule.Conditions, attributes) {
		return EvaluationResult{Enabled: false, Reason: ReasonConditions}
	}

	if rule.RolloutPct >= 100 {
		return EvaluationResult{Enabled: true, Reason: ReasonEnabled}
	}

	if rule.RolloutPct > 0 && bucket(userID) < rule.RolloutPct {
		return EvaluationResult{Enabled: true, Reason: ReasonRollout}
	}

	return EvaluationResult{Enabled: false, Reason: ReasonDisabled}
}

func evaluateConditions(group domain.ConditionGroup, attributes map[string]any) bool {
	if group.Empty() {
		return true // brak conditions = zawsze przechodzi
	}

	results := make([]bool, len(group.Conditions))
	for i, c := range group.Conditions {
		results[i] = evaluateCondition(c, attributes)
	}

	if group.Operator == domain.OperatorAND {
		for _, r := range results {
			if !r {
				return false
			}
		}
		return true
	}

	// OR
	for _, r := range results {
		if r {
			return true
		}
	}
	return false
}

func evaluateCondition(c domain.Condition, attributes map[string]any) bool {
	val, ok := attributes[c.Attribute]
	if !ok {
		return false
	}

	switch c.Operator {
	case domain.CompareEq:
		return fmt.Sprintf("%v", val) == fmt.Sprintf("%v", c.Value)
	case domain.CompareNeq:
		return fmt.Sprintf("%v", val) != fmt.Sprintf("%v", c.Value)
	case domain.CompareGt:
		return toFloat(val) > toFloat(c.Value)
	case domain.CompareLt:
		return toFloat(val) < toFloat(c.Value)
	case domain.CompareGte:
		return toFloat(val) >= toFloat(c.Value)
	case domain.CompareLte:
		return toFloat(val) <= toFloat(c.Value)
	case domain.CompareIn:
		return containsValue(c.Value, val)
	case domain.CompareNotIn:
		return !containsValue(c.Value, val)
	case domain.CompareContains:
		return strings.Contains(fmt.Sprintf("%v", val), fmt.Sprintf("%v", c.Value))
	}
	return false
}

func bucket(userID string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(userID))
	return int(h.Sum32() % 100)
}
