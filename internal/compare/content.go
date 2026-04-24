// Package compare provides runtime HCL content comparison for shadow mode.
//
// Unlike the test-only comparison in tests/comparison/, this package returns
// structured diff results without requiring *testing.T.
package compare

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// Diff represents a single content difference between two HCL outputs.
type Diff struct {
	// Resource is the "type.label" key (e.g., "pingone_davinci_variable.my_var").
	Resource string

	// Kind describes the difference category.
	Kind DiffKind

	// Attribute is the attribute or block name, if applicable.
	Attribute string

	// Expected is the value from the legacy (reference) output.
	Expected string

	// Actual is the value from the new pipeline output.
	Actual string
}

// DiffKind categorises a content difference.
type DiffKind string

const (
	// DiffMissingResource means a resource present in expected is absent from actual.
	DiffMissingResource DiffKind = "missing_resource"

	// DiffExtraResource means a resource present in actual is absent from expected.
	DiffExtraResource DiffKind = "extra_resource"

	// DiffMissingAttribute means an attribute present in expected is absent from actual.
	DiffMissingAttribute DiffKind = "missing_attribute"

	// DiffValueMismatch means an attribute exists in both but values differ.
	DiffValueMismatch DiffKind = "value_mismatch"

	// DiffMissingBlock means a nested block present in expected is absent from actual.
	DiffMissingBlock DiffKind = "missing_block"

	// DiffBlockMismatch means a nested block exists in both but content differs.
	DiffBlockMismatch DiffKind = "block_mismatch"

	// DiffExtraAttribute means an attribute present in actual is absent from expected.
	DiffExtraAttribute DiffKind = "extra_attribute"
)

// DiffSeverity classifies how severe a diff is.
type DiffSeverity string

const (
	// SeverityAcceptable means the diff is an addition (extra resource/attribute).
	SeverityAcceptable DiffSeverity = "acceptable"

	// SeverityBreaking means the diff indicates a regression.
	SeverityBreaking DiffSeverity = "breaking"
)

// ClassifyDiff returns the severity of a diff.
func ClassifyDiff(d Diff) DiffSeverity {
	switch d.Kind {
	case DiffExtraResource, DiffExtraAttribute:
		return SeverityAcceptable
	case DiffBlockMismatch:
		switch d.Attribute {
		case "graph_data", "required_providers":
			return SeverityAcceptable
		}
		return SeverityBreaking
	default:
		return SeverityBreaking
	}
}

// Result holds the outcome of an HCL content comparison.
type Result struct {
	Diffs []Diff
}

// HasDiffs returns true when any content differences were found.
func (r *Result) HasDiffs() bool {
	return len(r.Diffs) > 0
}

// HasBreakingDiffs returns true when any breaking diff exists.
func (r *Result) HasBreakingDiffs() bool {
	for _, d := range r.Diffs {
		if ClassifyDiff(d) == SeverityBreaking {
			return true
		}
	}
	return false
}

