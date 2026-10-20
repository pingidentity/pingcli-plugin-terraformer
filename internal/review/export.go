package review

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pingidentity/pingcli-plugin-terraformer/definitions"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/clients"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/core"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
)

// ClosureResult is the computed review set for the live export phase.
type ClosureResult struct {
	// Types is the dependency-closed review set (sorted).
	Types []string

	// SkippedLive maps types excluded from the live phase to the reason
	// (e.g. DaVinci in an SSO-only environment).
	SkippedLive map[string]string

	// FallbackTargets lists referenced types with no definition — the export
	// will emit fallback variables for them.
	FallbackTargets []string

	// Registry holds exactly the closure's definitions.
	Registry *schema.Registry
}

// isDaVinciType reports whether a resource type's definition lives under
// definitions/pingone/davinci/ — i.e. requires the DaVinci service.
func isDaVinciType(def *schema.ResourceDefinition) bool {
	return strings.Contains(def.API.SDKPackage, "pingone-go-client/davinci")
}

// ComputeClosure builds the dependency-closed review set for the given seed
// types from the embedded definitions.
//
// Seeds always include pingone_environment. Edges walked: dependencies.
// depends_on, every references_type (recursive through nested attributes),
// and embedded-reference-rule targets for types already in the set. Types
// referenced but without a definition are recorded as FallbackTargets.
// Definitions under definitions/pingone/davinci/ are excluded from the live
// set (SSO-only throwaway environments lack the DaVinci service; #163) and
// reported in SkippedLive.
func ComputeClosure(seedTypes []string, embeddedRules []core.EmbeddedReferenceRule) (*ClosureResult, error) {
	// Bootstrap: load all definitions into a lookup map.
	loader := schema.NewLoader()
	allDefs, err := loader.LoadFromFS(definitions.FS, "pingone")
	if err != nil {
		return nil, fmt.Errorf("load definitions: %w", err)
	}
	byType := make(map[string]*schema.ResourceDefinition, len(allDefs))
	for _, def := range allDefs {
		byType[def.Metadata.ResourceType] = def
	}

	result := &ClosureResult{
		SkippedLive: map[string]string{},
	}

	inSet := map[string]bool{}
	working := map[string]bool{}

	// add walks to fixpoint from a type.
	var add func(rt string)
	add = func(rt string) {
		if inSet[rt] || working[rt] {
			return
		}
		def, ok := byType[rt]
		if !ok {
			// Referenced but not defined — fallback-variable target.
			return
		}
		working[rt] = true

		// SSO-only guard: DaVinci types never join the live set.
		if isDaVinciType(def) {
			result.SkippedLive[rt] = "DaVinci service unavailable in SSO-only environment (see #163)"
			delete(working, rt)
			return
		}

		// Walk declared dependencies.
		for _, dep := range def.Dependencies.DependsOn {
			add(dep.ResourceType)
		}
		// Walk references_type targets (recursive through nested attrs).
		targets := map[string]bool{}
		collectRefTargets(def.Attributes, targets)
		refOrder := make([]string, 0, len(targets))
		for t := range targets {
			refOrder = append(refOrder, t)
		}
		sort.Strings(refOrder)
		for _, t := range refOrder {
			add(t)
		}
		// Walk embedded-reference rules declared by this type.
		for _, rule := range embeddedRules {
			if rule.ResourceType == rt && rule.TargetResourceType != "" {
				add(rule.TargetResourceType)
			}
		}

		delete(working, rt)
		inSet[rt] = true
	}

	add("pingone_environment")
	for _, rt := range seedTypes {
		add(rt)
	}

	// Fallback targets: referenced types absent from byType AND from the set.
	fallbackSeen := map[string]bool{}
	for rt := range inSet {
		def := byType[rt]
		targets := map[string]bool{}
		collectRefTargets(def.Attributes, targets)
		for _, rule := range embeddedRules {
			if rule.ResourceType == rt && rule.TargetResourceType != "" {
				targets[rule.TargetResourceType] = true
			}
		}
		for t := range targets {
			if !inSet[t] && !fallbackSeen[t] {
				if _, defined := byType[t]; !defined || result.SkippedLive[t] != "" {
					fallbackSeen[t] = true
					result.FallbackTargets = append(result.FallbackTargets, t)
				}
			}
		}
	}
	sort.Strings(result.FallbackTargets)

	// Build the filtered registry.
	result.Registry = schema.NewRegistry()
	for rt := range inSet {
		if err := result.Registry.Register(byType[rt]); err != nil {
			return nil, fmt.Errorf("register %s: %w", rt, err)
		}
		result.Types = append(result.Types, rt)
	}
	sort.Strings(result.Types)

	return result, nil
}

// ExportOptions configures the restricted live export.
type ExportOptions struct {
	// Closure is the computed review set.
	Closure *ClosureResult

	// EnvironmentID is the target environment to export.
	EnvironmentID string

	// Client is the constructed API client.
	Client clients.APIClient

	// ModuleName and ModuleDir name the generated module.
	ModuleName string
	ModuleDir  string

	// OutputDir receives the module output.
	OutputDir string
}

// ExportStats summarizes the restricted export for the checklist.
type ExportStats struct {
	// FallbackVariables are the fallback-variable names the export emitted.
	FallbackVariables []string

	// UnresolvedUUIDs maps resource_type → raw UUID count in generated HCL.
	UnresolvedUUIDs map[string]int
}

// uuidPattern is the unresolved-UUID regex from the new-resource skill's
// Phase 5b check, pre-compiled for the post-export scan.
// (Kept as a plain string here; compiled in the tool's scan step.)
const unresolvedUUIDDoc = `"([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})"`

// ReviewEnvironmentAlwaysPresent documents that pingone_environment is always
// in the closure: the environment handler lists the single target environment
// and the base definition is the only always-required type.
const ReviewEnvironmentAlwaysPresent = "pingone_environment"

// liveTypeOrder returns the closure types in export order (the orchestrator
// orders internally; this is for reporting).
func liveTypeOrder(c *ClosureResult) []string {
	out := make([]string, len(c.Types))
	copy(out, c.Types)
	sort.Strings(out)
	return out
}

// DescribeClosure renders the closure for summary/meta consumption.
func DescribeClosure(c *ClosureResult) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Review set (%d types): %s", len(c.Types), joinStrings(liveTypeOrder(c)))
	if len(c.SkippedLive) > 0 {
		skipped := make([]string, 0, len(c.SkippedLive))
		for rt := range c.SkippedLive {
			skipped = append(skipped, rt)
		}
		sort.Strings(skipped)
		fmt.Fprintf(&sb, "; skipped live: %s", joinStrings(skipped))
	}
	if len(c.FallbackTargets) > 0 {
		fmt.Fprintf(&sb, "; fallback targets: %s", joinStrings(c.FallbackTargets))
	}
	return sb.String()
}
