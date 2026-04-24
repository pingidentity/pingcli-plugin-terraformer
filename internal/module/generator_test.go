package module

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratorCreateDirectories(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create generator
	config := ModuleConfig{
		OutputDir:      tmpDir,
		ModuleDirName:  "test-module",
		IncludeImports: false,
		IncludeValues:  false,
		EnvironmentID:  "test-env-id",
	}
	generator := NewGenerator(config)

	// Generate directories
	err := generator.createDirectories()
	require.NoError(t, err)

	// Verify child module directory exists
	childModulePath := filepath.Join(tmpDir, "test-module")
	info, err := os.Stat(childModulePath)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestGeneratorVersionsTF(t *testing.T) {
	tmpDir := t.TempDir()

	config := ModuleConfig{
		OutputDir:     tmpDir,
		ModuleDirName: "test-module",
	}
	generator := NewGenerator(config)

	// Create directories first
	err := generator.createDirectories()
	require.NoError(t, err)

	// Generate versions.tf
	err = generator.generateVersionsTF()
	require.NoError(t, err)

	// Verify file was created
	versionsPath := filepath.Join(tmpDir, "test-module", "versions.tf")
	content, err := os.ReadFile(versionsPath)
	require.NoError(t, err)

	// Verify content
	assert.Contains(t, string(content), "terraform {")
	assert.Contains(t, string(content), "required_version")
	assert.Contains(t, string(content), "pingone")
	assert.Contains(t, string(content), "pingidentity/pingone")
}

func TestGeneratorVariablesTF(t *testing.T) {
	tmpDir := t.TempDir()

	config := ModuleConfig{
		OutputDir:     tmpDir,
		ModuleDirName: "test-module",
	}
	generator := NewGenerator(config)

	// Create directories
	err := generator.createDirectories()
	require.NoError(t, err)

	// Generate variables.tf with some test variables
	variables := []Variable{
		{
			Name:         "test_flow_name",
			Type:         "string",
			Description:  "Name of the test flow",
			Default:      "Test Flow",
			ResourceType: "flow",
			ResourceName: "test_flow",
		},
		{
			Name:         "test_var_value",
			Type:         "string",
			Description:  "Value of test variable",
			Default:      "test value",
			Sensitive:    true,
			ResourceType: "variable",
			ResourceName: "test_var",
		},
	}

	err = generator.generateVariablesTF(variables)
	require.NoError(t, err)

	// Verify file was created
	variablesPath := filepath.Join(tmpDir, "test-module", "variables.tf")
	content, err := os.ReadFile(variablesPath)
	require.NoError(t, err)

	// Verify content
	assert.Contains(t, string(content), "variable \"pingone_environment_id\"")
	assert.Contains(t, string(content), "variable \"test_flow_name\"")
	assert.Contains(t, string(content), "variable \"test_var_value\"")
	assert.Contains(t, string(content), "sensitive   = true")
	assert.Contains(t, string(content), "can(regex(")

	// Bug 10: Ensure no default values are emitted in child variables.tf
	assert.NotContains(t, string(content), "default     = ")
}

// TestGeneratorVariablesTF_NoDefaults ensures defaults are not present even if provided
func TestGeneratorVariablesTF_NoDefaults(t *testing.T) {
	tmpDir := t.TempDir()

	config := ModuleConfig{
		OutputDir:     tmpDir,
		ModuleDirName: "test-module",
	}
	generator := NewGenerator(config)

	// Create directories
	err := generator.createDirectories()
	require.NoError(t, err)

	variables := []Variable{
		{
			Name:         "with_default_string",
			Type:         "string",
			Description:  "String with default",
			Default:      "secret",
			ResourceType: "variable",
			ResourceName: "var1",
		},
		{
			Name:         "with_default_number",
			Type:         "number",
			Description:  "Number with default",
			Default:      42,
			ResourceType: "variable",
			ResourceName: "var2",
		},
	}

	err = generator.generateVariablesTF(variables)
	require.NoError(t, err)

	variablesPath := filepath.Join(tmpDir, "test-module", "variables.tf")
	content, err := os.ReadFile(variablesPath)
	require.NoError(t, err)
	contentStr := string(content)

	// Defaults must not appear in child module variables.tf
	assert.NotContains(t, contentStr, "default     = ")
	// Variables and descriptions should still be present
	assert.Contains(t, contentStr, "variable \"with_default_string\"")
	assert.Contains(t, contentStr, "String with default")
	assert.Contains(t, contentStr, "variable \"with_default_number\"")
	assert.Contains(t, contentStr, "Number with default")
}

func TestGeneratorOutputsTF(t *testing.T) {
	tmpDir := t.TempDir()

	config := ModuleConfig{
		OutputDir:     tmpDir,
		ModuleDirName: "test-module",
	}
	generator := NewGenerator(config)

	// Create directories
	err := generator.createDirectories()
	require.NoError(t, err)

	// Generate outputs.tf with test outputs
	outputs := []Output{
		{
			Name:        "flow_id",
			Description: "The ID of the flow",
			Value:       "pingone_davinci_flow.test.id",
		},
		{
			Name:        "secret_value",
			Description: "A secret value",
			Value:       "pingone_davinci_variable.secret.value",
			Sensitive:   true,
		},
	}

	err = generator.generateOutputsTF(outputs)
	require.NoError(t, err)

	// Verify file was created
	outputsPath := filepath.Join(tmpDir, "test-module", "outputs.tf")
	content, err := os.ReadFile(outputsPath)
	require.NoError(t, err)

	// Verify content
	assert.Contains(t, string(content), "output \"flow_id\"")
	assert.Contains(t, string(content), "output \"secret_value\"")
	assert.Contains(t, string(content), "pingone_davinci_flow.test.id")
	assert.Contains(t, string(content), "sensitive   = true")
}

func TestGeneratorModuleTF(t *testing.T) {
	tmpDir := t.TempDir()

	// Test verifies module.tf always uses variable references
	// regardless of IncludeValues flag (values come from tfvars)
	tests := []struct {
		name          string
		includeValues bool
	}{
		{
			name:          "Without values",
			includeValues: false,
		},
		{
			name:          "With values",
			includeValues: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ModuleConfig{
				OutputDir:     tmpDir,
				ModuleDirName: "test-module",
				IncludeValues: tt.includeValues,
				EnvironmentID: "test-env-123",
			}
			generator := NewGenerator(config)

			// Create test structure
			structure := &ModuleStructure{
				Config: config,
				Variables: []Variable{
					{
						Name:         "test_var",
						Type:         "string",
						Description:  "Test variable",
						Default:      "default value",
						ResourceType: "flow",
					},
				},
			}

			// Generate module.tf
			err := generator.generateModuleTF(structure)
			require.NoError(t, err)

			// Verify file was created (prefixed by default ModuleName)
			modulePath := filepath.Join(tmpDir, "ping-export-module.tf")
			content, err := os.ReadFile(modulePath)
			require.NoError(t, err)
			contentStr := string(content)

			// Verify content - always uses variable references
			assert.Contains(t, contentStr, "module \"ping-export\"") // Default module name
			assert.Contains(t, contentStr, "source = \"./test-module\"")
			assert.Contains(t, contentStr, "pingone_environment_id = var.pingone_environment_id")
			assert.Contains(t, contentStr, "test_var = var.test_var")
		})
	}
}

