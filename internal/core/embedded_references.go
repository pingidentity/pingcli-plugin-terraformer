package core

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
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

	// PreconditionKeyPath is an optional JSON key path (dot-notation, resolved
	// against the same parsed JSON blob as JSONKeyPath) that must resolve to
	// exactly PreconditionValue before this rule fires. Zero value ("") means
	// no precondition — the rule fires unconditionally, matching pre-existing
	// behavior. Used, for example, to gate a rule on a sibling mode-flag key
	// (e.g., only act on "themeId.value" when "theme.value" == "useThemeId").
	PreconditionKeyPath string

	// PreconditionValue is the exact string PreconditionKeyPath must resolve to
	// for the rule to fire. Ignored when PreconditionKeyPath is empty.
	PreconditionValue string

	// UnwrapMode controls how the value at JSONKeyPath is extracted and
	// re-embedded:
	//   ""          (default / zero value) — JSONKeyPath resolves directly to
	//                a plain string (pre-existing behavior).
	//   "rich_text" — the value at JSONKeyPath is itself a JSON string
	//                containing a Slate-style rich-text wrapper
	//                (`[{"children":[{"text":"<value>"}]}]`); the inner value
	//                is unwrapped before resolution and the resolved
	//                reference/variable is re-embedded back inside the
	//                wrapper on write.
	UnwrapMode string
}

// richTextUnwrapMode is the UnwrapMode value that enables Slate-style
// rich-text unwrap/rewrap of the value found at JSONKeyPath.
const richTextUnwrapMode = "rich_text"

// looksLikeUUID reports whether s is formatted as a valid UUID. It is used as
// an unconditional guard before any extracted (or unwrapped) string is ever
// treated as a resolvable reference/fallback-variable target — this rejects
// sentinel/mode-flag strings (e.g., "useThemeId", "activeTheme") without
// needing to enumerate them. Named to avoid colliding with the common local
// variable name "uuid" used at call sites for the extracted value.
func looksLikeUUID(s string) bool {
	return uuid.Validate(s) == nil
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

	// Shared across all rules/resources so that two different UUIDs which
	// derive the same human-readable variable name (e.g. two DaVinci nodes
	// both titled "Continue") are disambiguated instead of silently
	// colliding onto one variable (see #130).
	allocator := newFallbackVariableAllocator()

	for _, exportedData := range results {
		for _, rule := range rules {
			// Skip if resource type doesn't match
			if rule.ResourceType != exportedData.ResourceType {
				continue
			}

			// Process each resource of this type
			for _, resource := range exportedData.Resources {
				processResourceWithRule(resource, rule, g, allocator, varSeen, &fallbackVars)
			}
		}
	}

	return fallbackVars
}

// processResourceWithRule applies a single rule to a resource
func processResourceWithRule(resource *ResourceData, rule EmbeddedReferenceRule, g *graph.DependencyGraph, allocator *fallbackVariableAllocator, varSeen map[string]bool, fallbackVars *[]FallbackVariable) {
	// Parse the attribute path into segments
	pathSegments := strings.Split(rule.AttributePath, ".")

	// Walk the path and process all matching RawHCLValues
	walkAndProcessPath(resource.Attributes, pathSegments, 0, rule, resource, g, allocator, "", varSeen, fallbackVars)
}

// walkAndProcessPath recursively walks the attribute path and processes
// matching values. traversalKey accumulates the wildcard-matched map keys
// seen so far (e.g. a DaVinci node's own "data.id") — these are stable,
// environment-portable identifiers for whatever is being iterated, and are
// passed down to processRawHCLValue as a disambiguation hint distinct from
// the target UUID being resolved.
func walkAndProcessPath(
	current interface{},
	pathSegments []string,
	segmentIndex int,
	rule EmbeddedReferenceRule,
	resource *ResourceData,
	g *graph.DependencyGraph,
	allocator *fallbackVariableAllocator,
	traversalKey string,
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
			// Wildcard: process all keys in this map. The map key itself
			// becomes (part of) the disambiguation hint for whatever is
			// found beneath it.
			for key := range typedCurrent {
				nextValue := typedCurrent[key]
				nextTraversalKey := key
				if traversalKey != "" {
					nextTraversalKey = traversalKey + "_" + key
				}
				if isLastSegment {
					// This key should be a RawHCLValue - try to process it
					if rawValue, ok := nextValue.(RawHCLValue); ok {
						processedValue := processRawHCLValue(rawValue, rule, resource, g, allocator, nextTraversalKey, varSeen, fallbackVars)
						typedCurrent[key] = processedValue
					}
				} else {
					// Continue walking deeper
					walkAndProcessPath(nextValue, pathSegments, segmentIndex+1, rule, resource, g, allocator, nextTraversalKey, varSeen, fallbackVars)
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
					processedValue := processRawHCLValue(rawValue, rule, resource, g, allocator, traversalKey, varSeen, fallbackVars)
					typedCurrent[segment] = processedValue
				}
			} else {
				// Continue walking deeper
				walkAndProcessPath(nextValue, pathSegments, segmentIndex+1, rule, resource, g, allocator, traversalKey, varSeen, fallbackVars)
			}
		}
	}
}

