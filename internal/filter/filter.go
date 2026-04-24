// Copyright © 2025 Ping Identity Corporation
package filter

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// Pattern represents a compiled pattern matcher (glob or regex).
type Pattern struct {
	raw      string
	isRegex  bool
	compiled *regexp.Regexp // Only set if isRegex is true
}

// ParsePattern parses a raw pattern string.
// If prefixed with "regex:", compiles as regex (case-insensitive).
// Otherwise, treats as glob (case-insensitive).
// Returns error only for invalid regex or malformed glob.
func ParsePattern(raw string) (*Pattern, error) {
	p := &Pattern{
		raw:     raw,
		isRegex: false,
	}

	if strings.HasPrefix(raw, "regex:") {
		// Regex pattern: remove prefix and compile (lowercase)
		p.isRegex = true
		regexStr := raw[6:] // Remove "regex:" prefix
		loweredRegex := strings.ToLower(regexStr)
		compiled, err := regexp.Compile(loweredRegex)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}
		p.compiled = compiled
	} else {
		// Glob pattern: validate at parse time by attempting match
		loweredPattern := strings.ToLower(raw)
		_, err := filepath.Match(loweredPattern, "")
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern: %w", err)
		}
	}

	return p, nil
}

// Match returns true if address matches this pattern. Case-insensitive.
func (p *Pattern) Match(address string) bool {
	if p.isRegex {
		// Regex: lowercase address and use compiled pattern
		loweredAddress := strings.ToLower(address)
		return p.compiled.MatchString(loweredAddress)
	}

	// Glob: lowercase both pattern and address, then use filepath.Match
	loweredPattern := strings.ToLower(p.raw)
	loweredAddress := strings.ToLower(address)
	matched, err := filepath.Match(loweredPattern, loweredAddress)
	if err != nil {
		// Should not happen since we validated at parse time
		return false
	}
	return matched
}

// String returns the original raw pattern string.
func (p *Pattern) String() string {
	return p.raw
}

// ResourceFilter applies include/exclude patterns to resource addresses.
type ResourceFilter struct {
	includes []*Pattern
	excludes []*Pattern
}

// NewResourceFilter creates a filter from include/exclude pattern strings.
// Returns error if any pattern is invalid (bad regex or glob).
func NewResourceFilter(includes, excludes []string) (*ResourceFilter, error) {
	rf := &ResourceFilter{
		includes: make([]*Pattern, 0, len(includes)),
		excludes: make([]*Pattern, 0, len(excludes)),
	}

	// Parse include patterns
	for _, inc := range includes {
		pattern, err := ParsePattern(inc)
		if err != nil {
			return nil, fmt.Errorf("invalid include pattern %q: %w", inc, err)
		}
		rf.includes = append(rf.includes, pattern)
	}

	// Parse exclude patterns
	for _, exc := range excludes {
		pattern, err := ParsePattern(exc)
		if err != nil {
			return nil, fmt.Errorf("invalid exclude pattern %q: %w", exc, err)
		}
		rf.excludes = append(rf.excludes, pattern)
	}

	return rf, nil
}

// Allow returns true if address passes the filter:
// 1. If includes exist: address must match at least one include pattern
// 2. If no includes: address is included by default
// 3. If address matches any exclude pattern: rejected
func (f *ResourceFilter) Allow(address string) bool {
	// Check exclude patterns first: if anything matches, reject
	for _, exc := range f.excludes {
		if exc.Match(address) {
			return false
		}
	}

	// If there are include patterns, address must match at least one
	if len(f.includes) > 0 {
		for _, inc := range f.includes {
			if inc.Match(address) {
				return true
			}
		}
		return false
	}

	// No include patterns: allow by default (and not excluded)
	return true
}

// IsActive returns true if any patterns are configured.
func (f *ResourceFilter) IsActive() bool {
	return len(f.includes) > 0 || len(f.excludes) > 0
}

// IsExplicitlyExcluded returns true if address matches any exclude pattern.
func (f *ResourceFilter) IsExplicitlyExcluded(address string) bool {
	for _, exc := range f.excludes {
		if exc.Match(address) {
			return true
		}
	}
	return false
}