func TestGeneratorResourceFiles(t *testing.T) {
	tmpDir := t.TempDir()

	config := ModuleConfig{
		OutputDir:     tmpDir,
		ModuleDirName: "test-module",
	}
	generator := NewGenerator(config)

	// Create directories
	err := generator.createDirectories()
	require.NoError(t, err)

	// Generate resource files
	resources := ModuleResources{
		"pingone_davinci_flow":               "resource \"pingone_davinci_flow\" \"test\" {}",
		"pingone_davinci_connector_instance": "resource \"pingone_davinci_connector_instance\" \"http\" {}",
		"pingone_davinci_variable":           "resource \"pingone_davinci_variable\" \"company_name\" {}",
		"pingone_davinci_application":        "resource \"pingone_davinci_application\" \"app\" {}",
	}

	err = generator.generateResourceFiles(resources)
	require.NoError(t, err)

	// Verify files were created
	childModulePath := filepath.Join(tmpDir, "test-module")

	flowsPath := filepath.Join(childModulePath, "pingone_davinci_flow.tf")
	content, err := os.ReadFile(flowsPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "pingone_davinci_flow")

	connectionsPath := filepath.Join(childModulePath, "pingone_davinci_connector_instance.tf")
	content, err = os.ReadFile(connectionsPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "pingone_davinci_connector_instance")

	variablesPath := filepath.Join(childModulePath, "pingone_davinci_variable.tf")
	content, err = os.ReadFile(variablesPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "pingone_davinci_variable")
}

