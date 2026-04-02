// Package tfjson generates Terraform JSON configuration from processed resource data.
package tfjson

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/core"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/utils"
)

// FormatOptions controls JSON rendering behavior.
type FormatOptions struct {
	// SkipDependencies outputs raw UUIDs instead of Terraform references.
	SkipDependencies bool
	// EnvironmentID is the raw environment UUID used when SkipDependencies is true.
	EnvironmentID string
}

// Formatter generates Terraform JSON (.tf.json) output from processed resource data.
type Formatter struct{}

// NewFormatter creates a new Terraform JSON formatter.
func NewFormatter() *Formatter {
	return &Formatter{}
}

// FileExtension returns the file extension for Terraform JSON files.
func (f *Formatter) FileExtension() string {
	return ".tf.json"
}

// Format generates a single Terraform JSON resource block from resource data.
func (f *Formatter) Format(data *core.ResourceData, def *schema.ResourceDefinition, opts FormatOptions) (string, error) {
	if data == nil {
		return "", fmt.Errorf("resource data is nil")
	}
	if def == nil {
		return "", fmt.Errorf("resource definition is nil")
	}

	label := resourceLabel(data, def)
	if label == "" {
		return "", fmt.Errorf("resource has neither Name nor ID set")
	}

	attrs := buildAttributes(data, def, opts)

	doc := map[string]interface{}{
		"resource": map[string]interface{}{
			def.Metadata.ResourceType: map[string]interface{}{
				label: attrs,
			},
		},
	}

	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("json marshal: %w", err)
	}
	return string(b) + "\n", nil
}

// FormatList generates a single Terraform JSON document containing all resources
// of the same type, sorted deterministically by resource name. The output
// conforms to the .tf.json spec: one root object per file.
func (f *Formatter) FormatList(dataList []*core.ResourceData, def *schema.ResourceDefinition, opts FormatOptions) (string, error) {
	if def == nil {
		return "", fmt.Errorf("resource definition is nil")
	}

	sorted := make([]*core.ResourceData, len(dataList))
	copy(sorted, dataList)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})

	labels := make(map[string]interface{})
	for _, data := range sorted {
		if data == nil {
			continue
		}
		label := resourceLabel(data, def)
		if label == "" {
			return "", fmt.Errorf("resource has neither Name nor ID set")
		}
		labels[label] = buildAttributes(data, def, opts)
	}

	if len(labels) == 0 {
		return "", nil
	}

	doc := map[string]interface{}{
		"resource": map[string]interface{}{
			def.Metadata.ResourceType: labels,
		},
	}

	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("json marshal: %w", err)
	}
	return string(b) + "\n", nil
}

// FormatImportBlock generates a Terraform JSON import block for a resource.
func (f *Formatter) FormatImportBlock(data *core.ResourceData, def *schema.ResourceDefinition, environmentID string) (string, error) {
	if data == nil {
		return "", fmt.Errorf("resource data is nil")
	}
	if def == nil {
		return "", fmt.Errorf("resource definition is nil")
	}

	label := resourceLabel(data, def)

	importID := def.Dependencies.ImportIDFormat
	importID = strings.ReplaceAll(importID, "{env_id}", environmentID)
	importID = strings.ReplaceAll(importID, "{resource_id}", data.ID)
	for k, v := range data.Attributes {
		if s, ok := v.(string); ok {
			importID = strings.ReplaceAll(importID, "{"+k+"}", s)
		} else if ref, ok := v.(core.ResolvedReference); ok && ref.OriginalValue != "" {
			importID = strings.ReplaceAll(importID, "{"+k+"}", ref.OriginalValue)
		}
	}

	doc := map[string]interface{}{
		"import": []interface{}{
			map[string]interface{}{
				"to": def.Metadata.ResourceType + "." + label,
				"id": importID,
			},
		},
	}

	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("json marshal: %w", err)
	}
	return string(b) + "\n", nil
}

