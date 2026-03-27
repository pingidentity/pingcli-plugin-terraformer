// Package hcl generates Terraform HCL from processed resource data using hclwrite.
package hcl

import (
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/core"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/utils"
)

// FormatOptions controls HCL rendering behavior.
type FormatOptions struct {
	// SkipDependencies outputs raw UUIDs instead of Terraform references.
	SkipDependencies bool
	// EnvironmentID is the raw environment UUID used when SkipDependencies is true.
	EnvironmentID string
}

// Formatter generates HCL output from processed resource data using hclwrite.
type Formatter struct{}

// NewFormatter creates a new HCL formatter.
func NewFormatter() *Formatter {
	return &Formatter{}
}

// Format generates a single Terraform resource block from resource data.
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

	file := hclwrite.NewEmptyFile()
	block := file.Body().AppendNewBlock("resource", []string{def.Metadata.ResourceType, label})
	body := block.Body()

	for _, attrDef := range def.Attributes {
		// Skip computed-only attributes such as the imported id field.
		if attrDef.Computed && !attrDef.Required && attrDef.ReferencesType == "" {
			continue
		}

		tName := terraformName(attrDef)

		// Reference attributes: emit resolved reference or raw UUID.
		if attrDef.ReferencesType != "" {
			val, valOK := data.Attributes[tName]
			if !valOK || val == nil {
				continue
			}
			// ResolvedReference: render as traversal expression.
			if ref, ok := val.(core.ResolvedReference); ok {
				writeTraversalExpression(body, tName, ref.Expression())
				continue
			}
			// Fallback: raw string (SkipDependencies mode or unresolved).
			if raw, ok := val.(string); ok && raw != "" {
				if strings.HasPrefix(raw, "var.") || strings.Contains(raw, ".") {
					writeTraversalExpression(body, tName, raw)
				} else {
					body.SetAttributeValue(tName, cty.StringVal(raw))
				}
			}
			continue
		}

		// Nested object block.
		if attrDef.Type == "object" && len(attrDef.NestedAttributes) > 0 {
			writeObjectBlock(body, attrDef, data, opts)
			continue
		}

		// Map block with nested schema (e.g. graph_data.elements.nodes).
		if attrDef.Type == "map" && len(attrDef.NestedAttributes) > 0 {
			writeMapBlock(body, attrDef, data, opts)
			continue
		}

		// List of objects with nested schema (e.g. input_schema).
		if (attrDef.Type == "list" || attrDef.Type == "set") && len(attrDef.NestedAttributes) > 0 {
			writeListOfObjectsBlock(body, attrDef, data, opts)
			continue
		}

		// Dynamic object block (object type without nested_attributes, or type_discriminated_block).
		if (attrDef.Type == "object" && len(attrDef.NestedAttributes) == 0) || attrDef.Type == "type_discriminated_block" {
			val, ok := data.Attributes[tName]
			if !ok || val == nil {
				continue
			}
			// Custom transforms may return pre-rendered HCL for complex structures.
			if rv, ok := val.(core.RawHCLValue); ok {
				writeRawAttribute(body, tName, rv)
				continue
			}
			if valMap, ok := val.(map[string]interface{}); ok {
				writeDynamicBlock(body, tName, valMap)
			}
			continue
		}

		// Scalar attribute.
		val, ok := data.Attributes[tName]
		if !ok || val == nil {
			continue
		}
		// ResolvedReference: write as unquoted traversal (e.g. var.my_variable).
		if ref, ok := val.(core.ResolvedReference); ok {
			writeTraversalExpression(body, tName, ref.Expression())
			continue
		}
		writeScalarValue(body, tName, val)
	}

	rawBytes := file.Bytes()
	formatted := hclwrite.Format(rawBytes)
	return string(formatted), nil
}