// Summary returns a human-readable summary of all diffs.
func (r *Result) Summary() string {
	if !r.HasDiffs() {
		return "no content differences"
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d content difference(s):\n", len(r.Diffs)))
	for _, d := range r.Diffs {
		switch d.Kind {
		case DiffMissingResource:
			sb.WriteString(fmt.Sprintf("  - MISSING resource %s\n", d.Resource))
		case DiffExtraResource:
			sb.WriteString(fmt.Sprintf("  + EXTRA   resource %s\n", d.Resource))
		case DiffMissingAttribute:
			sb.WriteString(fmt.Sprintf("  - %s: attribute %q missing\n", d.Resource, d.Attribute))
		case DiffValueMismatch:
			sb.WriteString(fmt.Sprintf("  ~ %s: attribute %q: expected %q, got %q\n", d.Resource, d.Attribute, d.Expected, d.Actual))
		case DiffMissingBlock:
			sb.WriteString(fmt.Sprintf("  - %s: block %q missing\n", d.Resource, d.Attribute))
		case DiffBlockMismatch:
			sb.WriteString(fmt.Sprintf("  ~ %s: block %q content differs\n", d.Resource, d.Attribute))
		case DiffExtraAttribute:
			sb.WriteString(fmt.Sprintf("  + %s: attribute %q extra\n", d.Resource, d.Attribute))
		}
	}
	return sb.String()
}

// CompareHCL parses two HCL strings and returns content differences.
// expected is the reference (legacy) output; actual is the new pipeline output.
// Formatting differences (whitespace, blank lines, column alignment) are ignored.
func CompareHCL(expected, actual string) (*Result, error) {
	expResources, err := parseResources(expected)
	if err != nil {
		return nil, fmt.Errorf("parse expected HCL: %w", err)
	}
	actResources, err := parseResources(actual)
	if err != nil {
		return nil, fmt.Errorf("parse actual HCL: %w", err)
	}

	result := &Result{}

	// Check every expected resource exists in actual.
	expKeys := sortedMapKeys(expResources)
	for _, key := range expKeys {
		expRes := expResources[key]
		actRes, exists := actResources[key]
		if !exists {
			result.Diffs = append(result.Diffs, Diff{
				Resource: key,
				Kind:     DiffMissingResource,
			})
			continue
		}
		compareResource(result, key, expRes, actRes)
	}

	// Check for extra resources in actual that are not in expected.
	actKeys := sortedMapKeys(actResources)
	for _, key := range actKeys {
		if _, exists := expResources[key]; !exists {
			result.Diffs = append(result.Diffs, Diff{
				Resource: key,
				Kind:     DiffExtraResource,
			})
		}
	}

	return result, nil
}

// parsedResource holds parsed resource data for comparison.
type parsedResource struct {
	resType string
	label   string
	attrs   map[string]string // attribute name -> normalized value
	blocks  map[string]string // block name -> normalized body
}

// parseResources extracts resource blocks from HCL using hclwrite.
func parseResources(hclText string) (map[string]*parsedResource, error) {
	if strings.TrimSpace(hclText) == "" {
		return make(map[string]*parsedResource), nil
	}

	file, diags := hclwrite.ParseConfig([]byte(hclText), "compare.tf", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, fmt.Errorf("hclwrite parse: %s", diags.Error())
	}

	resources := make(map[string]*parsedResource)
	for _, block := range file.Body().Blocks() {
		if block.Type() != "resource" {
			continue
		}
		labels := block.Labels()
		if len(labels) < 2 {
			continue
		}
		key := labels[0] + "." + labels[1]
		resources[key] = extractBlock(labels[0], labels[1], block)
	}
	return resources, nil
}

// extractBlock pulls attributes and nested blocks from an hclwrite block.
func extractBlock(resType, label string, block *hclwrite.Block) *parsedResource {
	res := &parsedResource{
		resType: resType,
		label:   label,
		attrs:   make(map[string]string),
		blocks:  make(map[string]string),
	}

	body := block.Body()
	for name, attr := range body.Attributes() {
		tokens := attr.Expr().BuildTokens(nil)
		valStr := strings.TrimSpace(tokensToString(tokens))
		if isObjectLiteral(valStr) {
			res.blocks[name] = normalizeObjectLiteral(valStr)
		}
		res.attrs[name] = valStr
	}

	for _, nested := range body.Blocks() {
		blockType := nested.Type()
		if len(nested.Labels()) > 0 {
			blockType = blockType + " " + strings.Join(nested.Labels(), " ")
		}
		res.blocks[blockType] = normalizeBlockBody(nested)
	}

	return res
}

// compareResource appends diffs for attribute and block differences.
func compareResource(result *Result, key string, exp, act *parsedResource) {
	// Compare attributes.
	attrNames := sortedMapKeys(exp.attrs)
	for _, name := range attrNames {
		// Skip attributes that are object literals — they are compared via blocks.
		if _, isBlock := exp.blocks[name]; isBlock {
			continue
		}
		expVal := normalizeValue(exp.attrs[name])
		actVal, exists := act.attrs[name]
		if !exists {
			result.Diffs = append(result.Diffs, Diff{
				Resource:  key,
				Kind:      DiffMissingAttribute,
				Attribute: name,
				Expected:  expVal,
			})
			continue
		}
		actVal = normalizeValue(actVal)
		if expVal != actVal {
			result.Diffs = append(result.Diffs, Diff{
				Resource:  key,
				Kind:      DiffValueMismatch,
				Attribute: name,
				Expected:  expVal,
				Actual:    actVal,
			})
		}
	}

	// Check for extra attributes in actual that are not in expected.
	actAttrNames := sortedMapKeys(act.attrs)
	for _, name := range actAttrNames {
		if _, isBlock := act.blocks[name]; isBlock {
			continue
		}
		if _, exists := exp.attrs[name]; !exists {
			actVal := normalizeValue(act.attrs[name])
			result.Diffs = append(result.Diffs, Diff{
				Resource:  key,
				Kind:      DiffExtraAttribute,
				Attribute: name,
				Actual:    actVal,
			})
		}
	}

	// Compare nested blocks.
	blockNames := sortedMapKeys(exp.blocks)
	for _, name := range blockNames {
		actBlock, exists := act.blocks[name]
		if !exists {
			result.Diffs = append(result.Diffs, Diff{
				Resource:  key,
				Kind:      DiffMissingBlock,
				Attribute: name,
			})
			continue
		}
		if exp.blocks[name] != actBlock {
			result.Diffs = append(result.Diffs, Diff{
				Resource:  key,
				Kind:      DiffBlockMismatch,
				Attribute: name,
				Expected:  exp.blocks[name],
				Actual:    actBlock,
			})
		}
	}
}

// ── helpers ──

func tokensToString(tokens hclwrite.Tokens) string {
	var sb strings.Builder
	for _, tok := range tokens {
		sb.Write(tok.Bytes)
	}
	return sb.String()
}

func normalizeValue(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
		v = v[1 : len(v)-1]
	}
	return v
}

