// Copyright © 2025 Ping Identity Corporation
package filter_test

import (
	"testing"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- ParsePattern Tests ---

func TestParsePattern(t *testing.T) {
	testCases := []struct {
		name      string
		input     string
		shouldErr bool
		description string
	}{
		{
			name:      "Glob pattern simple",
			input:     "pingone_davinci_flow*",
			shouldErr: false,
			description: "glob pattern should parse successfully",
		},
		{
			name:      "Glob pattern with question mark",
			input:     "pingone_davinci_flow?",
			shouldErr: false,
			description: "glob pattern with ? should parse successfully",
		},
		{
			name:      "Glob pattern with brackets",
			input:     "pingone_davinci_flow[abc]",
			shouldErr: false,
			description: "glob pattern with brackets should parse successfully",
		},
		{
			name:      "Regex pattern valid",
			input:     "regex:^pingone_davinci_flow\\..*$",
			shouldErr: false,
			description: "regex pattern should parse successfully",
		},
		{
			name:      "Regex pattern with alternation",
			input:     "regex:^pingone_davinci_(flow|variable)\\..*$",
			shouldErr: false,
			description: "regex pattern with alternation should parse successfully",
		},
		{
			name:      "Regex pattern invalid unclosed group",
			input:     "regex:([invalid",
			shouldErr: true,
			description: "invalid regex should return error",
		},
		{
			name:      "Regex pattern invalid bad escape",
			input:     "regex:(?P<invalid",
			shouldErr: true,
			description: "invalid regex with bad escape should return error",
		},
		{
			name:      "Empty string",
			input:     "",
			shouldErr: false,
			description: "empty pattern should parse successfully as glob",
		},
		{
			name:      "Glob with wildcard",
			input:     "*",
			shouldErr: false,
			description: "wildcard glob should parse successfully",
		},
		{
			name:      "Glob with mixed case",
			input:     "PingOne_DaVinci_Flow*",
			shouldErr: false,
			description: "mixed case glob should parse successfully",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pattern, err := filter.ParsePattern(tc.input)
			if tc.shouldErr {
				assert.Error(t, err, tc.description)
				assert.Nil(t, pattern)
			} else {
				require.NoError(t, err, tc.description)
				require.NotNil(t, pattern)
				assert.Equal(t, tc.input, pattern.String(), "String() should return original input")
			}
		})
	}
}

// --- Pattern Match Tests ---

