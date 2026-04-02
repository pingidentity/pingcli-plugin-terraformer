package module

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/utils"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Generator handles the generation of Terraform module structure
type Generator struct {
	config ModuleConfig
}

// NewGenerator creates a new module generator with the given configuration
func NewGenerator(config ModuleConfig) *Generator {
	// Apply defaults if not set
	if config.ModuleName == "" {
		config.ModuleName = "ping-export"
	}
	if config.ModuleDirName == "" {
		config.ModuleDirName = "ping-export-module"
	}

	return &Generator{
		config: config,
	}
}

// Generate creates the complete module structure
func (g *Generator) Generate(structure *ModuleStructure) error {
	// Create directory structure
	if err := g.createDirectories(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Generate child module files
	if err := g.generateVersionsTF(); err != nil {
		return fmt.Errorf("failed to generate versions.tf: %w", err)
	}

	if err := g.generateVariablesTF(structure.Variables); err != nil {
		return fmt.Errorf("failed to generate variables.tf: %w", err)
	}

	if err := g.generateOutputsTF(structure.Outputs); err != nil {
		return fmt.Errorf("failed to generate outputs.tf: %w", err)
	}

	if err := g.generateResourceFiles(structure.Resources); err != nil {
		return fmt.Errorf("failed to generate resource files: %w", err)
	}

	// Generate root module files
	if err := g.generateRootVariablesTF(structure.Variables); err != nil {
		return fmt.Errorf("failed to generate root variables.tf: %w", err)
	}

	if err := g.generateModuleTF(structure); err != nil {
		return fmt.Errorf("failed to generate module.tf: %w", err)
	}

	if g.config.IncludeImports {
		if err := g.generateImportsTF(structure.ImportBlocks); err != nil {
			return fmt.Errorf("failed to generate imports.tf: %w", err)
		}
	}

	// Generate tfvars file
	if err := g.generateTFVarsFile(structure); err != nil {
		return fmt.Errorf("failed to generate tfvars: %w", err)
	}

	return nil
}

// createDirectories creates the necessary directory structure
func (g *Generator) createDirectories() error {
	childModulePath := filepath.Join(g.config.OutputDir, g.config.ModuleDirName)
	return os.MkdirAll(childModulePath, 0755)
}

// childModulePath returns the full path to the child module directory
func (g *Generator) childModulePath() string {
	return filepath.Join(g.config.OutputDir, g.config.ModuleDirName)
}

// writeFile writes content to a file in the specified directory
func (g *Generator) writeFile(dir, filename, content string) error {
	filePath := filepath.Join(dir, filename)
	return os.WriteFile(filePath, []byte(content), 0644)
}

// generateVersionsTF creates the versions.tf file in the child module
func (g *Generator) generateVersionsTF() error {
	content := `terraform {
  required_version = ">= 1.5"

  required_providers {
    pingone = {
      source  = "pingidentity/pingone"
      version = "1.18.0"
    }
  }
}
`
	return g.writeFile(g.childModulePath(), "versions.tf", content)
}

// generateVariablesTF creates the variables.tf file in the child module
func (g *Generator) generateVariablesTF(variables []Variable) error {
	var sb strings.Builder

	// Always include the core environment_id variable that child module resources use
	sb.WriteString(`variable "pingone_environment_id" {
  type        = string
  description = "The PingOne environment ID to configure DaVinci resources in"

  validation {
    condition     = can(regex("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$", var.pingone_environment_id))
    error_message = "The PingOne Environment ID must be a valid PingOne resource ID (UUID format)."
  }
}

`)

	// Group variables by resource type for better organization
	groupedVars := g.groupVariablesByResourceType(variables)

	// Generate variables in sorted order by resource type
	order := sortedKeys(groupedVars)
	for _, resourceType := range order {
		vars := groupedVars[resourceType]

		// Section header
		hdr := cases.Title(language.English).String(resourceType)
		sb.WriteString(fmt.Sprintf("# %s Variables\n\n", hdr))

		// Sort variables alphabetically by name for deterministic output
		sort.Slice(vars, func(i, j int) bool {
			return strings.ToLower(vars[i].Name) < strings.ToLower(vars[j].Name)
		})
		for _, v := range vars {
			sb.WriteString(g.generateVariableBlock(v))
		}
	}

	return g.writeFile(g.childModulePath(), "variables.tf", sb.String())
}

// generateRootVariablesTF creates the variables.tf file in the root module
// This mirrors the child module variables for use in module invocation
func (g *Generator) generateRootVariablesTF(variables []Variable) error {
	var sb strings.Builder

	// Always include the core environment_id variable
	sb.WriteString(`variable "pingone_environment_id" {
  type        = string
  description = "The PingOne environment ID to configure DaVinci resources in"

  validation {
    condition     = can(regex("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$", var.pingone_environment_id))
    error_message = "The PingOne Environment ID must be a valid PingOne resource ID (UUID format)."
  }
}

`)

	// Group variables by resource type for better organization
	groupedVars := g.groupVariablesByResourceType(variables)

	// Generate variables in sorted order by resource type
	order := sortedKeys(groupedVars)
	for _, resourceType := range order {
		vars := groupedVars[resourceType]

		// Section header
		hdr := cases.Title(language.English).String(resourceType)
		sb.WriteString(fmt.Sprintf("# %s Variables\n\n", hdr))

		// Sort variables alphabetically by name for deterministic output
		sort.Slice(vars, func(i, j int) bool {
			return strings.ToLower(vars[i].Name) < strings.ToLower(vars[j].Name)
		})
		for _, v := range vars {
			sb.WriteString(g.generateRootVariableBlock(v))
			sb.WriteString("\n")
		}
	}

	// Root variables file is prefixed by module name
	return g.writeFile(g.config.OutputDir, fmt.Sprintf("%s-variables.tf", g.config.ModuleName), sb.String())
}

// generateRootVariableBlock generates a single variable block for the root module
// Root variables do not have default values - those come from tfvars
func (g *Generator) generateRootVariableBlock(v Variable) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("variable \"%s\" {\n", v.Name))
	sb.WriteString(fmt.Sprintf("  type        = %s\n", v.Type))
	sb.WriteString(fmt.Sprintf("  description = %q\n", v.Description))

	if v.Sensitive {
		sb.WriteString("  sensitive   = true\n")
	}

	sb.WriteString("}\n")

	return sb.String()
}