// buildAttributes constructs the attribute map for a resource.
func buildAttributes(data *core.ResourceData, def *schema.ResourceDefinition, opts FormatOptions) map[string]interface{} {
	attrs := make(map[string]interface{})

	for _, attrDef := range def.Attributes {
		if attrDef.Computed && !attrDef.Required && attrDef.ReferencesType == "" {
			continue
		}

		tName := terraformName(attrDef)

		// Reference attributes.
		if attrDef.ReferencesType != "" {
			val, valOK := data.Attributes[tName]
			if !valOK || val == nil {
				continue
			}
			if ref, ok := val.(core.ResolvedReference); ok {
				attrs[tName] = "${" + ref.Expression() + "}"
				continue
			}
			if raw, ok := val.(string); ok && raw != "" {
				attrs[tName] = raw
			}
			continue
		}

		// Nested object.
		if attrDef.Type == "object" && len(attrDef.NestedAttributes) > 0 {
			val, ok := data.Attributes[tName]
			if !ok || val == nil {
				continue
			}
			valMap, ok := val.(map[string]interface{})
			if !ok {
				continue
			}
			result := renderNestedObject(attrDef.NestedAttributes, valMap, opts)
			if len(result) > 0 {
				attrs[tName] = result
			}
			continue
		}

		// Map with nested schema.
		if attrDef.Type == "map" && len(attrDef.NestedAttributes) > 0 {
			val, ok := data.Attributes[tName]
			if !ok || val == nil {
				continue
			}
			outerMap, ok := val.(map[string]interface{})
			if !ok || len(outerMap) == 0 {
				continue
			}
			result := make(map[string]interface{})
			mapKeys := sortedKeys(outerMap)
			for _, k := range mapKeys {
				mv, ok := outerMap[k].(map[string]interface{})
				if !ok {
					continue
				}
				sub := renderNestedObject(attrDef.NestedAttributes, mv, opts)
				if len(sub) > 0 {
					result[k] = sub
				}
			}
			if len(result) > 0 {
				attrs[tName] = result
			}
			continue
		}

		// List of objects with nested schema.
		if (attrDef.Type == "list" || attrDef.Type == "set") && len(attrDef.NestedAttributes) > 0 {
			val, ok := data.Attributes[tName]
			if !ok || val == nil {
				continue
			}
			slice, ok := val.([]interface{})
			if !ok {
				continue
			}
			// Empty slice: emit empty array (e.g., nil_value: keep_empty).
			if len(slice) == 0 {
				attrs[tName] = []interface{}{}
				continue
			}
			var result []interface{}
			for _, item := range slice {
				itemMap, ok := item.(map[string]interface{})
				if !ok {
					continue
				}
				sub := renderNestedObject(attrDef.NestedAttributes, itemMap, opts)
				if len(sub) > 0 {
					result = append(result, sub)
				}
			}
			if len(result) > 0 {
				attrs[tName] = result
			}
			continue
		}

		// Dynamic object (object without nested_attributes or type_discriminated_block).
		if (attrDef.Type == "object" && len(attrDef.NestedAttributes) == 0) || attrDef.Type == "type_discriminated_block" {
			val, ok := data.Attributes[tName]
			if !ok || val == nil {
				continue
			}
			if rv, ok := val.(core.RawHCLValue); ok {
				attrs[tName] = "${" + string(rv) + "}"
				continue
			}
			if valMap, ok := val.(map[string]interface{}); ok {
				result := renderDynamicObject(valMap)
				if len(result) > 0 {
					attrs[tName] = result
				}
			}
			continue
		}

		// Scalar attribute.
		val, ok := data.Attributes[tName]
		if !ok || val == nil {
			continue
		}
		attrs[tName] = renderScalar(val)
	}

	// Add depends_on array if runtime dependencies have resolved labels.
	if deps := resolvedDependsOn(data.DependsOnResources); len(deps) > 0 {
		var arr []interface{}
		for _, d := range deps {
			arr = append(arr, d.ResourceType+"."+d.Label)
		}
		attrs["depends_on"] = arr
	}

	return attrs
}

// resolvedDependsOn filters RuntimeDependsOn entries to those with non-empty labels.
func resolvedDependsOn(deps []core.RuntimeDependsOn) []core.RuntimeDependsOn {
	if len(deps) == 0 {
		return nil
	}
	var out []core.RuntimeDependsOn
	for _, d := range deps {
		if d.Label != "" {
			out = append(out, d)
		}
	}
	return out
}