func TestGeneratorImportsTF(t *testing.T) {
	tmpDir := t.TempDir()

	config := ModuleConfig{
		OutputDir:      tmpDir,
		ModuleDirName:  "test-module",
		IncludeImports: true,
	}
	generator := NewGenerator(config)

	// Generate imports.tf
	importBlocks := []ImportBlock{
		{
			To: "module.davinci.pingone_davinci_flow.test",
			ID: "env-id:flow-id",
		},
		{
			To: "module.davinci.pingone_davinci_variable.company_name",
			ID: "env-id:var-id",
		},
	}

	err := generator.generateImportsTF(importBlocks)
	require.NoError(t, err)

	// Verify file was created
	importsPath := filepath.Join(tmpDir, "ping-export-imports.tf")
	content, err := os.ReadFile(importsPath)
	require.NoError(t, err)

	// Verify content
	assert.Contains(t, string(content), "import {")
	assert.Contains(t, string(content), "module.davinci.pingone_davinci_flow.test")
	assert.Contains(t, string(content), "env-id:flow-id")
	assert.Contains(t, string(content), "module.davinci.pingone_davinci_variable.company_name")
}

func TestFullModuleGeneration(t *testing.T) {
	tmpDir := t.TempDir()

	config := ModuleConfig{
		OutputDir:      tmpDir,
		ModuleDirName:  "davinci-module",
		IncludeImports: true,
		IncludeValues:  true,
		EnvironmentID:  "test-env-123",
	}
	generator := NewGenerator(config)

	// Create full module structure
	structure := &ModuleStructure{
		Config: config,
		Variables: []Variable{
			{
				Name:         "davinci_flow_main_name",
				Type:         "string",
				Description:  "Name of main flow",
				Default:      "Main Flow",
				ResourceType: "flow",
			},
		},
		Outputs: []Output{
			{
				Name:        "main_flow_id",
				Description: "ID of main flow",
				Value:       "pingone_davinci_flow.main.id",
			},
		},
		Resources: ModuleResources{
			"pingone_davinci_flow":               "resource \"pingone_davinci_flow\" \"main\" {\n  environment_id = var.pingone_environment_id\n  name = var.davinci_flow_main_name\n}",
			"pingone_davinci_connector_instance": "resource \"pingone_davinci_connector_instance\" \"http\" {}",
		},
		ImportBlocks: []ImportBlock{
			{
				To: "module.davinci.pingone_davinci_flow.main",
				ID: "test-env-123:flow-123",
			},
		},
	}

	// Generate
	err := generator.Generate(structure)
	require.NoError(t, err)

	// Verify all files exist
	childModulePath := filepath.Join(tmpDir, "davinci-module")

	// Check child module files
	assert.FileExists(t, filepath.Join(childModulePath, "versions.tf"))
	assert.FileExists(t, filepath.Join(childModulePath, "variables.tf"))
	assert.FileExists(t, filepath.Join(childModulePath, "outputs.tf"))
	assert.FileExists(t, filepath.Join(childModulePath, "pingone_davinci_flow.tf"))
	assert.FileExists(t, filepath.Join(childModulePath, "pingone_davinci_connector_instance.tf"))

	// Check root module files
	assert.FileExists(t, filepath.Join(tmpDir, "ping-export-module.tf"))
	assert.FileExists(t, filepath.Join(tmpDir, "ping-export-imports.tf"))
}