// groupVariablesByResourceType groups variables by their resource type
func (g *Generator) groupVariablesByResourceType(variables []Variable) map[string][]Variable {
	grouped := make(map[string][]Variable)
	for _, v := range variables {
		grouped[v.ResourceType] = append(grouped[v.ResourceType], v)
	}
	return grouped
}

// sortedKeys returns the keys of the given map in sorted order.
func sortedKeys(m map[string][]Variable) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// generateVariableBlock generates a single variable block
func (g *Generator) generateVariableBlock(v Variable) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("variable \"%s\" {\n", v.Name))
	sb.WriteString(fmt.Sprintf("  type        = %s\n", v.Type))
	sb.WriteString(fmt.Sprintf("  description = %q\n", v.Description))

	// Do not include default values in child module variables.tf to avoid leaking secrets.
	// Actual values must be provided via ping-export-terraform.auto.tfvars.

	if v.Sensitive {
		sb.WriteString("  sensitive   = true\n")
	}

	if v.Validation != nil {
		sb.WriteString("\n  validation {\n")
		sb.WriteString(fmt.Sprintf("    condition     = %s\n", v.Validation.Condition))
		sb.WriteString(fmt.Sprintf("    error_message = %q\n", v.Validation.ErrorMessage))
		sb.WriteString("  }\n")
	}

	sb.WriteString("}\n")

	return sb.String()
}

// formatDefaultValue formats a default value for use in a .tfvars file.
// Dispatches on the runtime type of value rather than the declared variable
// type, since connector properties may carry bool/list/map values even when
// the variable schema declares them as "string".
//
// Complex types (slices, maps) are JSON-serialized and quoted so they
// remain compatible with the variable's declared type (typically "string").
func (g *Generator) formatDefaultValue(value interface{}, varType string) string {
	if value == nil {
		return "null"
	}

	switch v := value.(type) {
	case string:
		// Escape HCL template sequences so that literal "${" and "%{"
		// are not interpreted as interpolation/directives in .tfvars.
		escaped := strings.ReplaceAll(v, "${", "$${")
		escaped = strings.ReplaceAll(escaped, "%{", "%%{")
		return fmt.Sprintf("%q", escaped)
	case bool:
		if varType == "string" {
			// Variable declared as string — quote the bool.
			if v {
				return `"true"`
			}
			return `"false"`
		}
		if v {
			return "true"
		}
		return "false"
	case float64:
		if varType == "string" {
			if v == float64(int64(v)) {
				return fmt.Sprintf("%q", fmt.Sprintf("%d", int64(v)))
			}
			return fmt.Sprintf("%q", fmt.Sprintf("%g", v))
		}
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%g", v)
	case float32:
		if varType == "string" {
			return fmt.Sprintf("%q", fmt.Sprintf("%g", v))
		}
		return fmt.Sprintf("%g", v)
	case int:
		if varType == "string" {
			return fmt.Sprintf("%q", fmt.Sprintf("%d", v))
		}
		return fmt.Sprintf("%d", v)
	case int64:
		if varType == "string" {
			return fmt.Sprintf("%q", fmt.Sprintf("%d", v))
		}
		return fmt.Sprintf("%d", v)
	case []interface{}, map[string]interface{}:
		b, err := json.Marshal(v)
		if err != nil {
			return "null"
		}
		// Complex values are always quoted as JSON strings so they
		// match the variable's declared type (typically "string").
		escaped := strings.ReplaceAll(string(b), "${", "$${")
		escaped = strings.ReplaceAll(escaped, "%{", "%%{")
		return fmt.Sprintf("%q", escaped)
	default:
		// Fall back to quoted string representation for unknown types.
		escaped := strings.ReplaceAll(fmt.Sprintf("%v", v), "${", "$${")
		escaped = strings.ReplaceAll(escaped, "%{", "%%{")
		return fmt.Sprintf("%q", escaped)
	}
}