func TestPatternMatch(t *testing.T) {
	testCases := []struct {
		name     string
		pattern  string
		address  string
		expected bool
		description string
	}{
		// Glob matching - basic wildcards
		{
			name:     "Glob: wildcard matches extension",
			pattern:  "pingone_davinci_flow*",
			address:  "pingone_davinci_flow.pingcli__Login",
			expected: true,
			description: "glob *  should match anything after prefix",
		},
		{
			name:     "Glob: wildcard matches underscore and dot",
			pattern:  "pingone_davinci_flow*",
			address:  "pingone_davinci_flow_deploy.pingcli__Login",
			expected: true,
			description: "glob * matches _ and . characters",
		},
		{
			name:     "Glob: multi-wildcard in middle",
			pattern:  "*Login*",
			address:  "pingone_davinci_flow.pingcli__Login-0020-Flow",
			expected: true,
			description: "glob wildcards on both sides should match",
		},
		{
			name:     "Glob: no match different prefix",
			pattern:  "pingone_davinci_flow.pingcli__Login*",
			address:  "pingone_davinci_variable.pingcli__Login",
			expected: false,
			description: "glob should not match different prefix",
		},
		{
			name:     "Glob: explicit characters must match",
			pattern:  "pingone_davinci_flow.pingcli__Login*",
			address:  "pingone_davinci_flow.pingcli__Login-0020-Flow",
			expected: true,
			description: "glob with explicit characters should require exact prefix match",
		},

		// Glob - case insensitivity
		{
			name:     "Glob: case insensitive uppercase pattern",
			pattern:  "PINGONE_DAVINCI_FLOW*",
			address:  "pingone_davinci_flow.pingcli__My_Flow",
			expected: true,
			description: "glob should be case-insensitive",
		},
		{
			name:     "Glob: case insensitive lowercase match",
			pattern:  "*login*",
			address:  "pingone_davinci_flow.pingcli__Login-0020-Flow",
			expected: true,
			description: "glob should match case-insensitively",
		},
		{
			name:     "Glob: case insensitive mixed case",
			pattern:  "*LoGiN*",
			address:  "pingone_davinci_flow.pingcli__login_flow",
			expected: true,
			description: "glob should handle mixed case",
		},

		// Glob - special characters
		{
			name:     "Glob: single character match",
			pattern:  "?",
			address:  "a",
			expected: true,
			description: "glob ? should match single character",
		},
		{
			name:     "Glob: single question mark no match multiple",
			pattern:  "?",
			address:  "ab",
			expected: false,
			description: "glob ? should not match multiple characters",
		},
		{
			name:     "Glob: question mark in middle",
			pattern:  "pingone_davinci_flow?",
			address:  "pingone_davinci_flowx",
			expected: true,
			description: "glob ? should match single character in any position",
		},

		// Regex matching - basic patterns
		{
			name:     "Regex: alternation flow or variable",
			pattern:  "regex:^pingone_davinci_(flow|variable)\\..*$",
			address:  "pingone_davinci_flow.x",
			expected: true,
			description: "regex alternation should match first option",
		},
		{
			name:     "Regex: alternation matches second option",
			pattern:  "regex:^pingone_davinci_(flow|variable)\\..*$",
			address:  "pingone_davinci_variable.y",
			expected: true,
			description: "regex alternation should match second option",
		},
		{
			name:     "Regex: alternation no match",
			pattern:  "regex:^pingone_davinci_(flow|variable)\\..*$",
			address:  "pingone_davinci_application.z",
			expected: false,
			description: "regex alternation should not match other types",
		},

		// Regex - case insensitivity
		{
			name:     "Regex: case insensitive uppercase pattern",
			pattern:  "regex:.*LOGIN.*",
			address:  "pingone_davinci_flow.pingcli__login_flow",
			expected: true,
			description: "regex should be case-insensitive",
		},
		{
			name:     "Regex: case insensitive lowercase address",
			pattern:  "regex:.*TEST.*",
			address:  "pingone_davinci_flow.pingcli__test_flow",
			expected: true,
			description: "regex should match case-insensitively on lowercase address",
		},

		// Regex - anchors
		{
			name:     "Regex: anchor start",
			pattern:  "regex:^pingone_davinci_flow",
			address:  "pingone_davinci_flow.x",
			expected: true,
			description: "regex ^ anchor should match start",
		},
		{
			name:     "Regex: anchor start no match",
			pattern:  "regex:^application",
			address:  "pingone_davinci_application.x",
			expected: false,
			description: "regex ^ anchor should not match middle",
		},

		// Edge cases
		{
			name:     "Glob: empty pattern matches empty string",
			pattern:  "",
			address:  "",
			expected: true,
			description: "empty glob should match empty address",
		},
		{
			name:     "Glob: empty pattern no match non-empty",
			pattern:  "",
			address:  "pingone_davinci_flow.x",
			expected: false,
			description: "empty glob should not match non-empty address",
		},
		{
			name:     "Glob: wildcard matches empty suffix",
			pattern:  "pingone_davinci_flow*",
			address:  "pingone_davinci_flow",
			expected: true,
			description: "glob * can match zero characters",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pattern, err := filter.ParsePattern(tc.pattern)
			require.NoError(t, err, "pattern should parse without error")
			require.NotNil(t, pattern)

			result := pattern.Match(tc.address)
			assert.Equal(t, tc.expected, result, tc.description)
		})
	}
}

// --- ResourceFilter Allow Tests ---