// TestGenerator_GenerateRootVariablesTF verifies that root module variables.tf is generated correctly
func TestGenerator_GenerateRootVariablesTF(t *testing.T) {
	tmpDir := t.TempDir()

	config := ModuleConfig{
		OutputDir:     tmpDir,
		ModuleDirName: "davinci-module",
	}
	generator := NewGenerator(config)

	// Create test variables
	variables := []Variable{
		{
			Name:         "davinci_variable_company_name_value",
			Type:         "string",
			Description:  "Value for DaVinci variable: CompanyName",
			Sensitive:    false,
			IsSecret:     false,
			ResourceType: "variable",
			ResourceName: "company_name",
		},
		{
			Name:         "davinci_variable_secret_key_value",
			Type:         "string",
			Description:  "Value for DaVinci variable: SecretKey",
			Sensitive:    true,
			IsSecret:     true,
			ResourceType: "variable",
			ResourceName: "secret_key",
		},
		{
			Name:         "davinci_connection_http_base_url",
			Type:         "string",
			Description:  "Base URL for HTTP connection",
			Sensitive:    false,
			IsSecret:     false,
			ResourceType: "connection",
			ResourceName: "http_connector",
		},
	}

	// Generate root variables.tf
	err := generator.generateRootVariablesTF(variables)
	require.NoError(t, err)

	// Verify file exists
	rootVariablesPath := filepath.Join(tmpDir, "ping-export-variables.tf")
	require.FileExists(t, rootVariablesPath)

	// Read and verify content
	content, err := os.ReadFile(rootVariablesPath)
	require.NoError(t, err)
	contentStr := string(content)

	// Verify environment_id variable is present
	assert.Contains(t, contentStr, `variable "pingone_environment_id"`)
	assert.Contains(t, contentStr, "PingOne environment ID")

	// Verify davinci variable
	assert.Contains(t, contentStr, `variable "davinci_variable_company_name_value"`)
	assert.Contains(t, contentStr, "Value for DaVinci variable: CompanyName")

	// Verify secret variable
	assert.Contains(t, contentStr, `variable "davinci_variable_secret_key_value"`)
	assert.Contains(t, contentStr, "Value for DaVinci variable: SecretKey")
	assert.Contains(t, contentStr, "sensitive   = true")

	// Verify connection variable
	assert.Contains(t, contentStr, `variable "davinci_connection_http_base_url"`)
	assert.Contains(t, contentStr, "Base URL for HTTP connection")

	// Verify grouping comments present
	assert.Contains(t, contentStr, "# Variable Variables")
	assert.Contains(t, contentStr, "# Connection Variables")
}