// generateOutputsTF creates the outputs.tf file in the child module
func (g *Generator) generateOutputsTF(outputs []Output) error {
	var sb strings.Builder

	for _, o := range outputs {
		sb.WriteString(fmt.Sprintf("output \"%s\" {\n", o.Name))
		sb.WriteString(fmt.Sprintf("  description = %q\n", o.Description))
		sb.WriteString(fmt.Sprintf("  value       = %s\n", o.Value))

		if o.Sensitive {
			sb.WriteString("  sensitive   = true\n")
		}

		sb.WriteString("}\n\n")
	}

	return g.writeFile(g.childModulePath(), "outputs.tf", sb.String())
}

// resourceFileExtension returns the file extension for resource content files.
func (g *Generator) resourceFileExtension() string {
	if g.config.OutputFormat == "tfjson" {
		return ".tf.json"
	}
	return ".tf"
}

// generateResourceFiles creates the resource files in the child module
func (g *Generator) generateResourceFiles(resources ModuleResources) error {
	// Get sorted keys for deterministic output.
	keys := make([]string, 0, len(resources))
	for k := range resources {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, resourceType := range keys {
		content := resources[resourceType]
		if content == "" {
			continue
		}

		// Only sort HCL content; tfjson is already structured JSON.
		if g.config.OutputFormat != "tfjson" {
			content = utils.SortAllResourceBlocks(content)
		}

		filename := resourceType + g.resourceFileExtension()
		if err := g.writeFile(g.childModulePath(), filename, content); err != nil {
			return err
		}
	}

	return nil
}

// generateModuleTF creates the module.tf file in the root module
func (g *Generator) generateModuleTF(structure *ModuleStructure) error {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("module \"%s\" {\n", g.config.ModuleName))
	sb.WriteString(fmt.Sprintf("  source = \"./%s\"\n\n", g.config.ModuleDirName))

	// Core environment ID - always use variable reference
	sb.WriteString("  pingone_environment_id = var.pingone_environment_id\n\n")

	// Group variables by resource type
	groupedVars := g.groupVariablesByResourceType(structure.Variables)

	// Generate variable inputs
	order := sortedKeys(groupedVars)
	for _, resourceType := range order {
		vars := groupedVars[resourceType]

		hdr := cases.Title(language.English).String(resourceType)
		sb.WriteString(fmt.Sprintf("  # %s Variables\n", hdr))

		for _, v := range vars {
			sb.WriteString(g.generateModuleInput(v))
		}

		sb.WriteString("\n")
	}

	sb.WriteString("}\n")

	// Root file name is prefixed by module name
	return g.writeFile(g.config.OutputDir, fmt.Sprintf("%s-module.tf", g.config.ModuleName), sb.String())
}

// generateModuleInput generates a single module input line
// Always uses variable references (var.{name}) - values come from tfvars
func (g *Generator) generateModuleInput(v Variable) string {
	return fmt.Sprintf("  %s = var.%s\n", v.Name, v.Name)
}

// generateImportsTF creates the imports.tf file in the root module
func (g *Generator) generateImportsTF(importBlocks []ImportBlock) error {
	if g.config.OutputFormat == "tfjson" {
		return g.generateImportsTFJSON(importBlocks)
	}
	return g.generateImportsHCL(importBlocks)
}