// FormatImportBlock generates a Terraform 1.5+ import block for a resource.
// Returns an empty string (no error) when the definition has no import_id_format,
// which indicates the resource does not support Terraform import.
func (f *Formatter) FormatImportBlock(data *core.ResourceData, def *schema.ResourceDefinition, environmentID string) (string, error) {
	if data == nil {
		return "", fmt.Errorf("resource data is nil")
	}
	if def == nil {
		return "", fmt.Errorf("resource definition is nil")
	}
	if def.Dependencies.ImportIDFormat == "" {
		return "", nil
	}

	label := resourceLabel(data, def)

	importID := def.Dependencies.ImportIDFormat
	importID = strings.ReplaceAll(importID, "{env_id}", environmentID)
	importID = strings.ReplaceAll(importID, "{resource_id}", data.ID)
	// Expand any remaining {attr_name} placeholders from resource attributes.
	for k, v := range data.Attributes {
		if s, ok := v.(string); ok {
			importID = strings.ReplaceAll(importID, "{"+k+"}", s)
		} else if ref, ok := v.(core.ResolvedReference); ok && ref.OriginalValue != "" {
			importID = strings.ReplaceAll(importID, "{"+k+"}", ref.OriginalValue)
		}
	}

	file := hclwrite.NewEmptyFile()
	block := file.Body().AppendNewBlock("import", nil)
	body := block.Body()

	// Write: to = resource_type.label (unquoted reference)
	writeTraversalExpression(body, "to", def.Metadata.ResourceType+"."+label)
	body.SetAttributeValue("id", cty.StringVal(importID))

	return string(hclwrite.Format(file.Bytes())), nil
}

// FormatList generates HCL resource blocks for a slice of resource data,
// sorted deterministically by resource name.
func (f *Formatter) FormatList(dataList []*core.ResourceData, def *schema.ResourceDefinition, opts FormatOptions) (string, error) {
	sorted := make([]*core.ResourceData, len(dataList))
	copy(sorted, dataList)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})

	var sb strings.Builder
	for i, data := range sorted {
		if i > 0 {
			sb.WriteString("\n")
		}
		hcl, err := f.Format(data, def, opts)
		if err != nil {
			return "", fmt.Errorf("resource %s: %w", data.Name, err)
		}
		sb.WriteString(hcl)
	}
	return sb.String(), nil
}

// writeTraversalExpression writes an unquoted expression (e.g. var.foo, resource.label.attr)
// as raw tokens on the given body.
func writeTraversalExpression(body *hclwrite.Body, name, expr string) {
	tokens := hclwrite.Tokens{
		{Type: hclsyntax.TokenIdent, Bytes: []byte(expr)},
	}
	body.SetAttributeRaw(name, tokens)
}

// writeObjectBlock writes a nested object as an HCL object attribute assignment.
func writeObjectBlock(parentBody *hclwrite.Body, attrDef schema.AttributeDefinition, data *core.ResourceData, opts FormatOptions) {
	tName := terraformName(attrDef)
	val, ok := data.Attributes[tName]
	if !ok || val == nil {
		return
	}

	valMap, ok := val.(map[string]interface{})
	if !ok {
		return
	}

	tokens := nestedObjectTokens("    ", "  ", attrDef.NestedAttributes, valMap, opts)
	if len(tokens) == 0 {
		return
	}

	var all hclwrite.Tokens
	all = append(all, &hclwrite.Token{Type: hclsyntax.TokenOBrace, Bytes: []byte("{")})
	all = append(all, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
	all = append(all, tokens...)
	all = append(all, &hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte("  }")})
	parentBody.AppendNewline()
	parentBody.SetAttributeRaw(tName, all)
}

