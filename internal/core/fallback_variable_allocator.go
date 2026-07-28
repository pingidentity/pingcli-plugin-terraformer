package core

import (
	"strings"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/utils"
)

// fallbackVariableAllocator assigns Terraform variable names for fallback
// variable references, keyed by resource identity (a dedup key derived from
// the UUID) rather than by the derived name alone.
//
// Without this, two distinct resources that reference different UUIDs but
// derive the same human-readable or schema-derived variable name silently
// collapse onto one variable: the first UUID wins as the variable's default,
// and every other resource is rewritten to point at that same (wrong) ID with
// no error or warning (see #130, #138). This allocator ensures a given
// dedupKey always resolves to the same name (so genuine repeats of the same
// UUID still dedupe to one variable), and disambiguates the name with a
// UUID-derived suffix only when two distinct dedupKeys would otherwise
// collide on the same base name.
type fallbackVariableAllocator struct {
	byKey  map[string]string // dedupKey -> assigned variable name
	byName map[string]string // assigned variable name -> the dedupKey that claimed it
}

// newFallbackVariableAllocator creates an empty allocator. One allocator must
// be shared across an entire resolution pass (e.g. one per Export call) so
// that collisions are detected across all resources, not just within one.
func newFallbackVariableAllocator() *fallbackVariableAllocator {
	return &fallbackVariableAllocator{
		byKey:  make(map[string]string),
		byName: make(map[string]string),
	}
}

// allocate returns the variable name to use for dedupKey, deriving from
// baseName. Repeated calls with the same dedupKey always return the same
// name. When baseName is already claimed by a different dedupKey, the name
// is disambiguated using uuid so the two identities never share a variable.
func (a *fallbackVariableAllocator) allocate(dedupKey, baseName, uuid string) string {
	if existing, ok := a.byKey[dedupKey]; ok {
		return existing
	}

	name := baseName
	if claimedBy, exists := a.byName[name]; exists && claimedBy != dedupKey {
		name = baseName + "_" + uuidSuffix(uuid, 8)
		// Astronomically unlikely, but guarantee uniqueness even if the
		// short suffix also collides: fall back to the full UUID, which is
		// unique per dedupKey by construction.
		if claimedBy, exists := a.byName[name]; exists && claimedBy != dedupKey {
			name = baseName + "_" + uuidSuffix(uuid, len(uuid))
		}
	}

	a.byKey[dedupKey] = name
	a.byName[name] = dedupKey
	return name
}

// uuidSuffix returns the first n characters of uuid, sanitized for use in a
// Terraform variable name.
func uuidSuffix(uuid string, n int) string {
	s := uuid
	if len(s) > n {
		s = s[:n]
	}
	return utils.SanitizeVariableName(strings.ToLower(s))
}
