// Package imports generates Terraform import blocks from schema definitions using hclwrite.
package imports

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/core"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/utils"
)

// placeholderRe matches {placeholder} tokens in import ID format strings.
var placeholderRe = regexp.MustCompile(`\{([^}]+)\}`)

// Generator generates Terraform import blocks from schema definitions.
type Generator struct{}

// NewGenerator creates a new import block generator.
func NewGenerator() *Generator {
	return &Generator{}
}

// GenerateImportBlock generates a single Terraform 1.5+ import block for a resource.
//
// The import ID is built by expanding placeholders in def.Dependencies.ImportIDFormat:
//   - {env_id}         → environmentID parameter
//   - {resource_id}    → data.ID
//   - {<attr_name>}    → data.Attributes[<attr_name>] (string)
//
// This replaces the resource-type switch statement in internal/importgen with a
// schema-driven approach: any new resource only needs an import_id_format in its
// YAML definition.
func (g *Generator) GenerateImportBlock(data *core.ResourceData, def *schema.ResourceDefinition, environmentID string) (string, error) {
	if data == nil {
		return "", fmt.Errorf("resource data is nil")
	}
	if def == nil {
		return "", fmt.Errorf("resource definition is nil")
	}
	if def.Dependencies.ImportIDFormat == "" {
		return "", fmt.Errorf("resource definition has no import_id_format")
	}

	label := resourceLabel(data, def)
	if label == "" {
		return "", fmt.Errorf("resource has neither Name nor ID set")
	}

	importID, err := expandImportID(def.Dependencies.ImportIDFormat, data, environmentID)
	if err != nil {
		return "", fmt.Errorf("expand import ID: %w", err)
	}

	file := hclwrite.NewEmptyFile()
	block := file.Body().AppendNewBlock("import", nil)
	body := block.Body()

	// to = resource_type.label (unquoted reference)
	tokens := hclwrite.Tokens{
		{Type: hclsyntax.TokenIdent, Bytes: []byte(def.Metadata.ResourceType + "." + label)},
	}
	body.SetAttributeRaw("to", tokens)
	body.SetAttributeValue("id", cty.StringVal(importID))

	return string(hclwrite.Format(file.Bytes())), nil
}

// GenerateImportBlocks generates import blocks for a list of resources,
// sorted deterministically by resource name.
func (g *Generator) GenerateImportBlocks(dataList []*core.ResourceData, def *schema.ResourceDefinition, environmentID string) (string, error) {
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
		block, err := g.GenerateImportBlock(data, def, environmentID)
		if err != nil {
			return "", fmt.Errorf("resource %s: %w", data.Name, err)
		}
		sb.WriteString(block)
	}
	return sb.String(), nil
}

// expandImportID replaces placeholder tokens in the format string.
func expandImportID(format string, data *core.ResourceData, environmentID string) (string, error) {
	var missingFields []string

	result := placeholderRe.ReplaceAllStringFunc(format, func(match string) string {
		// Strip braces: {env_id} → env_id
		key := match[1 : len(match)-1]

		switch key {
		case "env_id":
			if environmentID == "" {
				missingFields = append(missingFields, key)
				return match
			}
			return environmentID
		case "resource_id":
			if data.ID == "" {
				missingFields = append(missingFields, key)
				return match
			}
			return data.ID
		default:
			// Look up in resource attributes.
			if v, ok := data.Attributes[key]; ok && v != nil {
				if s, ok := v.(string); ok && s != "" {
					return s
				}
				// Handle ResolvedReference — use the original UUID for import IDs.
				if ref, ok := v.(core.ResolvedReference); ok && ref.OriginalValue != "" {
					return ref.OriginalValue
				}
			}
			missingFields = append(missingFields, key)
			return match
		}
	})

	if len(missingFields) > 0 {
		return "", fmt.Errorf("unresolved placeholders in import ID format %q: %s", format, strings.Join(missingFields, ", "))
	}
	return result, nil
}

// resourceLabel derives the sanitized HCL resource label from resource data.
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
