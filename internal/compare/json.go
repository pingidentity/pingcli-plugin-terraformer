package compare

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// CompareJSON parses two Terraform JSON strings and returns content differences.
func CompareJSON(expected, actual string) (*Result, error) {
	var expData, actData map[string]interface{}
	if err := json.Unmarshal([]byte(expected), &expData); err != nil {
		return nil, fmt.Errorf("parse expected JSON: %w", err)
	}
	if err := json.Unmarshal([]byte(actual), &actData); err != nil {
		return nil, fmt.Errorf("parse actual JSON: %w", err)
	}

	result := &Result{}

	expResources := extractJSONResources(expData)
	actResources := extractJSONResources(actData)

	// Check expected resources.
	for _, key := range sortedMapKeys(expResources) {
		expRes := expResources[key]
		actRes, exists := actResources[key]
		if !exists {
			result.Diffs = append(result.Diffs, Diff{
				Resource: key,
				Kind:     DiffMissingResource,
			})
			continue
		}
		compareJSONResource(result, key, expRes, actRes)
	}

	// Check for extra resources.
	for _, key := range sortedMapKeys(actResources) {
		if _, exists := expResources[key]; !exists {
			result.Diffs = append(result.Diffs, Diff{
				Resource: key,
				Kind:     DiffExtraResource,
			})
		}
	}

	return result, nil
}

// extractJSONResources parses the "resource" key from Terraform JSON into a
// flat map keyed by "type_name.label".
func extractJSONResources(data map[string]interface{}) map[string]map[string]interface{} {
	resources := make(map[string]map[string]interface{})
	resSection, ok := data["resource"]
	if !ok {
		return resources
	}
	resMap, ok := resSection.(map[string]interface{})
	if !ok {
		return resources
	}
	for typeName, typeVal := range resMap {
		typeMap, ok := typeVal.(map[string]interface{})
		if !ok {
			continue
		}
		for label, labelVal := range typeMap {
			key := typeName + "." + label
			attrs, ok := labelVal.(map[string]interface{})
			if !ok {
				continue
			}
			resources[key] = attrs
		}
	}
	return resources
}

func compareJSONResource(result *Result, key string, exp, act map[string]interface{}) {
	expKeys := sortedJSONKeys(exp)
	for _, name := range expKeys {
		expVal := exp[name]
		actVal, exists := act[name]
		if !exists {
			result.Diffs = append(result.Diffs, Diff{
				Resource:  key,
				Kind:      DiffMissingAttribute,
				Attribute: name,
				Expected:  jsonValueString(expVal),
			})
			continue
		}
		expStr := deinterpolateJSONValue(jsonValueString(expVal))
		actStr := deinterpolateJSONValue(jsonValueString(actVal))
		if expStr != actStr {
			// Check if both are nested objects for block mismatch.
			_, expIsObj := expVal.(map[string]interface{})
			_, actIsObj := actVal.(map[string]interface{})
			if expIsObj && actIsObj {
				result.Diffs = append(result.Diffs, Diff{
					Resource:  key,
					Kind:      DiffBlockMismatch,
					Attribute: name,
					Expected:  expStr,
					Actual:    actStr,
				})
			} else {
				result.Diffs = append(result.Diffs, Diff{
					Resource:  key,
					Kind:      DiffValueMismatch,
					Attribute: name,
					Expected:  expStr,
					Actual:    actStr,
				})
			}
		}
	}

	// Extra attributes.
	for _, name := range sortedJSONKeys(act) {
		if _, exists := exp[name]; !exists {
			result.Diffs = append(result.Diffs, Diff{
				Resource:  key,
				Kind:      DiffExtraAttribute,
				Attribute: name,
				Actual:    jsonValueString(act[name]),
			})
		}
	}
}

// deinterpolateJSONValue normalizes a .tf.json compare value for the
// de-interpolation equivalence (issue #158): a RawHCL blob that older output
// rendered as a quoted interpolation-only ref and newer output renders as a
// bare traversal compare equal. The shapes reaching this function after
// jsonValueString:
//   - "${var.x}" / "${type.label.field}" (new tfjson wrap of a bare blob, and
//     plain template attributes)
//   - "${"${var.x}"}" (old tfjson wrap: the blob itself was the quoted
//     interpolation-only string, wrapped again by the formatter)
//   - "${jsonencode({... "value" : "${var.x}" ...})}" (jsonencode blob; old
//     side carries the quoted ref inside, new side the bare token)
//
// As in the HCL path, partial interpolations and non-traversal expressions are
// left untouched so genuine value changes still compare as mismatches.
func deinterpolateJSONValue(v string) string {
	// Old output wrapped a quoted interpolation-only blob in the template
	// wrapper: "${"${var.x}"}". Strip the wrapper and the blob's quotes, then
	// de-interpolate what is inside.
	if strings.HasPrefix(v, "${\"") && strings.HasSuffix(v, "\"}") {
		quoted := v[2 : len(v)-1]
		if len(quoted) >= 2 && quoted[0] == '"' && quoted[len(quoted)-1] == '"' {
			if inner, ok := interpolationOnlyBody(quoted[1 : len(quoted)-1]); ok {
				return inner
			}
		}
	}
	if strings.Contains(v, "jsonencode(") {
		return deinterpolateBlobText(v)
	}
	return deinterpolateValue(v)
}

func sortedJSONKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func jsonValueString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return fmt.Sprintf("%g", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case nil:
		return "null"
	default:
		b, _ := json.Marshal(val)
		return string(b)
	}
}