// renderNestedObject recursively builds a JSON-compatible map from nested attributes.
func renderNestedObject(nested []schema.AttributeDefinition, valMap map[string]interface{}, opts FormatOptions) map[string]interface{} {
	result := make(map[string]interface{})

	for _, attr := range nested {
		nName := terraformName(attr)
		nVal, nOk := valMap[nName]
		if !nOk || nVal == nil {
			continue
		}

		// ResolvedReference in nested context.
		if ref, ok := nVal.(core.ResolvedReference); ok {
			result[nName] = "${" + ref.Expression() + "}"
			continue
		}

		// Nested object: recurse.
		if attr.Type == "object" && len(attr.NestedAttributes) > 0 {
			subMap, ok := nVal.(map[string]interface{})
			if !ok || len(subMap) == 0 {
				continue
			}
			sub := renderNestedObject(attr.NestedAttributes, subMap, opts)
			if len(sub) > 0 {
				result[nName] = sub
			}
			continue
		}

		// Nested map with schema.
		if attr.Type == "map" && len(attr.NestedAttributes) > 0 {
			outerMap, ok := nVal.(map[string]interface{})
			if !ok || len(outerMap) == 0 {
				continue
			}
			mapResult := make(map[string]interface{})
			mapKeys := sortedKeys(outerMap)
			for _, mk := range mapKeys {
				mv, ok := outerMap[mk].(map[string]interface{})
				if !ok {
					continue
				}
				sub := renderNestedObject(attr.NestedAttributes, mv, opts)
				if len(sub) > 0 {
					mapResult[mk] = sub
				}
			}
			if len(mapResult) > 0 {
				result[nName] = mapResult
			}
			continue
		}

		// Nested list/set of objects.
		if (attr.Type == "list" || attr.Type == "set") && len(attr.NestedAttributes) > 0 {
			slice, ok := nVal.([]interface{})
			if !ok {
				continue
			}
			// Empty slice: emit empty array (e.g., nil_value: keep_empty).
			if len(slice) == 0 {
				result[nName] = []interface{}{}
				continue
			}
			var listResult []interface{}
			for _, item := range slice {
				itemMap, ok := item.(map[string]interface{})
				if !ok {
					continue
				}
				sub := renderNestedObject(attr.NestedAttributes, itemMap, opts)
				if len(sub) > 0 {
					listResult = append(listResult, sub)
				}
			}
			if len(listResult) > 0 {
				result[nName] = listResult
			}
			continue
		}

		// Scalar nested value.
		result[nName] = renderScalar(nVal)
	}

	return result
}

// renderDynamicObject renders a map without schema (dynamic object / type_discriminated_block).
func renderDynamicObject(m map[string]interface{}) map[string]interface{} {
	if len(m) == 0 {
		return nil
	}
	result := make(map[string]interface{})
	keys := sortedKeys(m)
	for _, k := range keys {
		v := m[k]
		switch val := v.(type) {
		case core.RawHCLValue:
			result[k] = "${" + string(val) + "}"
		default:
			result[k] = renderScalar(v)
		}
	}
	return result
}

// renderScalar converts a value to its JSON-compatible representation.
func renderScalar(val interface{}) interface{} {
	switch v := val.(type) {
	case core.RawHCLValue:
		return "${" + string(v) + "}"
	case core.ResolvedReference:
		return "${" + v.Expression() + "}"
	case string:
		return v
	case bool:
		return v
	case int64:
		return v
	case float64:
		if v == float64(int64(v)) {
			return int64(v)
		}
		return v
	case []interface{}:
		result := make([]interface{}, 0, len(v))
		for _, item := range v {
			result = append(result, renderScalar(item))
		}
		return result
	case map[string]interface{}:
		result := make(map[string]interface{})
		for mk, mv := range v {
			result[mk] = renderScalar(mv)
		}
		return result
	default:
		return fmt.Sprintf("%v", val)
	}
}

// sortedKeys returns the keys of a map sorted alphabetically.
func sortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// terraformName returns the terraform_name for an attribute, falling back to
// lowercased struct name.
func terraformName(attrDef schema.AttributeDefinition) string {
	if attrDef.TerraformName != "" {
		return attrDef.TerraformName
	}
	return strings.ToLower(attrDef.Name)
}

// resourceLabel derives the sanitized resource label from resource data.
func resourceLabel(data *core.ResourceData, def *schema.ResourceDefinition) string {
	if data.Label != "" {
		return data.Label
	}
	if len(def.API.LabelFields) > 0 {
		keys := make([]string, 0, len(def.API.LabelFields))
		for _, field := range def.API.LabelFields {
			if v, ok := data.Attributes[field]; ok && v != nil {
				if s, ok := v.(string); ok && s != "" {
					keys = append(keys, s)
				}
			}
		}
		if len(keys) > 0 {
			return utils.SanitizeMultiKeyResourceName(keys...)
		}
	}
	if data.Name != "" {
		return utils.SanitizeResourceName(data.Name)
	}
	return data.ID
}
