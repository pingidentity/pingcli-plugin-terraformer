package core

import (
	"fmt"
	"strings"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/utils"
)

// fallbackVariableAllocator assigns Terraform variable names for fallback
// variable references, keyed by resource identity (dedupKey) rather than by
// the derived name alone.
//
// Without this, two distinct resources that reference different UUIDs but
// derive the same human-readable or schema-derived variable name silently
// collapse onto one variable: the first UUID wins as the variable's default,
// and every other resource is rewritten to point at that same (wrong) ID with
// no error or warning (see #130, #138). This allocator ensures a given
// dedupKey always resolves to the same name (so genuine repeats of the same
// dedupKey still dedupe to one variable), and disambiguates the name — using
// a caller-supplied hint, not the dedupKey itself — only when two distinct
// dedupKeys would otherwise collide on the same base name.
//
// The disambiguation hint is deliberately NOT the target UUID being
// resolved: that UUID is per-environment API data, so baking it into the
// variable *name* (as opposed to its default value) would make the same
// logical reference site produce a different variable name in every
// environment — defeating the point of one checked-in module with
// per-environment tfvars. Callers should pass a hint derived from the
// *referencing* side's own stable identity (e.g. a resource's Terraform
// label, or a DaVinci node's own id), which is consistent across
// environments for the same exported configuration.
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
// is disambiguated using suffixHint — a stable, environment-portable
// identifier for the referencing side (e.g. a resource label or node id) —
// so the two identities never share a variable.
func (a *fallbackVariableAllocator) allocate(dedupKey, baseName, suffixHint string) string {
	if existing, ok := a.byKey[dedupKey]; ok {
		return existing
	}

	name := baseName
	if claimedBy, exists := a.byName[name]; exists && claimedBy != dedupKey {
		candidate := baseName + "_" + sanitizeHint(suffixHint)
		name = candidate
		// Extremely unlikely (would require two unrelated dedupKeys to also
		// share the identical suffixHint), but guarantee uniqueness with a
		// numeric tiebreaker rather than silently reusing a claimed name.
		for n := 2; ; n++ {
			claimedBy, exists := a.byName[name]
			if !exists || claimedBy == dedupKey {
				break
			}
			name = fmt.Sprintf("%s_%d", candidate, n)
		}
	}

	a.byKey[dedupKey] = name
	a.byName[name] = dedupKey
	return name
}

// sanitizeHint normalizes a disambiguation hint for use in a Terraform
// variable name.
func sanitizeHint(hint string) string {
	return utils.SanitizeVariableName(strings.ToLower(hint))
}
