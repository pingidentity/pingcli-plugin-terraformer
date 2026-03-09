// Copyright © 2025 Ping Identity Corporation
package utils_test

import (
	"testing"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/utils"
)

func TestSanitizeResourceName(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple alphanumeric",
			input:    "Customer",
			expected: "pingcli__Customer",
		},
		{
			name:     "Alphanumeric with capitals",
			input:    "CustomerHTMLFormPF",
			expected: "pingcli__CustomerHTMLFormPF",
		},
		{
			name:     "Spaces and parentheses",
			input:    "Customer HTML Form (PF)",
			expected: "pingcli__Customer-0020-HTML-0020-Form-0020--0028-PF-0029-",
		},
		{
			name:     "Special characters",
			input:    "Customer@HTML#Form$PF%",
			expected: "pingcli__Customer-0040-HTML-0023-Form-0024-PF-0025-",
		},
		{
			name:     "Flow with spaces",
			input:    "My Registration Flow",
			expected: "pingcli__My-0020-Registration-0020-Flow",
		},
		{
			name:     "Flow with special chars",
			input:    "Login & Signup",
			expected: "pingcli__Login-0020--0026--0020-Signup",
		},
		{
			name:     "Underscore preserved",
			input:    "flow_name",
			expected: "pingcli__flow_name",
		},
		{
			name:     "Hyphen preserved",
			input:    "flow-name",
			expected: "pingcli__flow-name",
		},
		{
			name:     "Mixed case with numbers",
			input:    "Flow123Test",
			expected: "pingcli__Flow123Test",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := utils.SanitizeResourceName(tc.input)
			if result != tc.expected {
				t.Errorf("SanitizeResourceName(%q) = %q, expected %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestSanitizeMultiKeyResourceName(t *testing.T) {
	testCases := []struct {
		name     string
		keys     []string
		expected string
	}{
		{
			name:     "Single key",
			keys:     []string{"origin"},
			expected: "pingcli__origin",
		},
		{
			name:     "Two keys - variable name and context",
			keys:     []string{"origin", "company"},
			expected: "pingcli__origin_company",
		},
		{
			name:     "Two keys - variable name and flowInstance context",
			keys:     []string{"origin", "flowInstance"},
			expected: "pingcli__origin_flowInstance",
		},
		{
			name:     "Three keys",
			keys:     []string{"config", "user", "profile"},
			expected: "pingcli__config_user_profile",
		},
		{
			name:     "Keys with special characters",
			keys:     []string{"API Key", "user"},
			expected: "pingcli__API-0020-Key_user",
		},
		{
			name:     "Empty keys array",
			keys:     []string{},
			expected: "pingcli__",
		},
		{
			name:     "Keys with spaces in multiple components",
			keys:     []string{"My Config", "Flow Instance"},
			expected: "pingcli__My-0020-Config_Flow-0020-Instance",
		},
		{
			name:     "Keys with underscores and hyphens preserved",
			keys:     []string{"flow_variable", "flow-instance"},
			expected: "pingcli__flow_variable_flow-instance",
		},
		{
			name:     "Four keys for complex composite resource",
			keys:     []string{"resource", "type", "context", "id"},
			expected: "pingcli__resource_type_context_id",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := utils.SanitizeMultiKeyResourceName(tc.keys...)
			if result != tc.expected {
				t.Errorf("SanitizeMultiKeyResourceName(%v) = %q, expected %q",
					tc.keys, result, tc.expected)
			}
		})
	}
}

func TestSanitizeVariableResourceName(t *testing.T) {
	testCases := []struct {
		name     string
		varName  string
		context  string
		expected string
	}{
		{
			name:     "Simple name with company context",
			varName:  "origin",
			context:  "company",
			expected: "pingcli__origin_company",
		},
		{
			name:     "Simple name with flowInstance context",
			varName:  "origin",
			context:  "flowInstance",
			expected: "pingcli__origin_flowInstance",
		},
		{
			name:     "CamelCase name with company context",
			varName:  "enableFeatureX",
			context:  "company",
			expected: "pingcli__enableFeatureX_company",
		},
		{
			name:     "CamelCase name with user context",
			varName:  "apiKey",
			context:  "user",
			expected: "pingcli__apiKey_user",
		},
		{
			name:     "Name with space and flow context",
			varName:  "API Key",
			context:  "flow",
			expected: "pingcli__API-0020-Key_flow",
		},
		{
			name:     "Name with special characters and company context",
			varName:  "config@value#1",
			context:  "company",
			expected: "pingcli__config-0040-value-0023-1_company",
		},
		{
			name:     "companyBool from sample JSON",
			varName:  "companyBool",
			context:  "company",
			expected: "pingcli__companyBool_company",
		},
		{
			name:     "Underscore preserved with flowInstance",
			varName:  "flow_variable",
			context:  "flowInstance",
			expected: "pingcli__flow_variable_flowInstance",
		},
		{
			name:     "Hyphen preserved with user",
			varName:  "user-setting",
			context:  "user",
			expected: "pingcli__user-setting_user",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := utils.SanitizeVariableResourceName(tc.varName, tc.context)
			if result != tc.expected {
				t.Errorf("SanitizeVariableResourceName(%q, %q) = %q, expected %q",
					tc.varName, tc.context, result, tc.expected)
			}

			// Also verify it matches the generic function
			genericResult := utils.SanitizeMultiKeyResourceName(tc.varName, tc.context)
			if result != genericResult {
				t.Errorf("SanitizeVariableResourceName != SanitizeMultiKeyResourceName: %q != %q",
					result, genericResult)
			}
		})
	}
}
