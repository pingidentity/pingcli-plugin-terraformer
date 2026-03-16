// Copyright © 2025 Ping Identity Corporation

package utils

import (
	"fmt"
	"regexp"
	"strings"
)

// SanitizeResourceName converts a resource name to a valid Terraform resource name
// using the same logic as pingcli's ImportBlock.Sanitize() method.
// This ensures consistency between the converter and pingcli export functionality.
//
// The sanitization process:
// 1. Hexadecimal encodes special characters (anything not alphanumeric, underscore, or hyphen)
// 2. Prefixes the name with "pingcli__"
//
// Examples:
//   - "Customer" -> "pingcli__Customer"
//   - "Customer HTML Form (PF)" -> "pingcli__Customer-0020-HTML-0020-Form-0020--0028-PF-0029-"
//   - "Customer@HTML#Form$PF%" -> "pingcli__Customer-0040-HTML-0023-Form-0024-PF-0025-"
func SanitizeResourceName(name string) string {
	// Hexadecimal encode special characters
	name = regexp.MustCompile(`[^0-9A-Za-z_\-]`).ReplaceAllStringFunc(name, func(s string) string {
		return fmt.Sprintf("-%04X-", s)
	})
	// Prefix resource names with pingcli__ (8 chars: "pingcli_")
	return "pingcli__" + name
}

// SanitizeMultiKeyResourceName creates a unique resource name by combining multiple key components.
// This prevents naming conflicts when resources share common attributes but differ in others.
//
// The sanitization process:
// 1. Hexadecimal encodes special characters in each key (anything not alphanumeric, underscore, or hyphen)
// 2. Joins all keys with underscores
// 3. Prefixes the result with "pingcli__"
//
// Examples:
//   - ("origin", "company") -> "pingcli__origin_company"
//   - ("origin", "flowInstance") -> "pingcli__origin_flowInstance"
//   - ("enableFeatureX", "company") -> "pingcli__enableFeatureX_company"
//   - ("API Key", "user", "profile") -> "pingcli__API-0020-Key_user_profile"
//
// Use cases:
//   - DaVinci variables: name + context (company, flowInstance, user, flow)
//   - Future resources with composite keys
func SanitizeMultiKeyResourceName(keys ...string) string {
	if len(keys) == 0 {
		return "pingcli__"
	}

	// Sanitize each key individually
	sanitizedKeys := make([]string, len(keys))
	for i, key := range keys {
		sanitizedKeys[i] = regexp.MustCompile(`[^0-9A-Za-z_\-]`).ReplaceAllStringFunc(key, func(s string) string {
			return fmt.Sprintf("-%04X-", s)
		})
	}

	// Join with underscores and prefix
	return fmt.Sprintf("pingcli__%s", strings.Join(sanitizedKeys, "_"))
}

// SanitizeVariableResourceName creates a unique resource name for DaVinci variables
// by combining the variable name and context. This is a convenience wrapper around
// SanitizeMultiKeyResourceName for the common case of variable resources.
//
// Deprecated: Use SanitizeMultiKeyResourceName(name, context) instead.
func SanitizeVariableResourceName(name, context string) string {
	return SanitizeMultiKeyResourceName(name, context)
}

// SanitizeVariableName produces a valid Terraform variable name by replacing
// any character that is not alphanumeric or underscore with an underscore.
// Unlike SanitizeResourceName, this does not add a prefix or hex-encode specials,
// since Terraform variables only require [a-zA-Z0-9_].
//
// Examples:
//   - "simple" -> "simple"
//   - "with-dashes" -> "with_dashes"
//   - "dots.and.slashes/here" -> "dots_and_slashes_here"
//   - "a b c" -> "a_b_c"
func SanitizeVariableName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}
