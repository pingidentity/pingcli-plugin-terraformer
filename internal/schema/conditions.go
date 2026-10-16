package schema

import "fmt"

// EvaluateConditions reports whether all conditions hold against the given
// processed-attribute map. Shared by conditional_defaults evaluation
// (processor) and conditional variable rule flags (extractor) so both use the
// same condition semantics.
func EvaluateConditions(attrs map[string]interface{}, conditions []DefaultCondition) bool {
	for _, cond := range conditions {
		if cond.AttributeEmpty != "" {
			val, exists := attrs[cond.AttributeEmpty]
			if exists && val != nil {
				if s, ok := val.(string); ok && s == "" {
					// empty string counts as empty
				} else {
					return false
				}
			}
			// nil or not-present counts as empty — condition passes
		}

		if cond.AttributeEquals != nil {
			val, exists := attrs[cond.AttributeEquals.Name]
			if !exists {
				return false
			}
			if !conditionValuesEqual(val, cond.AttributeEquals.Value) {
				return false
			}
		}
	}
	return true
}

// conditionValuesEqual compares two values for equality, handling type
// coercion between bool and interface{} from YAML unmarshaling.
func conditionValuesEqual(a, b interface{}) bool {
	if a == b {
		return true
	}
	// YAML unmarshals booleans as bool, but processor may store them differently.
	if ab, ok := a.(bool); ok {
		if bb, ok := b.(bool); ok {
			return ab == bb
		}
	}
	// Compare as strings for mixed-type edge cases.
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}
