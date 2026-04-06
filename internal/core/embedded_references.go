package core

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/graph"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/utils"
)

// EmbeddedReferenceRule describes a UUID reference embedded inside a RawHCLValue
type EmbeddedReferenceRule struct {
	ResourceType       string // owning resource type (e.g., "pingone_davinci_flow")
	AttributePath      string // dot-path with * wildcard (e.g., "graph_data.elements.nodes.*.data.properties")
	TargetResourceType string // what the UUID references (e.g., "pingone_davinci_flow")
	JSONKeyPath        string // path inside JSON blob (e.g., "subFlowId.value.value")
	ReferenceField     string // TF attribute (e.g., "id")

	// Strategy controls fallback behaviour when the UUID is not found in the graph.
	//   "reference"                (default / zero value) — resolve or skip
	//   "reference_with_fallback"  — resolve if possible, emit variable if not
	//   "variable"                 — always emit a variable, skip graph lookup
	Strategy string

	// VariablePrefix is combined with a name derived from VariableNamingPath to
	// produce the Terraform variable name (e.g., "davinci_form").
	VariablePrefix string

	// VariableNamingPath is a JSON key path inside the same blob used to derive
	// a human-readable variable suffix (e.g., "nodeTitle.value"). When the key
	// is absent the first 8 characters of the UUID are used instead.
	VariableNamingPath string
}

// EmbeddedReferenceRegistry collects rules
type EmbeddedReferenceRegistry struct {
	rules []EmbeddedReferenceRule
}

// NewEmbeddedReferenceRegistry creates a new registry
func NewEmbeddedReferenceRegistry() *EmbeddedReferenceRegistry {
	return &EmbeddedReferenceRegistry{
		rules: make([]EmbeddedReferenceRule, 0),
	}
}

// Register adds a rule to the registry
func (r *EmbeddedReferenceRegistry) Register(rule EmbeddedReferenceRule) {
	r.rules = append(r.rules, rule)
}

// Rules returns all registered rules
func (r *EmbeddedReferenceRegistry) Rules() []EmbeddedReferenceRule {
	return r.rules
}

// ResolveEmbeddedReferences walks exported resource data applying rules to replace
// embedded UUID strings with Terraform references and add corresponding
// dependency edges to the graph.
// It returns deduplicated FallbackVariable entries for any variable fallbacks
// produced by "reference_with_fallback" or "variable" strategy rules.
func ResolveEmbeddedReferences(
	results []*ExportedResourceData,
	g *graph.DependencyGraph,
	rules []EmbeddedReferenceRule,
) []FallbackVariable {
	if len(rules) == 0 || g == nil {
		return nil
	}

	varSeen := make(map[string]bool)
	var fallbackVars []FallbackVariable

	for _, exportedData := range results {
		for _, rule := range rules {
			// Skip if resource type doesn't match
			if rule.ResourceType != exportedData.ResourceType {
				continue
			}

			// Process each resource of this type
			for _, resource := range exportedData.Resources {
				processResourceWithRule(resource, rule, g, varSeen, &fallbackVars)
			}
		}
	}

	return fallbackVars
}

// processResourceWithRule applies a single rule to a resource
func processResourceWithRule(resource *ResourceData, rule EmbeddedReferenceRule, g *graph.DependencyGraph, varSeen map[string]bool, fallbackVars *[]FallbackVariable) {
	// Parse the attribute path into segments
	pathSegments := strings.Split(rule.AttributePath, ".")
	
	// Walk the path and process all matching RawHCLValues
	walkAndProcessPath(resource.Attributes, pathSegments, 0, rule, resource, g, varSeen, fallbackVars)
}

// walkAndProcessPath recursively walks the attribute path and processes matching values
func walkAndProcessPath(
	current interface{},
	pathSegments []string,
	segmentIndex int,
	rule EmbeddedReferenceRule,
	resource *ResourceData,
	g *graph.DependencyGraph,
	varSeen map[string]bool,
	fallbackVars *[]FallbackVariable,
) {
	if segmentIndex >= len(pathSegments) {
		return
	}

	segment := pathSegments[segmentIndex]

	// Check if this is the last segment - if so, we should have a RawHCLValue
	isLastSegment := segmentIndex == len(pathSegments)-1

	switch typedCurrent := current.(type) {
	case map[string]interface{}:
		if segment == "*" {
			// Wildcard: process all keys in this map
			for key := range typedCurrent {
				nextValue := typedCurrent[key]
				if isLastSegment {
					// This key should be a RawHCLValue - try to process it
					if rawValue, ok := nextValue.(RawHCLValue); ok {
						processedValue := processRawHCLValue(rawValue, rule, resource, g, varSeen, fallbackVars)
						typedCurrent[key] = processedValue
					}
				} else {
					// Continue walking deeper
					walkAndProcessPath(nextValue, pathSegments, segmentIndex+1, rule, resource, g, varSeen, fallbackVars)
				}
			}
		} else {
			// Regular key lookup
			nextValue, exists := typedCurrent[segment]
			if !exists {
				return
			}

			if isLastSegment {
				// This is a RawHCLValue - process it
				if rawValue, ok := nextValue.(RawHCLValue); ok {
					processedValue := processRawHCLValue(rawValue, rule, resource, g, varSeen, fallbackVars)
					typedCurrent[segment] = processedValue
				}
			} else {
				// Continue walking deeper
				walkAndProcessPath(nextValue, pathSegments, segmentIndex+1, rule, resource, g, varSeen, fallbackVars)
			}
		}
	}
}