// TestGenerator_VariableNaming_ChildModule verifies child module uses correct variable name
// TestGenerator_GenerateModuleTF_UsesVariableReferences verifies that module.tf uses variable references
func TestGenerator_GenerateModuleTF_UsesVariableReferences(t *testing.T) {
	tests := []struct {
		name          string
		includeValues bool
	}{
		{
			name:          "without include-values flag",
			includeValues: false,
		},
		{
			name:          "with include-values flag",
			includeValues: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			config := ModuleConfig{
				OutputDir:     tmpDir,
				ModuleDirName: "davinci-module",
				IncludeValues: tt.includeValues,
				EnvironmentID: "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			}
			generator := NewGenerator(config)

			// Create test module structure
			structure := &ModuleStructure{
				Config: config,
				Variables: []Variable{
					{
						Name:         "davinci_variable_company_name_value",
						Type:         "string",
						Description:  "Value for DaVinci variable: CompanyName",
						Default:      "ACME Corporation",
						ResourceType: "variable",
						ResourceName: "company_name",
					},
					{
						Name:         "davinci_variable_secret_key_value",
						Type:         "string",
						Description:  "Value for DaVinci variable: SecretKey",
						Sensitive:    true,
						IsSecret:     true,
						ResourceType: "variable",
						ResourceName: "secret_key",
					},
				},
			}

			// Generate module.tf
			err := generator.generateModuleTF(structure)
			require.NoError(t, err)

			// Read and verify content
			moduleTFPath := filepath.Join(tmpDir, "ping-export-module.tf")
			require.FileExists(t, moduleTFPath)
			content, err := os.ReadFile(moduleTFPath)
			require.NoError(t, err)
			contentStr := string(content)

			// Verify module block
			assert.Contains(t, contentStr, `module "ping-export" {`) // Default module name
			assert.Contains(t, contentStr, `source = "./davinci-module"`)

			// Verify environment_id uses variable reference
			assert.Contains(t, contentStr, "pingone_environment_id = var.pingone_environment_id")
			// Should NOT contain hardcoded environment ID
			assert.NotContains(t, contentStr, `pingone_environment_id = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"`)

			// Verify variables use variable references
			assert.Contains(t, contentStr, "davinci_variable_company_name_value = var.davinci_variable_company_name_value")
			assert.Contains(t, contentStr, "davinci_variable_secret_key_value = var.davinci_variable_secret_key_value")

			// Should NOT contain empty strings or hardcoded values
			assert.NotContains(t, contentStr, `davinci_variable_company_name_value = ""`)
			assert.NotContains(t, contentStr, `davinci_variable_company_name_value = "ACME Corporation"`)
			assert.NotContains(t, contentStr, `davinci_variable_secret_key_value = ""`)
		})
	}
}

// TestGenerator_GenerateTFVarsTemplate_WithoutValues verifies tfvars template generation without values
func TestGenerator_GenerateTFVarsTemplate_WithoutValues(t *testing.T) {
	tmpDir := t.TempDir()

	config := ModuleConfig{
		OutputDir:     tmpDir,
		ModuleDirName: "davinci-module",
		IncludeValues: false,
		EnvironmentID: "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
	}
	generator := NewGenerator(config)

	// Create test module structure with variables
	structure := &ModuleStructure{
		Config: config,
		Variables: []Variable{
			{
				Name:         "davinci_variable_company_name_value",
				Type:         "string",
				Description:  "Value for DaVinci variable: CompanyName",
				Default:      nil, // No default when values not included
				ResourceType: "variable",
				ResourceName: "company_name",
			},
			{
				Name:         "davinci_variable_secret_key_value",
				Type:         "string",
				Description:  "Value for DaVinci variable: SecretKey",
				Sensitive:    true,
				IsSecret:     true,
				ResourceType: "variable",
				ResourceName: "secret_key",
			},
			{
				Name:         "davinci_variable_port_value",
				Type:         "number",
				Description:  "Port number",
				ResourceType: "variable",
				ResourceName: "port",
			},
			{
				Name:         "davinci_variable_enabled_value",
				Type:         "bool",
				Description:  "Feature enabled flag",
				ResourceType: "variable",
				ResourceName: "enabled",
			},
		},
	}

	// Generate tfvars template
	err := generator.generateTFVarsFile(structure)
	require.NoError(t, err)

	// Verify file exists
	tfvarsPath := filepath.Join(tmpDir, "ping-export-terraform.auto.tfvars")
	require.FileExists(t, tfvarsPath)

	// Read and verify content
	content, err := os.ReadFile(tfvarsPath)
	require.NoError(t, err)
	contentStr := string(content)

	// Verify environment_id with empty value (no TODO comment when IncludeValues=false)
	assert.Contains(t, contentStr, `pingone_environment_id = ""`)
	assert.NotContains(t, contentStr, "# TODO: Provide PingOne environment ID")

	// Verify string variable with empty value
	assert.Contains(t, contentStr, `davinci_variable_company_name_value = ""`)

	// Verify secret variable with empty value (no comment when IncludeValues=false)
	assert.Contains(t, contentStr, `davinci_variable_secret_key_value = ""`)
	assert.NotContains(t, contentStr, "# Secret value")

	// Verify number variable with zero
	assert.Contains(t, contentStr, `davinci_variable_port_value = 0`)

	// Verify bool variable with false
	assert.Contains(t, contentStr, `davinci_variable_enabled_value = false`)

	// Verify grouping comments
	assert.Contains(t, contentStr, "# Variable Variables")

	// Verify variables within the group are sorted alphabetically
	varPos := strings.Index(contentStr, "# Variable Variables")
	if varPos >= 0 {
		section := contentStr[varPos:]
		aIdx := strings.Index(section, "davinci_variable_company_name_value")
		bIdx := strings.Index(section, "davinci_variable_enabled_value")
		cIdx := strings.Index(section, "davinci_variable_port_value")
		if aIdx < 0 || bIdx < 0 || cIdx < 0 || aIdx >= bIdx || bIdx >= cIdx {
			t.Errorf("expected alphabetical order within Variable group: company_name < enabled < port; got positions: %d, %d, %d", aIdx, bIdx, cIdx)
		}
	}
}