func TestResourceFilterAllow(t *testing.T) {
	testCases := []struct {
		name       string
		includes   []string
		excludes   []string
		address    string
		expected   bool
		description string
	}{
		// No patterns - empty filter
		{
			name:     "Empty filter allows all",
			includes: []string{},
			excludes: []string{},
			address:  "pingone_davinci_flow.x",
			expected: true,
			description: "empty filter should allow any address",
		},
		{
			name:     "Empty filter empty address",
			includes: []string{},
			excludes: []string{},
			address:  "",
			expected: true,
			description: "empty filter should allow empty address",
		},

		// Include only
		{
			name:     "Include single pattern matching",
			includes: []string{"pingone_davinci_flow*"},
			excludes: []string{},
			address:  "pingone_davinci_flow.x",
			expected: true,
			description: "should allow when include pattern matches",
		},
		{
			name:     "Include single pattern not matching",
			includes: []string{"pingone_davinci_flow*"},
			excludes: []string{},
			address:  "pingone_davinci_variable.y",
			expected: false,
			description: "should reject when include pattern doesn't match",
		},

		// Multiple includes (OR union)
		{
			name:     "Multiple includes first matches",
			includes: []string{"pingone_davinci_flow*", "pingone_davinci_variable*"},
			excludes: []string{},
			address:  "pingone_davinci_flow.x",
			expected: true,
			description: "should allow when first include pattern matches",
		},
		{
			name:     "Multiple includes second matches",
			includes: []string{"pingone_davinci_flow*", "pingone_davinci_variable*"},
			excludes: []string{},
			address:  "pingone_davinci_variable.y",
			expected: true,
			description: "should allow when second include pattern matches",
		},
		{
			name:     "Multiple includes none match",
			includes: []string{"pingone_davinci_flow*", "pingone_davinci_variable*"},
			excludes: []string{},
			address:  "pingone_application.z",
			expected: false,
			description: "should reject when no include patterns match",
		},

		// Exclude only
		{
			name:     "Exclude only not excluded",
			includes: []string{},
			excludes: []string{"*Test*"},
			address:  "pingone_davinci_flow.pingcli__Login",
			expected: true,
			description: "exclude-only filter should allow when exclude doesn't match",
		},
		{
			name:     "Exclude only excluded",
			includes: []string{},
			excludes: []string{"*Test*"},
			address:  "pingone_davinci_flow.pingcli__Test_Flow",
			expected: false,
			description: "exclude-only filter should reject when exclude matches",
		},

		// Include + Exclude combination
		{
			name:     "Include and exclude both pass",
			includes: []string{"pingone_davinci_flow*"},
			excludes: []string{"*Test*"},
			address:  "pingone_davinci_flow.pingcli__Login",
			expected: true,
			description: "should allow when include matches and exclude doesn't",
		},
		{
			name:     "Include matches but exclude also matches",
			includes: []string{"pingone_davinci_flow*"},
			excludes: []string{"*Test*"},
			address:  "pingone_davinci_flow.pingcli__Test_Flow",
			expected: false,
			description: "should reject when include matches but exclude also matches",
		},
		{
			name:     "Include match not excluded - different type",
			includes: []string{"pingone_davinci_flow*"},
			excludes: []string{"*Test*"},
			address:  "pingone_davinci_variable.y",
			expected: false,
			description: "should reject when include doesn't match (even if exclude doesn't match)",
		},

		// Case insensitivity
		{
			name:     "Exclude with case insensitive uppercase",
			includes: []string{},
			excludes: []string{"*DEV*"},
			address:  "pingone_davinci_flow.pingcli__Login_dev",
			expected: false,
			description: "exclude should be case-insensitive",
		},
		{
			name:     "Include with case insensitive",
			includes: []string{"*FLOW*"},
			excludes: []string{},
			address:  "pingone_davinci_flow.x",
			expected: true,
			description: "include should be case-insensitive",
		},

		// Edge cases
		{
			name:     "Include and exclude both exist empty address",
			includes: []string{"pingone_davinci_flow*"},
			excludes: []string{"*Test*"},
			address:  "",
			expected: false,
			description: "empty address should not match include patterns",
		},
		{
			name:     "Single include pattern matches everything",
			includes: []string{"*"},
			excludes: []string{},
			address:  "pingone_davinci_flow.x",
			expected: true,
			description: "wildcard include should match everything",
		},
		{
			name:     "Regex include pattern",
			includes: []string{"regex:^pingone_davinci_(flow|variable)\\..*$"},
			excludes: []string{},
			address:  "pingone_davinci_flow.x",
			expected: true,
			description: "regex include pattern should work",
		},
		{
			name:     "Regex exclude pattern",
			includes: []string{},
			excludes: []string{"regex:.*_test$"},
			address:  "pingone_davinci_flow_prod",
			expected: true,
			description: "regex exclude pattern should work for non-matching",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filter, err := filter.NewResourceFilter(tc.includes, tc.excludes)
			require.NoError(t, err, "NewResourceFilter should not error")
			require.NotNil(t, filter)

			result := filter.Allow(tc.address)
			assert.Equal(t, tc.expected, result, tc.description)
		})
	}
}