// processRawHCLValue extracts JSON from the RawHCLValue, finds the UUID at JSONKeyPath,
// and replaces it with a Terraform reference or variable depending on the rule's Strategy.
func processRawHCLValue(
	value RawHCLValue,
	rule EmbeddedReferenceRule,
	resource *ResourceData,
	g *graph.DependencyGraph,
	varSeen map[string]bool,
	fallbackVars *[]FallbackVariable,
) RawHCLValue {
	// Extract JSON from jsonencode(...)
	jsonStr := extractJSONFromRawHCL(value)
	if jsonStr == "" {
		return value
	}

	// Parse as JSON
	var jsonData interface{}
	err := json.Unmarshal([]byte(jsonStr), &jsonData)
	if err != nil {
		return value
	}

	// Walk the JSON path to find the UUID
	uuid := walkJSONPath(jsonData, rule.JSONKeyPath)
	if uuid == "" {
		return value
	}

	// Strategy: "variable" — always emit a variable, skip graph lookup
	if rule.Strategy == "variable" {
		varName := deriveVariableName(rule, jsonData, uuid)
		tfRef := fmt.Sprintf("${var.%s}", varName)
		newValue := replaceUUIDInRawHCL(value, uuid, tfRef)
		addEmbeddedFallbackVariable(varName, rule, uuid, varSeen, fallbackVars)
		return RawHCLValue(newValue)
	}

	// Strategy: "reference" (default) or "reference_with_fallback" — try graph lookup
	refName, err := g.GetReferenceName(rule.TargetResourceType, uuid)
	if err != nil {
		// UUID not found in graph
		if rule.Strategy == "reference_with_fallback" {
			varName := deriveVariableName(rule, jsonData, uuid)
			tfRef := fmt.Sprintf("${var.%s}", varName)
			newValue := replaceUUIDInRawHCL(value, uuid, tfRef)
			addEmbeddedFallbackVariable(varName, rule, uuid, varSeen, fallbackVars)
			return RawHCLValue(newValue)
		}
		// Default strategy: leave unchanged
		return value
	}

	// Build the terraform reference string
	tfRef := fmt.Sprintf("${%s.%s.%s}", rule.TargetResourceType, refName, rule.ReferenceField)

	// Replace the UUID string in the RawHCLValue with the terraform reference
	newValue := replaceUUIDInRawHCL(value, uuid, tfRef)

	// Add graph edge
	_ = g.AddEdge(resource.ResourceType, resource.ID, rule.TargetResourceType, uuid, "properties."+rule.JSONKeyPath, "")

	return RawHCLValue(newValue)
}

// deriveVariableName builds a Terraform variable name from the rule's VariablePrefix
// and a human-readable suffix derived from VariableNamingPath inside the JSON blob.
// Falls back to the first 8 characters of the UUID when the naming key is absent.
func deriveVariableName(rule EmbeddedReferenceRule, jsonData interface{}, uuid string) string {
	var suffix string
	if rule.VariableNamingPath != "" {
		suffix = walkJSONPath(jsonData, rule.VariableNamingPath)
	}

	if suffix == "" {
		// Fall back to first 8 chars of UUID
		if len(uuid) > 8 {
			suffix = uuid[:8]
		} else {
			suffix = uuid
		}
	}

	suffix = strings.ToLower(suffix)
	suffix = utils.SanitizeVariableName(suffix)

	if rule.VariablePrefix != "" {
		return rule.VariablePrefix + "_" + suffix
	}
	return suffix
}

// addEmbeddedFallbackVariable adds a FallbackVariable entry if not already seen.
func addEmbeddedFallbackVariable(varName string, rule EmbeddedReferenceRule, uuid string, seen map[string]bool, out *[]FallbackVariable) {
	if seen[varName] {
		return
	}
	seen[varName] = true

	resType := rule.TargetResourceType
	if resType == "" {
		resType = rule.ResourceType
	}

	*out = append(*out, FallbackVariable{
		Name:         varName,
		Type:         "string",
		Description:  fmt.Sprintf("ID for %s resource (not yet exported)", resType),
		ResourceType: resType,
		Default:      uuid,
	})
}

// extractJSONFromRawHCL extracts the JSON content from jsonencode(...) format
func extractJSONFromRawHCL(value RawHCLValue) string {
	str := string(value)
	
	// Find jsonencode(
	prefix := "jsonencode("
	if !strings.HasPrefix(str, prefix) {
		return ""
	}

	// Remove prefix and trailing )
	jsonStart := len(prefix)
	jsonEnd := len(str) - 1 // Remove the closing )
	
	if jsonEnd <= jsonStart {
		return ""
	}

	return str[jsonStart:jsonEnd]
}

// walkJSONPath walks through a JSON object following dot-separated path to find a string value
func walkJSONPath(jsonData interface{}, path string) string {
	if path == "" {
		return ""
	}

	pathSegments := strings.Split(path, ".")
	current := jsonData

	for _, segment := range pathSegments {
		switch typedCurrent := current.(type) {
		case map[string]interface{}:
			next, exists := typedCurrent[segment]
			if !exists {
				return ""
			}
			current = next
		default:
			return ""
		}
	}

	// The final value should be a string
	if str, ok := current.(string); ok {
		return str
	}

	return ""
}

// replaceUUIDInRawHCL replaces the first occurrence of the quoted UUID
// string with a quoted Terraform reference. Only the first match is
// replaced to avoid over-replacement when the same UUID appears in
// other JSON fields (labels, metadata, etc.).
func replaceUUIDInRawHCL(value RawHCLValue, uuid string, tfRef string) string {
	quotedUUID := fmt.Sprintf(`"%s"`, uuid)
	quotedRef := fmt.Sprintf(`"%s"`, tfRef)

	return strings.Replace(string(value), quotedUUID, quotedRef, 1)
}
