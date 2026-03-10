// Package config provides runtime configuration utilities for the export pipeline.
package config

import (
	"strings"
)

// isTruthy returns true for common truthy string values.
func isTruthy(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}
