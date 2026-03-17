package pingone

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/core"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/utils"
)

func init() {
	// API client dispatch.
	registerResource("pingone_davinci_connector_instance", resourceHandler{
		list: listConnectorInstances,
		get:  getConnectorInstance,
	})

	// Custom transform for connector instance property mapping.
	registerTransform("handleConnectorProperties", handleConnectorProperties)
}

// handleConnectorProperties formats the connector instance Properties map as
// a jsonencode({...}) expression suitable for Terraform HCL output. Each
// property preserves the API's {type, value} structure. Masked secrets
// ("******") are replaced with Terraform variable references.
//
// Returns core.TransformResultWithVariables so the processor can populate
// ResourceData.ExtractedVariables for module variable declarations.
func handleConnectorProperties(value interface{}, apiData interface{}, _ *schema.AttributeDefinition, def *schema.ResourceDefinition) (interface{}, error) {
	props, ok := value.(map[string]interface{})
	if !ok || len(props) == 0 {
		return nil, nil
	}

	resourceName := extractResourceName(apiData)

	// Read variable prefix from definition; fall back to hardcoded value
	// if the definition is nil (e.g., in tests without full setup).
	varPrefix := "davinci_connection_"
	if def != nil && def.Variables.VariablePrefix != "" {
		varPrefix = def.Variables.VariablePrefix
	}

	// Sort keys for deterministic output.
	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf strings.Builder
	buf.WriteString("jsonencode({\n")

	var extractedVars []core.ExtractedVariable

	// Filter to valid property keys before rendering so the
	// trailing-comma logic is based on the actual output count.
	validKeys := make([]string, 0, len(keys))
	for _, key := range keys {
		_, ok := props[key].(map[string]interface{})
		if !ok {
			continue
		}
		validKeys = append(validKeys, key)
	}

	for i, key := range validKeys {
		propMap := props[key].(map[string]interface{})

		normalized, isComplex := normalizeComplexProperty(propMap)
		if isComplex {
			// Complex property: type becomes "json", value is nested jsonencode.
			buf.WriteString(fmt.Sprintf("      \"%s\": {\n", key))
			buf.WriteString("          \"type\": \"json\",\n")

			innerJSON, nestedVars := formatComplexPropertyValue(normalized, varPrefix, resourceName, key, def.Metadata.ResourceType)
			buf.WriteString(fmt.Sprintf("          \"value\": %s\n", innerJSON))
			extractedVars = append(extractedVars, nestedVars...)

			if i < len(validKeys)-1 {
				buf.WriteString("      },\n")
			} else {
				buf.WriteString("      }\n")
			}
		} else {
			// Simple property: existing {type, value} handling.
			buf.WriteString(fmt.Sprintf("      \"%s\": {\n", key))

			// Write type field only when non-empty (matches API omitEmpty).
			if typeVal, ok := propMap["type"]; ok {
				if typeStr, ok := typeVal.(string); ok && strings.TrimSpace(typeStr) != "" {
					buf.WriteString(fmt.Sprintf("          \"type\": \"%s\",\n", typeStr))
				}
			}

			// Format value, replacing masked secrets with variable references.
			rawValue := propMap["value"]
			formatted := formatPropertyValue(rawValue, varPrefix, resourceName, key)
			buf.WriteString(fmt.Sprintf("          \"value\": %s\n", formatted))

			if i < len(validKeys)-1 {
				buf.WriteString("      },\n")
			} else {
				buf.WriteString("      }\n")
			}

			// Extract variable metadata for every property that has
			// a value (including masked secrets).
			if rawValue != nil {
				isMaskedSecret := false
				if strVal, ok := rawValue.(string); ok && strings.TrimSpace(strVal) == "******" {
					isMaskedSecret = true
				}

				varName := connectorVariableName(varPrefix, resourceName, key)

				ev := core.ExtractedVariable{
					AttributePath: "properties." + key,
					VariableName:  varName,
					VariableType:  inferPropertyVariableType(rawValue),
					Description:   key + " property for " + resourceName + " connector",
					Sensitive:     isMaskedSecret,
					IsSecret:      isMaskedSecret,
					ResourceType:  def.Metadata.ResourceType,
					ResourceName:  resourceName,
				}
				// Only include current value for non-secrets.
				if !isMaskedSecret {
					ev.CurrentValue = rawValue
				}
				extractedVars = append(extractedVars, ev)
			}
		}
	}

	buf.WriteString("  })")

	return core.TransformResultWithVariables{
		Value:     core.RawHCLValue(buf.String()),
		Variables: extractedVars,
	}, nil
}

// inferPropertyVariableType maps a Go runtime value to a Terraform variable type.
func inferPropertyVariableType(value interface{}) string {
	switch value.(type) {
	case bool:
		return "string" // Properties are always strings inside jsonencode
	case float64, int:
		return "string"
	default:
		return "string"
	}
}