// TestGenerator_GenerateTFVarsTemplate_WithValues verifies tfvars generation with actual values
func TestGenerator_GenerateTFVarsTemplate_WithValues(t *testing.T) {
	tmpDir := t.TempDir()

	config := ModuleConfig{
		OutputDir:     tmpDir,
		ModuleDirName: "davinci-module",
		IncludeValues: true,
		EnvironmentID: "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
	}
	generator := NewGenerator(config)

	// Create test module structure with variables having default values
	structure := &ModuleStructure{
		Config: config,
		Variables: []Variable{
			{
				Name:         "davinci_variable_company_name_value",
				Type:         "string",
				Description:  "Value for DaVinci variable: CompanyName",
				Default:      "ACME Corporation",
				ResourceType: "variable",
				ResourceName: "company_name",
			},
			{
				Name:         "davinci_variable_secret_key_value",
				Type:         "string",
				Description:  "Value for DaVinci variable: SecretKey",
				Default:      "********", // Masked by API
				Sensitive:    true,
				IsSecret:     true,
				ResourceType: "variable",
				ResourceName: "secret_key",
			},
			{
				Name:         "davinci_variable_port_value",
				Type:         "number",
				Description:  "Port number",
				Default:      8080,
				ResourceType: "variable",
				ResourceName: "port",
			},
			{
				Name:         "davinci_variable_enabled_value",
				Type:         "bool",
				Description:  "Feature enabled flag",
				Default:      true,
				ResourceType: "variable",
				ResourceName: "enabled",
			},
		},
	}

	// Generate tfvars with values
	err := generator.generateTFVarsFile(structure)
	require.NoError(t, err)

	// Verify file exists
	tfvarsPath := filepath.Join(tmpDir, "ping-export-terraform.auto.tfvars")
	require.FileExists(t, tfvarsPath)

	// Read and verify content
	content, err := os.ReadFile(tfvarsPath)
	require.NoError(t, err)
	contentStr := string(content)

	// Verify environment_id with actual value
	assert.Contains(t, contentStr, `environment_id = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"`)
	assert.NotContains(t, contentStr, "# TODO: Provide PingOne environment ID")

	// Verify non-secret string variable with actual value
	assert.Contains(t, contentStr, `davinci_variable_company_name_value = "ACME Corporation"`)
	assert.NotContains(t, contentStr, `davinci_variable_company_name_value = ""`)

	// Verify secret variable still empty (even with --include-values)
	assert.Contains(t, contentStr, `davinci_variable_secret_key_value = ""`)
	assert.Contains(t, contentStr, "# Secret value")
	assert.NotContains(t, contentStr, "********")

	// Verify number variable with actual value
	assert.Contains(t, contentStr, `davinci_variable_port_value = 8080`)
	assert.NotContains(t, contentStr, `davinci_variable_port_value = 0`)

	// Verify bool variable with actual value
	assert.Contains(t, contentStr, `davinci_variable_enabled_value = true`)
	assert.NotContains(t, contentStr, `davinci_variable_enabled_value = false`)

	// Verify grouping comments
	assert.Contains(t, contentStr, "# Variable Variables")

	// Verify alphabetical sorting within the Variable group
	varPos := strings.Index(contentStr, "# Variable Variables")
	if varPos >= 0 {
		section := contentStr[varPos:]
		aIdx := strings.Index(section, "davinci_variable_company_name_value")
		bIdx := strings.Index(section, "davinci_variable_enabled_value")
		cIdx := strings.Index(section, "davinci_variable_port_value")
		if aIdx < 0 || bIdx < 0 || cIdx < 0 || aIdx >= bIdx || bIdx >= cIdx {
			t.Errorf("expected alphabetical order within Variable group: company_name < enabled < port; got positions: %d, %d, %d", aIdx, bIdx, cIdx)
		}
	}
}