// generateImportsHCL creates HCL-format import blocks.
func (g *Generator) generateImportsHCL(importBlocks []ImportBlock) error {
	var comments strings.Builder
	var blocks strings.Builder

	// First, emit all commented terraform import commands together
	for _, ib := range importBlocks {
		comments.WriteString(fmt.Sprintf("# terraform import %s %q\n", ib.To, ib.ID))
	}

	// Then emit actual import blocks
	for _, ib := range importBlocks {
		blocks.WriteString("import {\n")
		blocks.WriteString(fmt.Sprintf("  to = %s\n", ib.To))
		blocks.WriteString(fmt.Sprintf("  id = %q\n", ib.ID))
		blocks.WriteString("}\n\n")
	}

	// Combine: comments at top, a blank line, then blocks
	final := comments.String()
	if blocks.Len() > 0 {
		final += "\n" + blocks.String()
	}

	// Root file name is prefixed by module name
	return g.writeFile(g.config.OutputDir, fmt.Sprintf("%s-imports.tf", g.config.ModuleName), final)
}

// generateImportsTFJSON creates JSON-format import blocks.
func (g *Generator) generateImportsTFJSON(importBlocks []ImportBlock) error {
	type jsonImport struct {
		To string `json:"to"`
		ID string `json:"id"`
	}

	wrapper := struct {
		Import []jsonImport `json:"import"`
	}{
		Import: make([]jsonImport, 0, len(importBlocks)),
	}

	for _, ib := range importBlocks {
		wrapper.Import = append(wrapper.Import, jsonImport{
			To: ib.To,
			ID: ib.ID,
		})
	}

	data, err := json.MarshalIndent(wrapper, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal import blocks to JSON: %w", err)
	}

	// Root file name is prefixed by module name
	return g.writeFile(g.config.OutputDir, fmt.Sprintf("%s-imports.tf.json", g.config.ModuleName), string(data)+"\n")
}

// generateTFVarsFile creates the ping-export-terraform.auto.tfvars file
// When IncludeValues is false, creates a template with empty values
// When IncludeValues is true, populates with actual values from variables
func (g *Generator) generateTFVarsFile(structure *ModuleStructure) error {
	var sb strings.Builder

	// Add file header comment
	sb.WriteString("# Terraform variable values for DaVinci export\n")
	sb.WriteString("# Generated by pingcli tf export\n\n")

	// Environment ID
	if g.config.IncludeValues {
		sb.WriteString(fmt.Sprintf("pingone_environment_id = %q\n\n", g.config.EnvironmentID))
	} else {
		sb.WriteString("pingone_environment_id = \"\"\n\n")
	}

	// Group variables by resource type
	groupedVars := g.groupVariablesByResourceType(structure.Variables)

	// Generate variable values
	order := sortedKeys(groupedVars)
	for _, resourceType := range order {
		vars := groupedVars[resourceType]

		hdr := cases.Title(language.English).String(resourceType)
		sb.WriteString(fmt.Sprintf("# %s Variables\n\n", hdr))

		// Sort variables alphabetically within each resource type group
		sort.Slice(vars, func(i, j int) bool {
			return strings.ToLower(vars[i].Name) < strings.ToLower(vars[j].Name)
		})
		for _, v := range vars {
			sb.WriteString(g.generateTFVarValue(v))
		}

		sb.WriteString("\n")
	}

	// Root tfvars file is prefixed by module name
	return g.writeFile(g.config.OutputDir, fmt.Sprintf("%s-terraform.auto.tfvars", g.config.ModuleName), sb.String())
}

// generateTFVarValue generates a single tfvar value line
func (g *Generator) generateTFVarValue(v Variable) string {
	// Secrets always get empty values regardless of IncludeValues.
	// Only annotate the comment when IncludeValues is true, since in that
	// mode the empty value is the exception and needs explanation.
	if v.IsSecret {
		if g.config.IncludeValues {
			return fmt.Sprintf("%s = \"\"  # Secret value - provide manually\n", v.Name)
		}
		return fmt.Sprintf("%s = \"\"\n", v.Name)
	}

	// If IncludeValues is true and we have a default, use it
	if g.config.IncludeValues && v.Default != nil {
		return fmt.Sprintf("%s = %s\n", v.Name, g.formatDefaultValue(v.Default, v.Type))
	}

	// Otherwise, use empty/zero values based on type
	switch v.Type {
	case "string":
		return fmt.Sprintf("%s = \"\"\n", v.Name)
	case "number":
		return fmt.Sprintf("%s = 0\n", v.Name)
	case "bool":
		return fmt.Sprintf("%s = false\n", v.Name)
	default:
		return fmt.Sprintf("%s = null\n", v.Name)
	}
}