// formatPropertyValue renders a single property value for use inside
// a jsonencode expression. All non-nil values are replaced with variable refs.
func formatPropertyValue(value interface{}, prefix, resourceName, propertyName string) string {
	if value == nil {
		return "null"
	}
	varName := connectorVariableName(prefix, resourceName, propertyName)
	return fmt.Sprintf("\"${var.%s}\"", varName)
}

// extractResourceName retrieves the Name field from an API data struct via
// reflection and sanitizes it to produce a Terraform resource label.
func extractResourceName(apiData interface{}) string {
	rv := reflect.ValueOf(apiData)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Struct {
		f := rv.FieldByName("Name")
		if f.IsValid() && f.Kind() == reflect.String {
			return utils.SanitizeResourceName(f.String())
		}
	}
	return ""
}

// isScalarValue returns true if the value is a scalar (string, number, bool),
// false for arrays, objects, and other complex types.
func isScalarValue(value interface{}) bool {
	switch value.(type) {
	case string, bool, float64, int:
		return true
	default:
		return false
	}
}

// connectorVariableName builds a Terraform variable name for a connector
// property. Format: {prefix}{cleanedName}_{propertyName}
// The output is sanitized to match schema-driven variable naming:
// non-alphanumeric/underscore characters are replaced with underscores,
// and preserves the casing from the original components.
func connectorVariableName(prefix, resourceName, propertyName string) string {
	cleanName := resourceName
	for _, pfx := range []string{"pingcli__", "davinci__"} {
		cleanName = strings.TrimPrefix(cleanName, pfx)
	}
	return utils.SanitizeVariableName(prefix + cleanName + "_" + propertyName)
}

// isComplexProperty returns true when a property map contains a nested
// "properties" object, indicating a complex type (customAuth, oauth2, openId, saml).
// It handles both top-level properties (Shape 1) and properties nested inside
// "value" (Shape 2).
func isComplexProperty(propMap map[string]interface{}) bool {
	_, ok := normalizeComplexProperty(propMap)
	return ok
}

// normalizeComplexProperty detects and normalizes the two API shapes for complex
// connector properties into a canonical form with "properties" at the top level.
//
// Shape 1 (top-level): {"type": "array", "properties": {...}}
//   - Returned as-is.
//
// Shape 2a (nested map): {"type": "object", "value": {"properties": {...}}}
//   - The inner "properties" is lifted to the top level.
//
// Shape 2b (nested JSON string): {"type": "object", "value": "{\"properties\":{...}}"}
//   - The string is unmarshalled; if valid and contains "properties", it is lifted.
//
// Returns the normalized map and true if the property is complex, or the original
// map and false otherwise.
func normalizeComplexProperty(propMap map[string]interface{}) (map[string]interface{}, bool) {
	// Shape 1: "properties" at top level.
	if _, ok := propMap["properties"]; ok {
		return propMap, true
	}

	rawValue, hasValue := propMap["value"]
	if !hasValue {
		return propMap, false
	}

	// Shape 2a: "value" is a map containing "properties".
	if valueMap, ok := rawValue.(map[string]interface{}); ok {
		if nested, ok := valueMap["properties"]; ok {
			normalized := map[string]interface{}{
				"type":       propMap["type"],
				"properties": nested,
			}
			return normalized, true
		}
	}

	// Shape 2b: "value" is a JSON string containing "properties".
	if strValue, ok := rawValue.(string); ok {
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(strValue), &parsed); err == nil {
			if nested, ok := parsed["properties"]; ok {
				normalized := map[string]interface{}{
					"type":       propMap["type"],
					"properties": nested,
				}
				return normalized, true
			}
		}
	}

	return propMap, false
}

// isNestedSecret detects whether a nested property within a complex property
// is a secret. Uses two detection signals:
//  1. "secure": true flag in the property metadata
//  2. "value": "******" masked value from the API
func isNestedSecret(_ string, propMap map[string]interface{}) bool {
	// Signal 1: explicit "secure" flag from API metadata.
	if secure, ok := propMap["secure"].(bool); ok && secure {
		return true
	}

	// Signal 2: masked value from API response.
	if val, ok := propMap["value"].(string); ok && strings.TrimSpace(val) == "******" {
		return true
	}

	return false
}