// writeMapBlock writes a map with nested schema as an HCL map attribute.
// Used for attributes like graph_data.elements.nodes where the API
// returns a slice that was converted to a map keyed by ID.
func writeMapBlock(parentBody *hclwrite.Body, attrDef schema.AttributeDefinition, data *core.ResourceData, opts FormatOptions) {
	tName := terraformName(attrDef)
	val, ok := data.Attributes[tName]
	if !ok || val == nil {
		return
	}

	outerMap, ok := val.(map[string]interface{})
	if !ok || len(outerMap) == 0 {
		return
	}

	mapKeys := make([]string, 0, len(outerMap))
	for k := range outerMap {
		mapKeys = append(mapKeys, k)
	}
	sort.Strings(mapKeys)

	var all hclwrite.Tokens
	all = append(all, &hclwrite.Token{Type: hclsyntax.TokenOBrace, Bytes: []byte("{")})
	all = append(all, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})

	entryIndent := "    "
	childIndent := "      "

	for _, k := range mapKeys {
		mv, ok := outerMap[k].(map[string]interface{})
		if !ok {
			continue
		}
		subTokens := nestedObjectTokens(childIndent, entryIndent, attrDef.NestedAttributes, mv, opts)
		if len(subTokens) == 0 {
			continue
		}
		all = append(all, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(entryIndent + fmt.Sprintf("%q", k))})
		all = append(all, &hclwrite.Token{Type: hclsyntax.TokenEqual, Bytes: []byte(" = ")})
		all = append(all, &hclwrite.Token{Type: hclsyntax.TokenOBrace, Bytes: []byte("{")})
		all = append(all, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
		all = append(all, subTokens...)
		all = append(all, &hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte(entryIndent + "}")})
		all = append(all, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
	}

	all = append(all, &hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte("  }")})
	parentBody.AppendNewline()
	parentBody.SetAttributeRaw(tName, all)
}

// writeListOfObjectsBlock writes a list of schema-driven objects as an HCL list attribute.
// Used for attributes like input_schema where each list item has known nested attributes.
func writeListOfObjectsBlock(parentBody *hclwrite.Body, attrDef schema.AttributeDefinition, data *core.ResourceData, opts FormatOptions) {
	tName := terraformName(attrDef)
	val, ok := data.Attributes[tName]
	if !ok || val == nil {
		return
	}

	slice, ok := val.([]interface{})
	if !ok || len(slice) == 0 {
		return
	}

	var all hclwrite.Tokens
	all = append(all, &hclwrite.Token{Type: hclsyntax.TokenOBrack, Bytes: []byte("[")})
	all = append(all, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})

	entryIndent := "    "
	childIndent := "      "

	for i, item := range slice {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		subTokens := nestedObjectTokens(childIndent, entryIndent, attrDef.NestedAttributes, itemMap, opts)
		if len(subTokens) == 0 {
			continue
		}
		all = append(all, &hclwrite.Token{Type: hclsyntax.TokenOBrace, Bytes: []byte(entryIndent + "{")})
		all = append(all, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
		all = append(all, subTokens...)
		all = append(all, &hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte(entryIndent + "}")})
		if i < len(slice)-1 {
			all = append(all, &hclwrite.Token{Type: hclsyntax.TokenComma, Bytes: []byte(",")})
		}
		all = append(all, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
	}

	all = append(all, &hclwrite.Token{Type: hclsyntax.TokenCBrack, Bytes: []byte("  ]")})
	parentBody.AppendNewline()
	parentBody.SetAttributeRaw(tName, all)
}