// processRawHCLValue extracts JSON from the RawHCLValue, finds the UUID at JSONKeyPath,
// and replaces it with a Terraform reference or variable depending on the rule's Strategy.
// traversalKey is a stable, environment-portable identifier (e.g. a DaVinci
// node's own "data.id", accumulated by walkAndProcessPath's wildcard steps)
// used to disambiguate fallback variable names when two different reference
// sites derive the same human-readable base name — see allocatedVariableName.
func processRawHCLValue(
	value RawHCLValue,
	rule EmbeddedReferenceRule,
	resource *ResourceData,
	g *graph.DependencyGraph,
	allocator *fallbackVariableAllocator,
	traversalKey string,
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

	// Precondition: a sibling JSON key path must resolve to exactly
	// PreconditionValue before this rule fires. Zero-value PreconditionKeyPath
	// means no precondition — fires unconditionally, matching pre-existing
	// behavior.
	if rule.PreconditionKeyPath != "" {
		if walkJSONPath(jsonData, rule.PreconditionKeyPath) != rule.PreconditionValue {
			return value
		}
	}

	// Walk the JSON path to find the value. In "rich_text" UnwrapMode this is
	// the Slate wrapper string; otherwise it is the plain UUID string.
	extracted := walkJSONPath(jsonData, rule.JSONKeyPath)
	if extracted == "" {
		return value
	}

	var wrapper string
	uuidStr := extracted
	if rule.UnwrapMode == richTextUnwrapMode {
		wrapper = extracted
		uuidStr = unwrapRichText(wrapper)
		if uuidStr == "" {
			// Malformed/unexpected wrapper shape — no-op, no panic.
			return value
		}
	}

	// Unconditional UUID-format guard: only strings shaped like a UUID are
	// ever treated as a resolvable reference/fallback-variable target. Any
	// other string (e.g. a mode-flag sentinel such as "useThemeId" or
	// "activeTheme") no-ops exactly like "no value found", regardless of
	// Strategy.
	if !looksLikeUUID(uuidStr) {
		return value
	}

	// Strategy: "variable" — always emit a variable, skip graph lookup
	if rule.Strategy == "variable" {
		varName := allocatedVariableName(rule, jsonData, uuidStr, traversalKey, allocator)
		tfRef := fmt.Sprintf("${var.%s}", varName)
		newValue := replaceExtractedValue(value, rule, wrapper, uuidStr, tfRef)
		addEmbeddedFallbackVariable(varName, rule, uuidStr, varSeen, fallbackVars)
		return newValue
	}

	// Strategy: "reference" (default) or "reference_with_fallback" — try graph lookup
	refName, err := g.GetReferenceName(rule.TargetResourceType, uuidStr)
	if err != nil {
		// UUID not found in graph
		if rule.Strategy == "reference_with_fallback" {
			varName := allocatedVariableName(rule, jsonData, uuidStr, traversalKey, allocator)
			tfRef := fmt.Sprintf("${var.%s}", varName)
			newValue := replaceExtractedValue(value, rule, wrapper, uuidStr, tfRef)
			addEmbeddedFallbackVariable(varName, rule, uuidStr, varSeen, fallbackVars)
			return newValue
		}
		// Default strategy: leave unchanged
		return value
	}

	// Build the terraform reference string
	tfRef := fmt.Sprintf("${%s.%s.%s}", rule.TargetResourceType, refName, rule.ReferenceField)

	// Replace the UUID string in the RawHCLValue with the terraform reference
	newValue := replaceExtractedValue(value, rule, wrapper, uuidStr, tfRef)

	// Add graph edge
	_ = g.AddEdge(resource.ResourceType, resource.ID, rule.TargetResourceType, uuidStr, "properties."+rule.JSONKeyPath, "")

	return newValue
}