// TestGenerator_GenerateTFVars_EscapesHCLTemplateSequences verifies that
// string values containing "${" or "%{" are escaped to "$${" / "%%{" so
// Terraform does not interpret them as interpolation / directives.
func TestGenerator_GenerateTFVars_EscapesHCLTemplateSequences(t *testing.T) {
	tmpDir := t.TempDir()

	config := ModuleConfig{
		OutputDir:     tmpDir,
		ModuleDirName: "davinci-module",
		IncludeValues: true,
		EnvironmentID: "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
	}
	generator := NewGenerator(config)

	// JavaScript code snippet that contains JS template literal "${...}" patterns.
	jsCode := "const linkparsed = `[${parsed.stringA},${parsed.stringB}]`"

	structure := &ModuleStructure{
		Config: config,
		Variables: []Variable{
			{
				Name:         "davinci_connection_Code_0020_Snippet_code",
				Type:         "string",
				Description:  "code property for Code Snippet connector",
				Default:      jsCode,
				ResourceType: "pingone_davinci_connector_instance",
				ResourceName: "Code_0020_Snippet",
			},
			{
				Name:         "davinci_connection_test_directive",
				Type:         "string",
				Description:  "value with HCL directive syntax",
				Default:      "before %{if true}yes%{endif} after",
				ResourceType: "pingone_davinci_connector_instance",
				ResourceName: "test",
			},
		},
	}

	err := generator.generateTFVarsFile(structure)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(tmpDir, "ping-export-terraform.auto.tfvars"))
	require.NoError(t, err)
	contentStr := string(content)

	// "${" must be escaped to "$${" so Terraform treats it as literal text.
	assert.Contains(t, contentStr, `$${parsed.stringA}`)
	assert.Contains(t, contentStr, `$${parsed.stringB}`)

	// "%{" must be escaped to "%%{" so Terraform treats it as literal text.
	assert.Contains(t, contentStr, `%%{if true}`)
	assert.Contains(t, contentStr, `%%{endif}`)

	// Ensure no unescaped "${" remains (other than the escaped form "$${").
	// Remove all escaped sequences then check for bare "${".
	stripped := strings.ReplaceAll(contentStr, "$${", "")
	// The environment_id line uses %q which doesn't produce "${", so no issues.
	assert.NotContains(t, stripped, "${parsed.")
}

// TestGenerator_DefaultModuleName tests that the default module name "ping-export" is used in module.tf
func TestGenerator_DefaultModuleName(t *testing.T) {
	tmpDir := t.TempDir()

	config := ModuleConfig{
		OutputDir:      tmpDir,
		ModuleDirName:  "ping-export-module",
		ModuleName:     "ping-export", // Default module name
		IncludeImports: false,
		IncludeValues:  false,
		EnvironmentID:  "test-env-123",
	}

	structure := &ModuleStructure{
		Config: config,
		Resources: ModuleResources{
			"pingone_davinci_flow": "resource \"pingone_davinci_flow\" \"test\" {}\n",
		},
	}

	generator := NewGenerator(config)
	err := generator.Generate(structure)
	require.NoError(t, err)

	// Read module.tf
	moduleContent, err := os.ReadFile(filepath.Join(tmpDir, "ping-export-module.tf"))
	require.NoError(t, err)
	contentStr := string(moduleContent)

	// Verify module block uses "ping-export" as module name
	assert.Contains(t, contentStr, `module "ping-export" {`)
	assert.NotContains(t, contentStr, `module "davinci" {`)

	// Verify source references the module directory
	assert.Contains(t, contentStr, `source = "./ping-export-module"`)
}