// nestedObjectTokens recursively generates HCL tokens for nested attributes.
// indent is the whitespace prefix for attribute lines (e.g. "    ", "      ").
// closingIndent is the whitespace prefix for the closing brace of child objects.
func nestedObjectTokens(indent, closingIndent string, nested []schema.AttributeDefinition, valMap map[string]interface{}, opts FormatOptions) hclwrite.Tokens {
	var tokens hclwrite.Tokens

	for _, attr := range nested {
		nName := terraformName(attr)
		nVal, nOk := valMap[nName]
		if !nOk || nVal == nil {
			continue
		}

		// ResolvedReference in nested context: emit as traversal expression.
		if ref, ok := nVal.(core.ResolvedReference); ok {
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(indent + nName)})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenEqual, Bytes: []byte(" = ")})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(ref.Expression())})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
			continue
		}

		// Nested object: recurse.
		if attr.Type == "object" && len(attr.NestedAttributes) > 0 {
			subMap, ok := nVal.(map[string]interface{})
			if !ok || len(subMap) == 0 {
				continue
			}
			childIndent := indent + "  "
			subTokens := nestedObjectTokens(childIndent, indent, attr.NestedAttributes, subMap, opts)
			if len(subTokens) == 0 {
				continue
			}
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(indent + nName)})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenEqual, Bytes: []byte(" = ")})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOBrace, Bytes: []byte("{")})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
			tokens = append(tokens, subTokens...)
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte(indent + "}")})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
			continue
		}

		// Nested map with schema: render as name = { "key1" = { ... }, "key2" = { ... } }
		if attr.Type == "map" && len(attr.NestedAttributes) > 0 {
			outerMap, ok := nVal.(map[string]interface{})
			if !ok || len(outerMap) == 0 {
				continue
			}
			mapKeys := make([]string, 0, len(outerMap))
			for k := range outerMap {
				mapKeys = append(mapKeys, k)
			}
			sort.Strings(mapKeys)

			childIndent := indent + "  "
			entryIndent := childIndent + "  "

			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(indent + nName)})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenEqual, Bytes: []byte(" = ")})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOBrace, Bytes: []byte("{")})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})

			for _, mk := range mapKeys {
				mv, ok := outerMap[mk].(map[string]interface{})
				if !ok {
					continue
				}
				subTokens := nestedObjectTokens(entryIndent, childIndent, attr.NestedAttributes, mv, opts)
				if len(subTokens) == 0 {
					continue
				}
				tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(childIndent + fmt.Sprintf("%q", mk))})
				tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenEqual, Bytes: []byte(" = ")})
				tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOBrace, Bytes: []byte("{")})
				tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
				tokens = append(tokens, subTokens...)
				tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte(childIndent + "}")})
				tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
			}

			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte(indent + "}")})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
			continue
		}

		// Nested list/set of objects: render as name = [ { ... }, { ... } ]
		if (attr.Type == "list" || attr.Type == "set") && len(attr.NestedAttributes) > 0 {
			slice, ok := nVal.([]interface{})
			if !ok || len(slice) == 0 {
				continue
			}
			childIndent := indent + "  "

			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(indent + nName)})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenEqual, Bytes: []byte(" = ")})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOBrack, Bytes: []byte("[")})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})

			for i, item := range slice {
				itemMap, ok := item.(map[string]interface{})
				if !ok {
					continue
				}
				subTokens := nestedObjectTokens(childIndent+"  ", childIndent, attr.NestedAttributes, itemMap, opts)
				if len(subTokens) == 0 {
					continue
				}
				tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOBrace, Bytes: []byte(childIndent + "{")})
				tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
				tokens = append(tokens, subTokens...)
				tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte(childIndent + "}")})
				if i < len(slice)-1 {
					tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenComma, Bytes: []byte(",")})
				}
				tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
			}

			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCBrack, Bytes: []byte(indent + "]")})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
			continue
		}

		// Scalar nested attribute.
		tokens = append(tokens, scalarTokens(indent, nName, nVal)...)
	}

	return tokens
}

// writeDynamicBlock writes a map as an HCL object attribute assignment.
// Used for object types without nested_attributes (e.g. variable value blocks).
// Values of type core.RawHCLValue are written without quoting.
func writeDynamicBlock(parentBody *hclwrite.Body, name string, m map[string]interface{}) {
	if len(m) == 0 {
		return
	}

	// Sort keys for deterministic output.
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var tokens hclwrite.Tokens
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOBrace, Bytes: []byte("{")})
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})

	for _, k := range keys {
		v := m[k]
		switch val := v.(type) {
		case core.RawHCLValue:
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("    " + k)})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenEqual, Bytes: []byte(" = ")})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(string(val))})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
		default:
			tokens = append(tokens, scalarTokens("    ", k, v)...)
		}
	}

	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte("  }")})
	parentBody.AppendNewline()
	parentBody.SetAttributeRaw(name, tokens)
}

