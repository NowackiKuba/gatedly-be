package evaluation

import (
	"fmt"
	"strings"
)

// toFloat converts any numeric-ish value to float64 for comparison.
// JSON numbers unmarshal as float64, but we also handle int/int64 just in case.
func toFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case int32:
		return float64(n)
	}
	return 0
}

// getValueByPath returns the value at a dotted path in a nested map, e.g. "subscription.tier"
// in {"subscription": {"tier": "premium"}} returns "premium", true.
// If the path has no dots, it is a single top-level key.
func getValueByPath(m map[string]any, path string) (any, bool) {
	if m == nil || path == "" {
		return nil, false
	}
	parts := strings.Split(path, ".")
	var current any = m
	for _, key := range parts {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, false
		}
		nested, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = nested[key]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

// containsValue checks whether needle is present in haystack.
// haystack is expected to be []any (how JSON arrays unmarshal).
func containsValue(haystack, needle any) bool {
	list, ok := haystack.([]any)
	if !ok {
		return false
	}
	needleStr := fmt.Sprintf("%v", needle)
	for _, item := range list {
		if fmt.Sprintf("%v", item) == needleStr {
			return true
		}
	}
	return false
}