// replaceExtractedValue re-embeds a resolved reference/variable string in
// place of the originally extracted value. It chooses the escaping-aware
// rich-text rewrap path when the rule uses UnwrapMode == "rich_text", and
// falls back to the plain substring replace (replaceUUIDInRawHCL) otherwise.
func replaceExtractedValue(value RawHCLValue, rule EmbeddedReferenceRule, wrapper string, uuidStr string, tfRef string) RawHCLValue {
	if rule.UnwrapMode == richTextUnwrapMode {
		return replaceRichTextInRawHCL(value, wrapper, uuidStr, tfRef)
	}
	return RawHCLValue(replaceUUIDInRawHCL(value, uuidStr, tfRef))
}

// unwrapRichText extracts the inner "text" value from a Slate-style
// rich-text wrapper of the shape `[{"children":[{"text":"<value>"}]}]`. Only
// the first array element's first child is read, matching the confirmed
// evidence shape. Returns "" on any shape mismatch (not an array, empty
// array, missing "children", missing/non-string "text") — mirroring
// walkJSONPath's existing "return empty on mismatch" convention. Never
// panics on malformed input.
func unwrapRichText(wrapper string) string {
	var elements []interface{}
	if err := json.Unmarshal([]byte(wrapper), &elements); err != nil {
		return ""
	}
	if len(elements) == 0 {
		return ""
	}

	elem, ok := elements[0].(map[string]interface{})
	if !ok {
		return ""
	}

	childrenRaw, exists := elem["children"]
	if !exists {
		return ""
	}
	children, ok := childrenRaw.([]interface{})
	if !ok || len(children) == 0 {
		return ""
	}

	child, ok := children[0].(map[string]interface{})
	if !ok {
		return ""
	}

	text, ok := child["text"].(string)
	if !ok {
		return ""
	}

	return text
}

// replaceRichTextInRawHCL re-embeds a resolved reference/variable string
// inside a Slate-style rich-text wrapper and substitutes the escaped wrapper
// text within the raw HCL string. Unlike replaceUUIDInRawHCL's plain
// substring replace, this accounts for the wrapper being JSON-encoded a
// second time when the outer `properties` map was marshaled by
// transformJSONEncodeRaw — so quotes inside the wrapper appear
// backslash-escaped in the RawHCLValue text. Re-deriving both the old and
// new wrapper's escaped form via json.Marshal (rather than hand-escaping)
// normalizes the escaping identically to how the original text was produced.
func replaceRichTextInRawHCL(value RawHCLValue, wrapper string, uuidStr string, tfRef string) RawHCLValue {
	newWrapper := strings.Replace(wrapper, uuidStr, tfRef, 1)

	oldEscaped, err := json.Marshal(wrapper)
	if err != nil {
		return value
	}
	newEscaped, err := json.Marshal(newWrapper)
	if err != nil {
		return value
	}

	return RawHCLValue(strings.Replace(string(value), string(oldEscaped), string(newEscaped), 1))
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

// allocatedVariableName derives the base variable name via deriveVariableName
// and then resolves it through allocator, keyed on uuid. This guarantees that
// two different UUIDs which happen to derive the same human-readable name
// (e.g. two DaVinci nodes both titled "Continue") get distinct variables
// instead of silently colliding onto one (see #130) — while repeated calls
// for the same UUID still return the same name.
//
// traversalKey (the wildcard-matched map key(s) leading to this value, e.g.
// a DaVinci node's own "data.id") is used as the disambiguation hint rather
// than the UUID itself: the UUID is per-environment API data, so baking it
// into the variable *name* would make the same logical reference site
// produce a different variable name in every environment export. The
// traversal key is stable across environments for the same exported
// configuration. Falls back to a UUID prefix only if no traversal key was
// captured (defensive — every wildcard-based rule populates one).
func allocatedVariableName(rule EmbeddedReferenceRule, jsonData interface{}, uuid string, traversalKey string, allocator *fallbackVariableAllocator) string {
	baseName := deriveVariableName(rule, jsonData, uuid)
	hint := traversalKey
	if hint == "" {
		hint = uuid
	}
	return allocator.allocate(uuid, baseName, hint)
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