// scalarTokens returns tokens for a single "indent name = value\n" line.
func scalarTokens(indent, name string, val interface{}) hclwrite.Tokens {
	var tokens hclwrite.Tokens
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(indent + name)})
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenEqual, Bytes: []byte(" = ")})

	switch v := val.(type) {
	case core.ResolvedReference:
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(v.Expression())})
	case string:
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenQuotedLit, Bytes: []byte(fmt.Sprintf("%q", v))})
	case bool:
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(fmt.Sprintf("%t", v))})
	case int64:
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNumberLit, Bytes: []byte(fmt.Sprintf("%d", v))})
	case float64:
		if v == float64(int64(v)) {
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNumberLit, Bytes: []byte(fmt.Sprintf("%d", int64(v)))})
		} else {
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNumberLit, Bytes: []byte(fmt.Sprintf("%g", v))})
		}
	case []interface{}:
		tokens = append(tokens, listTokens(v)...)
	case core.RawHCLValue:
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(string(v))})
	default:
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(fmt.Sprintf("%v", val))})
	}

	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
	return tokens
}

// listTokens generates HCL tokens for a list value: ["a", "b", "c"].
func listTokens(items []interface{}) hclwrite.Tokens {
	var tokens hclwrite.Tokens
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOBrack, Bytes: []byte("[")})
	for i, item := range items {
		if i > 0 {
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenComma, Bytes: []byte(", ")})
		}
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenQuotedLit, Bytes: []byte(fmt.Sprintf("%q", item))})
	}
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCBrack, Bytes: []byte("]")})
	return tokens
}

// writeRawAttribute writes a pre-rendered HCL value as a raw attribute.
// Used by custom transforms that produce complex structures (graph_data, settings, etc.).
func writeRawAttribute(body *hclwrite.Body, name string, rv core.RawHCLValue) {
	body.SetAttributeRaw(name, hclwrite.Tokens{
		{Type: hclsyntax.TokenIdent, Bytes: []byte(string(rv))},
	})
}

// writeScalarValue writes a single attribute using hclwrite SetAttributeValue or SetAttributeRaw.
func writeScalarValue(body *hclwrite.Body, name string, val interface{}) {
	switch v := val.(type) {
	case core.RawHCLValue:
		writeRawAttribute(body, name, v)
	case string:
		body.SetAttributeValue(name, cty.StringVal(v))
	case bool:
		body.SetAttributeValue(name, cty.BoolVal(v))
	case int64:
		body.SetAttributeValue(name, cty.NumberIntVal(v))
	case float64:
		if v == float64(int64(v)) {
			body.SetAttributeValue(name, cty.NumberIntVal(int64(v)))
		} else {
			body.SetAttributeValue(name, cty.NumberVal(new(big.Float).SetFloat64(v)))
		}
	case []interface{}:
		if len(v) == 0 {
			body.SetAttributeValue(name, cty.EmptyTupleVal)
			return
		}
		vals := make([]cty.Value, 0, len(v))
		for _, item := range v {
			vals = append(vals, cty.StringVal(fmt.Sprintf("%v", item)))
		}
		body.SetAttributeValue(name, cty.TupleVal(vals))
	default:
		// Fallback: write as string representation.
		body.SetAttributeValue(name, cty.StringVal(fmt.Sprintf("%v", val)))
	}
}

// terraformName returns the terraform_name for an attribute, falling back to
// lowercased struct name.
func terraformName(attrDef schema.AttributeDefinition) string {
	if attrDef.TerraformName != "" {
		return attrDef.TerraformName
	}
	return strings.ToLower(attrDef.Name)
}

// resourceLabel derives the sanitized HCL resource label from resource data.
// If the schema defines label_fields, it collects those attribute values in
// order and calls utils.SanitizeMultiKeyResourceName. Otherwise it falls back
// to utils.SanitizeResourceName applied to data.Name, or data.ID as a last resort.
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