func isObjectLiteral(s string) bool {
	s = strings.TrimSpace(s)
	return strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")
}

func normalizeObjectLiteral(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '{' && s[len(s)-1] == '}' {
		s = s[1 : len(s)-1]
	}
	var parts []string
	for _, line := range strings.Split(s, "\n") {
		t := strings.TrimSpace(line)
		if t == "" {
			continue
		}
		p := strings.SplitN(t, "=", 2)
		if len(p) == 2 {
			key := normalizeValue(strings.TrimSpace(p[0]))
			parts = append(parts, key+" = "+strings.TrimSpace(p[1]))
		}
	}
	sort.Strings(parts)
	return strings.Join(parts, "\n")
}

func normalizeBlockBody(block *hclwrite.Block) string {
	var parts []string
	body := block.Body()

	names := make([]string, 0, len(body.Attributes()))
	for name := range body.Attributes() {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		attr := body.Attributes()[name]
		tokens := attr.Expr().BuildTokens(nil)
		valStr := strings.TrimSpace(tokensToString(tokens))
		parts = append(parts, fmt.Sprintf("%s = %s", name, valStr))
	}
	return strings.Join(parts, "\n")
}

func sortedMapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// CompareModuleResources compares two sets of per-type HCL strings and returns
// a combined Result. Each field (variables, connections, flows, applications,
// flow_policies) is compared independently.
func CompareModuleResources(expected, actual map[string]string) (*Result, error) {
	combined := &Result{}
	keys := sortedMapKeys(expected)
	for _, key := range keys {
		expHCL := expected[key]
		actHCL := actual[key]
		res, err := CompareHCL(expHCL, actHCL)
		if err != nil {
			return nil, fmt.Errorf("compare %s: %w", key, err)
		}
		combined.Diffs = append(combined.Diffs, res.Diffs...)
	}

	// Check for resource types in actual that are not in expected.
	for _, key := range sortedMapKeys(actual) {
		if _, exists := expected[key]; !exists {
			actHCL := actual[key]
			res, err := CompareHCL("", actHCL)
			if err != nil {
				return nil, fmt.Errorf("compare extra %s: %w", key, err)
			}
			combined.Diffs = append(combined.Diffs, res.Diffs...)
		}
	}

	return combined, nil
}