// TestGenerator_CustomModuleName tests that custom module names work correctly
func TestGenerator_CustomModuleName(t *testing.T) {
	tmpDir := t.TempDir()

	config := ModuleConfig{
		OutputDir:      tmpDir,
		ModuleDirName:  "my-custom-module",
		ModuleName:     "my_flows", // Custom module name
		IncludeImports: false,
		IncludeValues:  false,
		EnvironmentID:  "test-env-123",
	}

	structure := &ModuleStructure{
		Config: config,
		Resources: ModuleResources{
			"pingone_davinci_flow": "resource \"pingone_davinci_flow\" \"test\" {}\n",
		},
	}

	generator := NewGenerator(config)
	err := generator.Generate(structure)
	require.NoError(t, err)

	// Read module.tf
	moduleContent, err := os.ReadFile(filepath.Join(tmpDir, "my_flows-module.tf"))
	require.NoError(t, err)
	contentStr := string(moduleContent)

	// Verify module block uses custom module name
	assert.Contains(t, contentStr, `module "my_flows" {`)
	assert.NotContains(t, contentStr, `module "ping-export" {`)
	assert.NotContains(t, contentStr, `module "davinci" {`)

	// Verify source references the module directory (not the module name)
	assert.Contains(t, contentStr, `source = "./my-custom-module"`)
}

// TestGenerator_ImportBlocksUseModuleName tests that import blocks use the module name, not the folder name
func TestGenerator_ImportBlocksUseModuleName(t *testing.T) {
	tmpDir := t.TempDir()

	config := ModuleConfig{
		OutputDir:      tmpDir,
		ModuleDirName:  "my-folder", // Folder name
		ModuleName:     "my_module", // Module name (different from folder)
		IncludeImports: true,
		IncludeValues:  false,
		EnvironmentID:  "test-env-123",
	}

	// Create import blocks that should reference the module name
	importBlocks := []ImportBlock{
		{
			To: "module.my_module.pingone_davinci_flow.test_flow",
			ID: "env-id/flow-id",
		},
		{
			To: "module.my_module.pingone_davinci_variable.test_var",
			ID: "env-id/var-id",
		},
	}

	structure := &ModuleStructure{
		Config:       config,
		ImportBlocks: importBlocks,
		Resources: ModuleResources{
			"pingone_davinci_flow": "resource \"pingone_davinci_flow\" \"test_flow\" {}\n",
		},
	}

	generator := NewGenerator(config)
	err := generator.Generate(structure)
	require.NoError(t, err)

	// Read imports.tf
	importsContent, err := os.ReadFile(filepath.Join(tmpDir, "my_module-imports.tf"))
	require.NoError(t, err)
	importsStr := string(importsContent)

	// Verify import blocks use module name, not folder name
	assert.Contains(t, importsStr, "module.my_module.pingone_davinci_flow.test_flow")
	assert.Contains(t, importsStr, "module.my_module.pingone_davinci_variable.test_var")
	assert.NotContains(t, importsStr, "module.my-folder")

	// Read module.tf to verify it uses the correct module name
	moduleContent, err := os.ReadFile(filepath.Join(tmpDir, "my_module-module.tf"))
	require.NoError(t, err)
	moduleStr := string(moduleContent)

	// Module block should use module name
	assert.Contains(t, moduleStr, `module "my_module" {`)
	// But source should use folder name
	assert.Contains(t, moduleStr, `source = "./my-folder"`)
}
