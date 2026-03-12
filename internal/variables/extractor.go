package variables

import (
	"fmt"
	"strings"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/core"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/utils"
)

// VariableExtractor evaluates schema-driven rules to determine which resource
// attributes should become module variables.
type VariableExtractor struct {
	registry *schema.Registry
}

// NewVariableExtractor creates a VariableExtractor backed by the given schema registry.
func NewVariableExtractor(registry *schema.Registry) *VariableExtractor {
	return &VariableExtractor{registry: registry}
}

// Extract examines a resource against its schema definition and returns
// variables for every eligible attribute. It evaluates both the
// attribute-level variable_eligible flag and the variables.eligible_attributes rules.
//
// data is the processed attribute map (attribute name -> value).
// resourceName is the sanitized Terraform resource label.
func (e *VariableExtractor) Extract(
	resourceType string,
	data map[string]interface{},
	resourceName string,
) ([]*core.ExtractedVariable, error) {
	def, err := e.registry.Get(resourceType)
	if err != nil {
		return nil, fmt.Errorf("extract variables: %w", err)
	}

	// Use the full resource_type for routing and grouping.
	resourceTypeKey := def.Metadata.ResourceType
	if resourceTypeKey == "" {
		resourceTypeKey = resourceType
	}

	var results []*core.ExtractedVariable

	// Build a lookup of variable rules by attribute_path.
	ruleMap := make(map[string]schema.VariableRule)
	for _, rule := range def.Variables.EligibleAttributes {
		ruleMap[rule.AttributePath] = rule
	}

	// Walk attributes and apply rules.
	for _, attr := range def.Attributes {
		if !attr.VariableEligible {
			continue
		}

		// Use CanonicalAttributeKey for consistent key derivation with the processor.
		tfName := schema.CanonicalAttributeKey(attr)

		value, exists := data[tfName]
		if !exists || value == nil {
			continue
		}

		// Look up any declared extraction rule (keyed by attribute_path which
		// uses terraform_name).
		rule, hasRule := ruleMap[tfName]

		// For type_discriminated_block values, the processor stores a
		// single-key map like {"string": "Acme Corp"}. Unwrap to get the
		// inner value for tfvars output.
		currentValue := unwrapTypeDiscriminatedBlock(value)

		ev := &core.ExtractedVariable{
			ResourceType:  resourceTypeKey,
			ResourceName:  resourceName,
			AttributePath: tfName,
			CurrentValue:  currentValue,
			Sensitive:     attr.Sensitive,
			VariableType:  inferTerraformType(attr.Type),
		}

		// Derive variable name.
		if hasRule && rule.VariablePrefix != "" {
			ev.VariableName = utils.SanitizeVariableName(rule.VariablePrefix + resourceName + "_" + tfName)
		} else {
			ev.VariableName = utils.SanitizeVariableName(resourceType + "_" + resourceName + "_" + tfName)
		}

		// Apply rule overrides.
		if hasRule {
			ev.IsSecret = rule.IsSecret
			if rule.Description != "" {
				ev.Description = expandDescription(rule.Description, resourceName)
			}
		}

		results = append(results, ev)
	}

	return results, nil
}

// --- Helpers ---

// inferTerraformType maps YAML schema types to Terraform variable types.
func inferTerraformType(schemaType string) string {
	switch schemaType {
	case "string":
		return "string"
	case "number":
		return "number"
	case "bool":
		return "bool"
	case "object", "map":
		return "any"
	case "list", "set":
		return "list(any)"
	default:
		return "string"
	}
}

// unwrapTypeDiscriminatedBlock extracts the inner value from a
// type_discriminated_block map. The processor represents these as
// map[string]interface{} with a single key (e.g., {"string": "Acme"}).
// Returns the inner value for simple scalar blocks, or the original
// value if it is not a single-key map.
func unwrapTypeDiscriminatedBlock(value interface{}) interface{} {
	m, ok := value.(map[string]interface{})
	if !ok || len(m) != 1 {
		return value
	}
	for _, v := range m {
		return v
	}
	return value
}

// expandDescription replaces {name} placeholders in rule descriptions.
func expandDescription(template, resourceName string) string {
	return strings.ReplaceAll(template, "{name}", resourceName)
}
