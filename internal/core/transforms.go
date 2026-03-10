// Package core provides the generic resource processing engine.
package core

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
)

// TransformFunc converts a raw API value into its Terraform representation
// using the attribute definition for context.
type TransformFunc func(value interface{}, attr *schema.AttributeDefinition) (interface{}, error)

// StandardTransforms maps transform names declared in YAML definitions to
// their Go implementations. The "custom" transform is intentionally absent —
// it is handled separately via the custom handler registry.
var StandardTransforms = map[string]TransformFunc{
	"passthrough":    transformPassthrough,
	"base64_encode":  transformBase64Encode,
	"base64_decode":  transformBase64Decode,
	"json_encode":    transformJSONEncode,
	"json_decode":    transformJSONDecode,
	"jsonencode_raw": transformJSONEncodeRaw,
	"value_map":      transformValueMap,
	"to_string":      transformToString,
}

// ApplyTransform applies the named transform to value.
// If the transform name is empty or "passthrough", the value is returned as-is.
// Returns an error if the transform name is unknown.
func ApplyTransform(name string, value interface{}, attr *schema.AttributeDefinition) (interface{}, error) {
	if name == "" || name == "passthrough" {
		return value, nil
	}

	// "custom" is handled by the custom handler registry; pass through here.
	if name == "custom" {
		return value, nil
	}

	fn, ok := StandardTransforms[name]
	if !ok {
		return nil, fmt.Errorf("unknown transform %q for attribute %s", name, attr.Name)
	}

	return fn(value, attr)
}

// RegisterTransform adds or replaces a named transform at runtime.
// Useful for tests and platform-specific extensions.
func RegisterTransform(name string, fn TransformFunc) {
	StandardTransforms[name] = fn
}

// --- Standard transform implementations ---

// transformPassthrough returns the value unchanged.
func transformPassthrough(value interface{}, _ *schema.AttributeDefinition) (interface{}, error) {
	return value, nil
}

// transformBase64Encode encodes a string value to base64.
func transformBase64Encode(value interface{}, attr *schema.AttributeDefinition) (interface{}, error) {
	s, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("base64_encode: attribute %s: expected string, got %T", attr.Name, value)
	}
	return base64.StdEncoding.EncodeToString([]byte(s)), nil
}

// transformBase64Decode decodes a base64-encoded string value.
func transformBase64Decode(value interface{}, attr *schema.AttributeDefinition) (interface{}, error) {
	s, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("base64_decode: attribute %s: expected string, got %T", attr.Name, value)
	}
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("base64_decode: attribute %s: %w", attr.Name, err)
	}
	return string(decoded), nil
}

// transformJSONEncode marshals any value to a JSON string.
func transformJSONEncode(value interface{}, attr *schema.AttributeDefinition) (interface{}, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("json_encode: attribute %s: %w", attr.Name, err)
	}
	return string(b), nil
}

// transformJSONDecode parses a JSON string into a Go value.
func transformJSONDecode(value interface{}, attr *schema.AttributeDefinition) (interface{}, error) {
	s, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("json_decode: attribute %s: expected string, got %T", attr.Name, value)
	}
	var result interface{}
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return nil, fmt.Errorf("json_decode: attribute %s: %w", attr.Name, err)
	}
	return result, nil
}

// transformToString converts any value to its string representation.
func transformToString(value interface{}, _ *schema.AttributeDefinition) (interface{}, error) {
	if value == nil {
		return "", nil
	}
	return fmt.Sprintf("%v", value), nil
}

// transformJSONEncodeRaw marshals the value to JSON and wraps it in a
// jsonencode() HCL function call, returning a RawHCLValue so the
// formatter emits the expression without quoting.
// HCL template markers (${ and %{) in string values are escaped to
// prevent Terraform from interpreting them as interpolation.
func transformJSONEncodeRaw(value interface{}, attr *schema.AttributeDefinition) (interface{}, error) {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("jsonencode_raw: attribute %s: %w", attr.Name, err)
	}
	// Escape HCL template markers in JSON string values.
	// jsonencode() arguments are Terraform expressions; "${" and "%{" inside
	// string literals trigger template interpolation, so they must be doubled.
	s := string(b)
	s = strings.ReplaceAll(s, "${", "$${")
	s = strings.ReplaceAll(s, "%{", "%%{")
	return RawHCLValue(fmt.Sprintf("jsonencode(%s)", s)), nil
}

// transformValueMap looks up the string value in the attribute's ValueMap and
// returns the mapped value. If the value is not found, returns the
// ValueMapDefault (if set) or the original value.
func transformValueMap(value interface{}, attr *schema.AttributeDefinition) (interface{}, error) {
	s, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("value_map: attribute %s: expected string, got %T", attr.Name, value)
	}

	if attr.ValueMap != nil {
		if mapped, found := attr.ValueMap[s]; found {
			return mapped, nil
		}
	}

	if attr.ValueMapDefault != "" {
		return attr.ValueMapDefault, nil
	}

	return s, nil
}

// GeneratedVariable describes a Terraform variable created by masked secret
// replacement, to be registered with the variable extraction system.
type GeneratedVariable struct {
	Name      string
	Sensitive bool
}

// BuildMaskedSecretVarName builds a Terraform variable name from the
// configured parts, substituting runtime values for resource_name and
// attribute_key sources.
func BuildMaskedSecretVarName(parts []schema.VariableNamePart, resourceName, attrKey string) string {
	var segments []string
	for _, p := range parts {
		switch p.Source {
		case "resource_name":
			segments = append(segments, resourceName)
		case "attribute_key":
			segments = append(segments, attrKey)
		case "literal":
			segments = append(segments, p.Value)
		}
	}
	return strings.Join(segments, "_")
}

// ReplaceMaskedSecrets walks a string or map value replacing sentinel values
// with Terraform variable references. Returns the modified value and any
// generated variables. For maps, each key is checked individually; for
// strings, the entire value is checked.
func ReplaceMaskedSecrets(value interface{}, config *schema.MaskedSecretConfig, resourceName string) (interface{}, []GeneratedVariable) {
	if config == nil {
		return value, nil
	}

	sentinel := config.Sentinel
	if sentinel == "" {
		return value, nil
	}

	switch v := value.(type) {
	case string:
		if v == sentinel {
			varName := BuildMaskedSecretVarName(config.VariableNameParts, resourceName, "")
			return fmt.Sprintf("${var.%s}", varName), []GeneratedVariable{{Name: varName, Sensitive: true}}
		}
		return v, nil

	case map[string]interface{}:
		var vars []GeneratedVariable
		result := make(map[string]interface{}, len(v))
		for k, val := range v {
			if s, ok := val.(string); ok && s == sentinel {
				varName := BuildMaskedSecretVarName(config.VariableNameParts, resourceName, k)
				result[k] = fmt.Sprintf("${var.%s}", varName)
				vars = append(vars, GeneratedVariable{Name: varName, Sensitive: true})
			} else {
				result[k] = val
			}
		}
		return result, vars

	default:
		return value, nil
	}
}