// formatComplexPropertyValue processes a complex property's nested properties
// and renders them as a jsonencode({...}) expression. Metadata fields
// (displayName, preferredControlType, info, etc.) are stripped — only "value"
// and "secure" (for secrets) are preserved. Secret values are replaced with
// Terraform variable references.
//
// Returns the formatted jsonencode string and any extracted variables.
func formatComplexPropertyValue(propMap map[string]interface{}, prefix, resourceName, parentKey, resourceType string) (string, []core.ExtractedVariable) {
	nestedProps, ok := propMap["properties"].(map[string]interface{})
	if !ok || len(nestedProps) == 0 {
		// No nested properties — strip metadata and encode the remaining fields.
		cleaned := make(map[string]interface{})
		for k, v := range propMap {
			switch k {
			case "displayName", "preferredControlType", "info":
				continue
			default:
				cleaned[k] = v
			}
		}
		jsonBytes, err := json.Marshal(cleaned)
		if err != nil {
			return "null", nil
		}
		return fmt.Sprintf("jsonencode(%s)", string(jsonBytes)), nil
	}

	// Sort nested property keys for deterministic output.
	nestedKeys := make([]string, 0, len(nestedProps))
	for k := range nestedProps {
		nestedKeys = append(nestedKeys, k)
	}
	sort.Strings(nestedKeys)

	// Build the cleaned structure: only "value" and "secure" per nested property.
	cleanedProps := make(map[string]interface{})
	var extractedVars []core.ExtractedVariable

	for _, nk := range nestedKeys {
		nv, ok := nestedProps[nk].(map[string]interface{})
		if !ok {
			continue
		}

		secret := isNestedSecret(nk, nv)
		rawValue := nv["value"]

		entry := make(map[string]interface{})
		if secret {
			varName := connectorVariableName(prefix, resourceName, parentKey+"_"+nk)
			entry["value"] = fmt.Sprintf("${var.%s}", varName)
			entry["secure"] = true

			extractedVars = append(extractedVars, core.ExtractedVariable{
				AttributePath: "properties." + parentKey + "." + nk,
				VariableName:  varName,
				VariableType:  "string",
				Description:   nk + " property for " + resourceName + " connector (" + parentKey + ")",
				Sensitive:     true,
				IsSecret:      true,
				ResourceType:  resourceType,
				ResourceName:  resourceName,
			})
		} else {
			// Skip properties with no value — omitting them lets Terraform
			// use provider defaults and avoids generating orphan variables.
			if rawValue == nil {
				continue
			}

			varName := connectorVariableName(prefix, resourceName, parentKey+"_"+nk)

			// For scalar values, use variable reference; for arrays/objects, preserve structure
			if isScalarValue(rawValue) {
				entry["value"] = fmt.Sprintf("${var.%s}", varName)
			} else {
				entry["value"] = rawValue
			}
			// Only emit "secure" when the API actually included it.
			// Emitting false when the API omitted it causes a Terraform plan diff.
			if _, hasSecure := nv["secure"]; hasSecure {
				entry["secure"] = false
			}

			extractedVars = append(extractedVars, core.ExtractedVariable{
				AttributePath: "properties." + parentKey + "." + nk,
				VariableName:  varName,
				VariableType:  "string",
				Description:   nk + " property for " + resourceName + " connector (" + parentKey + ")",
				Sensitive:     false,
				IsSecret:      false,
				CurrentValue:  rawValue,
				ResourceType:  resourceType,
				ResourceName:  resourceName,
			})
		}
		cleanedProps[nk] = entry
	}

	// Build nested jsonencode() expression.
	// hclwrite.Format correctly re-indents nested jsonencode() calls when
	// the entire properties attribute is emitted as a single TokenIdent.
	outerMap := map[string]interface{}{
		"properties": cleanedProps,
	}
	jsonBytes, err := json.MarshalIndent(outerMap, "          ", "    ")
	if err != nil {
		return "null", extractedVars
	}

	return fmt.Sprintf("jsonencode(%s)", string(jsonBytes)), extractedVars
}

// listConnectorInstances implements list-then-get: lists all connector instances
// to collect IDs, then calls get for each to retrieve full details including
// properties (which the list endpoint omits).
func listConnectorInstances(ctx context.Context, c *Client, _ string) ([]interface{}, error) {
	resp, _, err := c.apiClient.DaVinciConnectorsApi.GetConnectorInstances(ctx, c.environmentID).Execute()
	if err != nil {
		return nil, fmt.Errorf("list connector instances: %w", err)
	}
	embedded, ok := resp.GetEmbeddedOk()
	if !ok || embedded == nil {
		return []interface{}{}, nil
	}
	instances, ok := embedded.GetConnectorInstancesOk()
	if !ok || instances == nil {
		return []interface{}{}, nil
	}
	result := make([]interface{}, 0, len(instances))
	for _, inst := range instances {
		// Skip User Pool connector (not manageable via Terraform).
		if connector, ok := inst.GetConnectorOk(); ok && connector != nil {
			if strings.EqualFold(connector.GetId(), "skUserPool") {
				continue
			}
		}

		detail, _, err := c.apiClient.DaVinciConnectorsApi.GetConnectorInstanceById(ctx, c.environmentID, inst.GetId()).Execute()
		if err != nil {
			return nil, fmt.Errorf("get connector instance %s: %w", inst.GetId(), err)
		}
		result = append(result, detail)
	}
	return result, nil
}

func getConnectorInstance(ctx context.Context, c *Client, _ string, resourceID string) (interface{}, error) {
	detail, _, err := c.apiClient.DaVinciConnectorsApi.GetConnectorInstanceById(ctx, c.environmentID, resourceID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get connector instance: %w", err)
	}
	return detail, nil
}