// CompareHCLGeneric parses two HCL strings and returns content differences
// for all block types (resource, module, variable, output, provider, data,
// locals, terraform, import, etc.).
func CompareHCLGeneric(expected, actual string) (*Result, error) {
	expBlocks, expTopAttrs, err := parseBlocksGeneric(expected)
	if err != nil {
		return nil, fmt.Errorf("parse expected HCL: %w", err)
	}
	actBlocks, actTopAttrs, err := parseBlocksGeneric(actual)
	if err != nil {
		return nil, fmt.Errorf("parse actual HCL: %w", err)
	}

	result := &Result{}

	// Compare blocks.
	for _, key := range sortedMapKeys(expBlocks) {
		expRes := expBlocks[key]
		actRes, exists := actBlocks[key]
		if !exists {
			result.Diffs = append(result.Diffs, Diff{
				Resource: key,
				Kind:     DiffMissingResource,
			})
			continue
		}
		compareResource(result, key, expRes, actRes)
	}

	for _, key := range sortedMapKeys(actBlocks) {
		if _, exists := expBlocks[key]; !exists {
			result.Diffs = append(result.Diffs, Diff{
				Resource: key,
				Kind:     DiffExtraResource,
			})
		}
	}

	// Compare top-level attributes (e.g., .tfvars files).
	if len(expTopAttrs) > 0 || len(actTopAttrs) > 0 {
		topResource := &parsedResource{attrs: expTopAttrs, blocks: make(map[string]string)}
		actTopResource := &parsedResource{attrs: actTopAttrs, blocks: make(map[string]string)}
		compareResource(result, "top-level", topResource, actTopResource)
	}

	return result, nil
}

func parseBlocksGeneric(hclText string) (map[string]*parsedResource, map[string]string, error) {
	if strings.TrimSpace(hclText) == "" {
		return make(map[string]*parsedResource), make(map[string]string), nil
	}

	file, diags := hclwrite.ParseConfig([]byte(hclText), "compare.tf", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, nil, fmt.Errorf("hclwrite parse: %s", diags.Error())
	}

	blocks := make(map[string]*parsedResource)
	for _, block := range file.Body().Blocks() {
		key := genericBlockKey(block)
		if key == "" {
			continue
		}
		labels := block.Labels()
		resType := block.Type()
		label := ""
		if len(labels) > 0 {
			label = labels[len(labels)-1]
		}
		blocks[key] = extractBlock(resType, label, block)
	}

	// Extract top-level attributes (for .tfvars and similar files).
	topAttrs := make(map[string]string)
	for name, attr := range file.Body().Attributes() {
		tokens := attr.Expr().BuildTokens(nil)
		topAttrs[name] = strings.TrimSpace(tokensToString(tokens))
	}

	return blocks, topAttrs, nil
}

func genericBlockKey(block *hclwrite.Block) string {
	blockType := block.Type()
	labels := block.Labels()

	switch blockType {
	case "resource":
		if len(labels) >= 2 {
			return blockType + "." + labels[0] + "." + labels[1]
		}
	case "data":
		if len(labels) >= 2 {
			return blockType + "." + labels[0] + "." + labels[1]
		}
	case "module", "variable", "output", "provider":
		if len(labels) >= 1 {
			return blockType + "." + labels[0]
		}
	case "locals", "terraform":
		return blockType
	case "import":
		for name, attr := range block.Body().Attributes() {
			if name == "to" {
				tokens := attr.Expr().BuildTokens(nil)
				val := normalizeValue(strings.TrimSpace(tokensToString(tokens)))
				return "import." + val
			}
		}
		return "import"
	}
	// Fallback: blockType + labels
	if len(labels) > 0 {
		return blockType + "." + strings.Join(labels, ".")
	}
	return blockType
}