// --- ResourceFilter IsActive Tests ---

func TestResourceFilterIsActive(t *testing.T) {
	testCases := []struct {
		name        string
		includes    []string
		excludes    []string
		expected    bool
		description string
	}{
		{
			name:        "Empty patterns not active",
			includes:    []string{},
			excludes:    []string{},
			expected:    false,
			description: "filter with no patterns should not be active",
		},
		{
			name:        "Include patterns active",
			includes:    []string{"pingone_davinci_flow*"},
			excludes:    []string{},
			expected:    true,
			description: "filter with include patterns should be active",
		},
		{
			name:        "Exclude patterns active",
			includes:    []string{},
			excludes:    []string{"*Test*"},
			expected:    true,
			description: "filter with exclude patterns should be active",
		},
		{
			name:        "Both include and exclude active",
			includes:    []string{"pingone_davinci_flow*"},
			excludes:    []string{"*Test*"},
			expected:    true,
			description: "filter with both include and exclude should be active",
		},
		{
			name:        "Multiple includes active",
			includes:    []string{"pingone_davinci_flow*", "pingone_davinci_variable*"},
			excludes:    []string{},
			expected:    true,
			description: "filter with multiple includes should be active",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filter, err := filter.NewResourceFilter(tc.includes, tc.excludes)
			require.NoError(t, err)
			require.NotNil(t, filter)

			result := filter.IsActive()
			assert.Equal(t, tc.expected, result, tc.description)
		})
	}
}

// --- Error Cases ---

func TestNewResourceFilterErrors(t *testing.T) {
	testCases := []struct {
		name        string
		includes    []string
		excludes    []string
		shouldErr   bool
		description string
	}{
		{
			name:        "Empty patterns no error",
			includes:    []string{},
			excludes:    []string{},
			shouldErr:   false,
			description: "empty patterns should not error",
		},
		{
			name:        "Valid glob patterns no error",
			includes:    []string{"pingone_davinci_flow*"},
			excludes:    []string{"*_test*"},
			shouldErr:   false,
			description: "valid glob patterns should not error",
		},
		{
			name:        "Valid regex patterns no error",
			includes:    []string{"regex:^pingone_davinci_.*"},
			excludes:    []string{},
			shouldErr:   false,
			description: "valid regex patterns should not error",
		},
		{
			name:        "Invalid regex in includes",
			includes:    []string{"regex:([invalid"},
			excludes:    []string{},
			shouldErr:   true,
			description: "invalid regex in includes should error",
		},
		{
			name:        "Invalid regex in excludes",
			includes:    []string{},
			excludes:    []string{"regex:(?P<bad"},
			shouldErr:   true,
			description: "invalid regex in excludes should error",
		},
		{
			name:        "Invalid regex in mixed valid patterns",
			includes:    []string{"pingone_davinci_flow*"},
			excludes:    []string{"regex:([invalid"},
			shouldErr:   true,
			description: "one invalid regex should cause error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filter, err := filter.NewResourceFilter(tc.includes, tc.excludes)
			if tc.shouldErr {
				assert.Error(t, err, tc.description)
				assert.Nil(t, filter)
			} else {
				require.NoError(t, err, tc.description)
				require.NotNil(t, filter)
			}
		})
	}
}
