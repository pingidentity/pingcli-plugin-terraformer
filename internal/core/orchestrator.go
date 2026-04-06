package core

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/clients"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/filter"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/graph"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/utils"
)

// ExportOptions configures the export pipeline.
type ExportOptions struct {
	// SkipDependencies outputs raw UUIDs instead of Terraform references.
	SkipDependencies bool

	// GenerateImports causes import blocks to be generated alongside resources.
	GenerateImports bool

	// EnvironmentID is the PingOne environment being exported.
	EnvironmentID string

	// ResourceFilter filters resources by pattern matching on addresses.
	// nil = no filtering (all resources included).
	ResourceFilter *filter.ResourceFilter

	// ListOnly returns resources after processing but skips reference resolution.
	ListOnly bool

	// IncludeUpstream automatically includes upstream dependencies of
	// matched resources when filtering is active.
	IncludeUpstream bool
}

// ExportedResourceData holds processed data for a single resource type.
type ExportedResourceData struct {
	// ResourceType is the Terraform resource type.
	ResourceType string

	// Definition is the schema definition for this resource type.
	Definition *schema.ResourceDefinition

	// Resources is the list of processed resource data items.
	Resources []*ResourceData
}

// ExportResult contains the full output of an orchestrated export.
type ExportResult struct {
	// ResourcesByType holds processed resources keyed by resource type,
	// ordered by dependency.
	ResourcesByType []*ExportedResourceData

	// Graph is the dependency graph built during export.
	Graph *graph.DependencyGraph

	// EnvironmentID is the exported environment.
	EnvironmentID string

	// FallbackVariables holds variable declarations for references that
	// resolved to variable fallbacks (e.g., filter-excluded upstream resources).
	// The caller should include these in the module's variables.tf.
	FallbackVariables []FallbackVariable
}

// FallbackVariable represents a Terraform variable that must be declared
// because a cross-resource reference could not be resolved to an in-scope
// resource (e.g., the upstream resource was excluded by a filter).
type FallbackVariable struct {
	// Name is the Terraform variable name (e.g., "pingone_davinci_application_pingcli__my_app_id").
	Name string

	// Type is the Terraform variable type (always "string" for ID references).
	Type string

	// Description provides context for the variable.
	Description string

	// ResourceType is the referenced resource type for organizational grouping.
	ResourceType string

	// Default is the original value (e.g., UUID) to populate in tfvars when --include-values is set.
	Default interface{}
}

// ProgressFunc is called by the orchestrator to report status.
// message is a human-readable status update.
type ProgressFunc func(message string)

// ExportOrchestrator coordinates the end-to-end schema-driven export pipeline.
type ExportOrchestrator struct {
	registry     *schema.Registry
	processor    *Processor
	client       clients.APIClient
	progressFn   ProgressFunc
	embeddedRefs *EmbeddedReferenceRegistry
}

// OrchestratorOption configures optional ExportOrchestrator behaviour.
type OrchestratorOption func(*ExportOrchestrator)

// WithProgressFunc sets a progress reporting callback.
func WithProgressFunc(fn ProgressFunc) OrchestratorOption {
	return func(o *ExportOrchestrator) {
		o.progressFn = fn
	}
}

// WithEmbeddedReferences sets the registry of embedded reference rules
// for post-processing UUID references inside RawHCLValue blobs.
func WithEmbeddedReferences(reg *EmbeddedReferenceRegistry) OrchestratorOption {
	return func(o *ExportOrchestrator) {
		o.embeddedRefs = reg
	}
}

// NewExportOrchestrator creates a new ExportOrchestrator.
//
// registry must be populated with resource definitions before calling Export.
// processor must be configured with any custom handler registries.
// client is used to fetch resources from the API.
func NewExportOrchestrator(
	registry *schema.Registry,
	processor *Processor,
	client clients.APIClient,
	opts ...OrchestratorOption,
) *ExportOrchestrator {
	o := &ExportOrchestrator{
		registry:  registry,
		processor: processor,
		client:    client,
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// progress emits a progress message if a ProgressFunc was provided.
func (o *ExportOrchestrator) progress(msg string) {
	if o.progressFn != nil {
		o.progressFn(msg)
	}
}

// Export runs the full schema-driven export pipeline:
//
//  1. Discover resource types from the registry for the given service.
//  2. Order them by declared dependencies (topological sort).
//  3. For each type: fetch from API, process via Processor, populate graph.
//  4. Return processed data for all types.
//
// The caller is responsible for formatting (HCL, JSON, etc.), variable
// extraction, import block generation, and module assembly using the
// returned ExportResult.
func (o *ExportOrchestrator) Export(ctx context.Context, opts ExportOptions) (*ExportResult, error) {
	if opts.EnvironmentID == "" {
		return nil, fmt.Errorf("environment ID is required")
	}

	// 1. Discover resource types for this client's platform.
	platform := o.client.Platform()
	defs := o.registry.ListByPlatform(platform)
	if len(defs) == 0 {
		return nil, fmt.Errorf("no resource definitions found for platform %s", platform)
	}

	// 2. Determine processing order from declared dependencies.
	ordered, err := o.orderByDependencies(defs)
	if err != nil {
		return nil, fmt.Errorf("dependency ordering: %w", err)
	}

	o.progress(fmt.Sprintf("Exporting %d resource types from %s", len(ordered), platform))

	// 3. Process each resource type in dependency order.
	depGraph := graph.New()
	var results []*ExportedResourceData
	excludedIDs := make(map[string]bool) // tracks IDs excluded by filter

	for _, def := range ordered {
		resourceType := def.Metadata.ResourceType
		o.progress(fmt.Sprintf("Fetching %s...", def.Metadata.Name))

		// Fetch from API.
		rawList, err := o.client.ListResources(ctx, resourceType, opts.EnvironmentID)
		if err != nil {
			return nil, fmt.Errorf("list %s: %w", resourceType, err)
		}

		// Process each raw item through the processor.
		processed, err := o.processor.ProcessResourceList(resourceType, rawList)
		if err != nil {
			return nil, fmt.Errorf("process %s: %w", resourceType, err)
		}

		// Assign unique Terraform labels to each resource.
		if err := assignLabels(processed, def); err != nil {
			return nil, fmt.Errorf("%s: %w", resourceType, err)
		}

		// Add ALL resources to the graph before filtering so that
		// excluded resources' labels are available for variable naming
		// when downstream references fall back to variable references.
		for _, rd := range processed {
			depGraph.AddResource(rd.ResourceType, rd.ID, rd.Label)
		}

		// Apply filtering if active (skip when IncludeUpstream is true)
		// Address uses the sanitized Label (e.g. "pingone_davinci_flow.pingcli__Login-0020-Flow")
		// so that patterns match the same format shown by --list-resources.
		if opts.ResourceFilter != nil && opts.ResourceFilter.IsActive() && !opts.IncludeUpstream {
			var filtered []*ResourceData
			for _, rd := range processed {
				address := resourceType + "." + rd.Label
				if opts.ResourceFilter.Allow(address) {
					filtered = append(filtered, rd)
				} else {
					excludedIDs[rd.ID] = true
				}
			}
			processed = filtered
		}

		o.progress(fmt.Sprintf("✓ Processed %d %s resources", len(processed), def.Metadata.ShortName))

		// Exclude types with no resources only when filtering is active
		// When no filter is active, include all types even if empty
		if len(processed) == 0 && opts.ResourceFilter != nil && opts.ResourceFilter.IsActive() {
			continue
		}

		results = append(results, &ExportedResourceData{
			ResourceType: resourceType,
			Definition:   def,
			Resources:    processed,
		})
	}

	// Resolve embedded references in RawHCLValue blobs (e.g., subFlowId
	// inside jsonencode'd node properties). Must run after the graph is
	// fully populated and before upstream expansion so discovered edges
	// are visible to the graph walker.
	var embeddedFallbackVars []FallbackVariable
	if o.embeddedRefs != nil && !opts.SkipDependencies {
		embeddedFallbackVars = ResolveEmbeddedReferences(results, depGraph, o.embeddedRefs.Rules())
	}

	// When IncludeUpstream is active, build edges and apply filter + expansion
	if opts.IncludeUpstream && opts.ResourceFilter != nil && opts.ResourceFilter.IsActive() {
		results = o.applyUpstreamExpansion(results, depGraph, opts.ResourceFilter, excludedIDs)
	}

	// Return early if ListOnly flag is set.
	if opts.ListOnly {
		o.progress(fmt.Sprintf("✓ Listed %d resources across %d types", o.totalResources(results), len(results)))
		return &ExportResult{
			ResourcesByType: results,
			Graph:           depGraph,
			EnvironmentID:   opts.EnvironmentID,
		}, nil
	}

	// Emit dependency warnings if filter is active.
	if opts.ResourceFilter != nil && opts.ResourceFilter.IsActive() {
		// Warn if no resources matched
		if o.totalResources(results) == 0 {
			o.progress("Warning: filter matched 0 resources")
		}
		o.emitDependencyWarnings(results, ordered)
	}

	// 4. Resolve cross-resource references in all processed attributes.
	// This replaces raw UUID strings with ResolvedReference values so that
	// formatters can render them without needing graph access.
	var fallbackVars []FallbackVariable
	if !opts.SkipDependencies {
		fallbackVars = o.resolveReferences(results, depGraph, opts.EnvironmentID, excludedIDs)

		// Merge in fallback variables from embedded reference resolution.
		fallbackVars = append(fallbackVars, embeddedFallbackVars...)

		// Resolve runtime depends_on entries (custom handler __depends_on)
		// to Terraform labels via the dependency graph.
		for _, erd := range results {
			resolveDependsOnResources(erd.Resources, depGraph)
		}
	} else {
		// Even in skip-dependencies mode, inject the raw environment ID for
		// attributes that reference pingone_environment but have no extracted
		// value (no source_path). Without this, resources whose definitions
		// omit source_path on environment_id lose that attribute entirely.
		o.injectEnvironmentIDs(results, opts.EnvironmentID)
	}

	// 5. Validate graph.
	if err := depGraph.Validate(); err != nil {
		o.progress(fmt.Sprintf("Warning: dependency graph validation: %v", err))
	}

	o.progress(fmt.Sprintf("✓ Export complete — %d resource types, %d total resources", len(results), o.totalResources(results)))

	return &ExportResult{
		ResourcesByType:   results,
		Graph:             depGraph,
		EnvironmentID:     opts.EnvironmentID,
		FallbackVariables: fallbackVars,
	}, nil
}

// totalResources counts all individual resources across all types.
func (o *ExportOrchestrator) totalResources(results []*ExportedResourceData) int {
	n := 0
	for _, r := range results {
		n += len(r.Resources)
	}
	return n
}

// emitDependencyWarnings checks if filtered resources reference types that were
// excluded by the filter. It recursively scans all attributes (including nested)
// and warns about missing dependencies. Each resource→referenced pair is warned
// only once.
func (o *ExportOrchestrator) emitDependencyWarnings(included []*ExportedResourceData, allDefs []*schema.ResourceDefinition) {
	// Build set of included resource types
	includedTypes := make(map[string]bool)
	for _, erd := range included {
		includedTypes[erd.ResourceType] = true
	}

	// Build set of all defined types for validation
	allDefinedTypes := make(map[string]bool)
	for _, def := range allDefs {
		allDefinedTypes[def.Metadata.ResourceType] = true
	}

	// Track emitted warnings to avoid duplicates
	warned := make(map[string]bool)

	// For each included type, check what it references
	for _, erd := range included {
		o.checkAttributeReferences(erd.ResourceType, erd.Definition.Attributes, includedTypes, allDefinedTypes, warned)
	}
}

// checkAttributeReferences recursively scans attributes (including nested) for
// references and warns if a referenced type is excluded.
func (o *ExportOrchestrator) checkAttributeReferences(resourceType string, attrs []schema.AttributeDefinition, includedTypes, allDefinedTypes map[string]bool, warned map[string]bool) {
	for _, attr := range attrs {
		referencedType := attr.ReferencesType
		if referencedType == "" || referencedType == "pingone_environment" {
			continue // Skip empty or environment references
		}

		// Check if referenced type is missing from included set
		warningKey := resourceType + "->" + referencedType
		if !includedTypes[referencedType] && allDefinedTypes[referencedType] && !warned[warningKey] {
			o.progress(fmt.Sprintf("Warning: %s references %s which was excluded by filter", resourceType, referencedType))
			warned[warningKey] = true
		}

		// Recurse into nested attributes
		if len(attr.NestedAttributes) > 0 {
			o.checkAttributeReferences(resourceType, attr.NestedAttributes, includedTypes, allDefinedTypes, warned)
		}
	}
}

// resolveReferences walks all processed resources and replaces raw UUID strings
// with ResolvedReference values for any attribute that declares references_type
// in the schema. This covers:
//   - Top-level attributes (e.g. environment_id, connection_id)
//   - Nested object attributes (e.g. config.flow_ref)
//
// Environment references always resolve to variable references.
// Other references attempt graph lookup; if unknown, fall back to variable references.
// excludedIDs contains resource IDs that were removed by filtering — references to
// these produce variable fallbacks with label-derived names.
//
// Returns deduplicated FallbackVariable entries for variable declarations.
func (o *ExportOrchestrator) resolveReferences(results []*ExportedResourceData, g *graph.DependencyGraph, environmentID string, excludedIDs map[string]bool) []FallbackVariable {
	varSeen := make(map[string]bool)
	var fallbackVars []FallbackVariable

	for _, erd := range results {
		for _, rd := range erd.Resources {
			resolveAttrs(rd.Attributes, erd.Definition.Attributes, g, environmentID, excludedIDs)
			resolveCorrelatedReferences(rd.Attributes, erd.Definition.Attributes)
		}
	}

	// Collect fallback variables from all resolved references.
	for _, erd := range results {
		for _, rd := range erd.Resources {
			collectFallbackVars(rd.Attributes, erd.Definition.Attributes, varSeen, &fallbackVars)
		}
	}

	return fallbackVars
}

// collectFallbackVars walks attributes and collects FallbackVariable entries
// from ResolvedReference values that are variable fallbacks (IsVariable == true)
// and are NOT the standard pingone_environment_id variable.
func collectFallbackVars(attrs map[string]interface{}, defs []schema.AttributeDefinition, seen map[string]bool, out *[]FallbackVariable) {
	for _, attrDef := range defs {
		tName := attrDef.TerraformName
		if tName == "" {
			tName = strings.ToLower(attrDef.Name)
		}

		val, ok := attrs[tName]
		if !ok || val == nil {
			continue
		}

		if ref, ok := val.(ResolvedReference); ok && ref.IsVariable {
			// Skip the standard environment variable — it's always declared.
			if ref.VariableName == "pingone_environment_id" {
				continue
			}
			if !seen[ref.VariableName] {
				seen[ref.VariableName] = true
				*out = append(*out, FallbackVariable{
					Name:         ref.VariableName,
					Type:         "string",
					Description:  fmt.Sprintf("ID of %s resource (excluded from export)", ref.ResourceType),
					ResourceType: ref.ResourceType,
					Default:      ref.OriginalValue,
				})
			}
			continue
		}

		// Recurse into nested structures.
		if attrDef.Type == "object" && len(attrDef.NestedAttributes) > 0 {
			if m, ok := val.(map[string]interface{}); ok {
				collectFallbackVars(m, attrDef.NestedAttributes, seen, out)
			}
		}
		if (attrDef.Type == "list" || attrDef.Type == "set") && len(attrDef.NestedAttributes) > 0 {
			if slice, ok := val.([]interface{}); ok {
				for _, item := range slice {
					if itemMap, ok := item.(map[string]interface{}); ok {
						collectFallbackVars(itemMap, attrDef.NestedAttributes, seen, out)
					}
				}
			}
		}
		if attrDef.Type == "map" && len(attrDef.NestedAttributes) > 0 {
			if m, ok := val.(map[string]interface{}); ok {
				for _, entryVal := range m {
					if entryMap, ok := entryVal.(map[string]interface{}); ok {
						collectFallbackVars(entryMap, attrDef.NestedAttributes, seen, out)
					}
				}
			}
		}
	}
}

// injectEnvironmentIDs ensures every attribute that references pingone_environment
// has the raw environment UUID, even when full reference resolution is skipped.
// This handles definitions where environment_id has no source_path and relies on
// the orchestrator to provide the value.
func (o *ExportOrchestrator) injectEnvironmentIDs(results []*ExportedResourceData, environmentID string) {
	for _, erd := range results {
		for _, rd := range erd.Resources {
			injectEnvIDAttrs(rd.Attributes, erd.Definition.Attributes, environmentID)
		}
	}
}

// injectEnvIDAttrs walks attribute definitions and injects the raw environment
// UUID for any pingone_environment reference attribute that is missing.
func injectEnvIDAttrs(attrs map[string]interface{}, defs []schema.AttributeDefinition, environmentID string) {
	for _, attrDef := range defs {
		tName := attrDef.TerraformName
		if tName == "" {
			tName = strings.ToLower(attrDef.Name)
		}

		if attrDef.ReferencesType == "pingone_environment" {
			if _, ok := attrs[tName]; !ok {
				attrs[tName] = environmentID
			}
		}
	}
}

// resolveAttrs recursively resolves reference attributes in an attribute map
// against the corresponding schema attribute definitions.
func resolveAttrs(attrs map[string]interface{}, defs []schema.AttributeDefinition, g *graph.DependencyGraph, environmentID string, excludedIDs map[string]bool) {
	for _, attrDef := range defs {
		tName := attrDef.TerraformName
		if tName == "" {
			tName = strings.ToLower(attrDef.Name)
		}

		// Top-level reference attribute.
		if attrDef.ReferencesType != "" {
			val, ok := attrs[tName]
			if !ok || val == nil {
				// Attribute missing from processor output (no source_path in definition).
				// For environment references, inject a ResolvedReference using the
				// export context's environment ID.
				if attrDef.ReferencesType == "pingone_environment" && environmentID != "" {
					attrs[tName] = resolveOneReference(attrDef, environmentID, g, excludedIDs)
				}
				continue
			}
			uuid, ok := val.(string)
			if !ok || uuid == "" {
				continue
			}
			attrs[tName] = resolveOneReference(attrDef, uuid, g, excludedIDs)
			continue
		}

		// Nested object — recurse into its attributes.
		if attrDef.Type == "object" && len(attrDef.NestedAttributes) > 0 {
			val, ok := attrs[tName]
			if !ok || val == nil {
				continue
			}
			if m, ok := val.(map[string]interface{}); ok {
				resolveAttrs(m, attrDef.NestedAttributes, g, environmentID, excludedIDs)
			}
			continue
		}

		// Nested map with schema — recurse into each map entry.
		if attrDef.Type == "map" && len(attrDef.NestedAttributes) > 0 {
			val, ok := attrs[tName]
			if !ok || val == nil {
				continue
			}
			if m, ok := val.(map[string]interface{}); ok {
				for _, entryVal := range m {
					if entryMap, ok := entryVal.(map[string]interface{}); ok {
						resolveAttrs(entryMap, attrDef.NestedAttributes, g, environmentID, excludedIDs)
					}
				}
			}
			continue
		}

		// Nested list of objects — recurse into each list item.
		if (attrDef.Type == "list" || attrDef.Type == "set") && len(attrDef.NestedAttributes) > 0 {
			val, ok := attrs[tName]
			if !ok || val == nil {
				continue
			}
			if slice, ok := val.([]interface{}); ok {
				for _, item := range slice {
					if itemMap, ok := item.(map[string]interface{}); ok {
						resolveAttrs(itemMap, attrDef.NestedAttributes, g, environmentID, excludedIDs)
					}
				}
			}
			continue
		}
	}
}

// resolveOneReference resolves a single UUID string to a ResolvedReference.
// Environment references always become variable references. Other references
// attempt graph lookup and fall back to variable references if not found.
// When a resource is found in the graph but its ID is in excludedIDs (removed
// by filtering), a variable reference is produced using the resource label for
// a unique, human-readable variable name.
func resolveOneReference(attrDef schema.AttributeDefinition, uuid string, g *graph.DependencyGraph, excludedIDs map[string]bool) ResolvedReference {
	field := "id"
	if attrDef.ReferenceField != "" {
		field = attrDef.ReferenceField
	}

	// Try graph lookup for resource-to-resource references.
	if g != nil {
		name, err := g.GetReferenceName(attrDef.ReferencesType, uuid)
		if err == nil {
			// If the resource was excluded by filter, produce a variable
			// reference with a label-derived name instead of a resource ref.
			// Exception: pingone_environment always uses the canonical
			// "pingone_environment_id" variable for backward compatibility.
			if excludedIDs[uuid] {
				var varName string
				if attrDef.ReferencesType == "pingone_environment" {
					varName = "pingone_environment_id"
				} else {
					varName = utils.SanitizeVariableName(attrDef.ReferencesType + "_" + name + "_" + field)
				}
				return ResolvedReference{
					IsVariable:    true,
					VariableName:  varName,
					ResourceType:  attrDef.ReferencesType,
					OriginalValue: uuid,
				}
			}
			return ResolvedReference{
				ResourceType:  attrDef.ReferencesType,
				ResourceName:  name,
				Field:         field,
				OriginalValue: uuid,
			}
		}
	}

	// Fallback to variable reference when resource not found in graph.
	varName := attrDef.ReferencesType
	if attrDef.ReferenceField != "" {
		varName = attrDef.ReferencesType + "_" + attrDef.ReferenceField
	}
	return ResolvedReference{
		IsVariable:    true,
		VariableName:  varName,
		OriginalValue: uuid,
	}
}

// resolveCorrelatedReferences performs a second pass over processed attributes
// to resolve reference attributes whose values are non-string (e.g., numeric
// version numbers). These cannot be resolved via UUID graph lookup. Instead,
// they correlate with a sibling or ancestor attribute that already resolved to
// a ResolvedReference with the same ReferencesType.
//
// Example: flow_deploy has flow_id resolved to pingone_davinci_flow.my_flow.id
// and deployed_version (numeric) referencing pingone_davinci_flow with field
// current_version. The correlated resolution produces
// pingone_davinci_flow.my_flow.current_version.
func resolveCorrelatedReferences(attrs map[string]interface{}, defs []schema.AttributeDefinition) {
	resolved := collectResolvedReferences(attrs, defs)
	if len(resolved) == 0 {
		return
	}
	applyCorrelatedReferences(attrs, defs, resolved)
}

// collectResolvedReferences walks an attribute tree and returns all non-variable
// ResolvedReference values indexed by their ResourceType.
func collectResolvedReferences(attrs map[string]interface{}, defs []schema.AttributeDefinition) map[string]ResolvedReference {
	result := make(map[string]ResolvedReference)
	for _, attrDef := range defs {
		tName := attrDef.TerraformName
		if tName == "" {
			tName = strings.ToLower(attrDef.Name)
		}

		val, ok := attrs[tName]
		if !ok || val == nil {
			continue
		}

		if ref, ok := val.(ResolvedReference); ok && !ref.IsVariable {
			result[ref.ResourceType] = ref
			continue
		}

		// Recurse into nested objects.
		if attrDef.Type == "object" && len(attrDef.NestedAttributes) > 0 {
			if m, ok := val.(map[string]interface{}); ok {
				for k, v := range collectResolvedReferences(m, attrDef.NestedAttributes) {
					if _, exists := result[k]; !exists {
						result[k] = v
					}
				}
			}
		}

		// Recurse into nested lists.
		if (attrDef.Type == "list" || attrDef.Type == "set") && len(attrDef.NestedAttributes) > 0 {
			if slice, ok := val.([]interface{}); ok {
				for _, item := range slice {
					if itemMap, ok := item.(map[string]interface{}); ok {
						for k, v := range collectResolvedReferences(itemMap, attrDef.NestedAttributes) {
							if _, exists := result[k]; !exists {
								result[k] = v
							}
						}
					}
				}
			}
		}

		// Recurse into nested maps.
		if attrDef.Type == "map" && len(attrDef.NestedAttributes) > 0 {
			if m, ok := val.(map[string]interface{}); ok {
				for _, entryVal := range m {
					if entryMap, ok := entryVal.(map[string]interface{}); ok {
						for k, v := range collectResolvedReferences(entryMap, attrDef.NestedAttributes) {
							if _, exists := result[k]; !exists {
								result[k] = v
							}
						}
					}
				}
			}
		}
	}
	return result
}

// applyCorrelatedReferences walks an attribute tree and resolves unresolved
// non-string reference attributes by correlating with already-resolved references
// of the same ResourceType.
func applyCorrelatedReferences(attrs map[string]interface{}, defs []schema.AttributeDefinition, resolved map[string]ResolvedReference) {
	for _, attrDef := range defs {
		tName := attrDef.TerraformName
		if tName == "" {
			tName = strings.ToLower(attrDef.Name)
		}

		val, ok := attrs[tName]
		if !ok || val == nil {
			continue
		}

		// Skip already-resolved references.
		if _, ok := val.(ResolvedReference); ok {
			continue
		}

		// Non-string reference attribute — try correlated resolution.
		if attrDef.ReferencesType != "" {
			if _, isStr := val.(string); !isStr {
				if sibling, found := resolved[attrDef.ReferencesType]; found {
					field := "id"
					if attrDef.ReferenceField != "" {
						field = attrDef.ReferenceField
					}
					attrs[tName] = ResolvedReference{
						ResourceType:  sibling.ResourceType,
						ResourceName:  sibling.ResourceName,
						Field:         field,
						OriginalValue: fmt.Sprintf("%v", val),
					}
				}
			}
			continue
		}

		// Recurse into nested objects.
		if attrDef.Type == "object" && len(attrDef.NestedAttributes) > 0 {
			if m, ok := val.(map[string]interface{}); ok {
				applyCorrelatedReferences(m, attrDef.NestedAttributes, resolved)
			}
		}

		// Recurse into nested lists.
		if (attrDef.Type == "list" || attrDef.Type == "set") && len(attrDef.NestedAttributes) > 0 {
			if slice, ok := val.([]interface{}); ok {
				for _, item := range slice {
					if itemMap, ok := item.(map[string]interface{}); ok {
						applyCorrelatedReferences(itemMap, attrDef.NestedAttributes, resolved)
					}
				}
			}
		}

		// Recurse into nested maps.
		if attrDef.Type == "map" && len(attrDef.NestedAttributes) > 0 {
			if m, ok := val.(map[string]interface{}); ok {
				for _, entryVal := range m {
					if entryMap, ok := entryVal.(map[string]interface{}); ok {
						applyCorrelatedReferences(entryMap, attrDef.NestedAttributes, resolved)
					}
				}
			}
		}
	}
}

// orderByDependencies returns definitions in dependency order.
// Resources with no dependencies come first. Resources whose depends_on
// includes another type come after that type.
func (o *ExportOrchestrator) orderByDependencies(defs []*schema.ResourceDefinition) ([]*schema.ResourceDefinition, error) {
	// Build adjacency: type -> types it depends on.
	defMap := make(map[string]*schema.ResourceDefinition)
	depends := make(map[string][]string)
	for _, d := range defs {
		rt := d.Metadata.ResourceType
		defMap[rt] = d
		for _, dep := range d.Dependencies.DependsOn {
			depends[rt] = append(depends[rt], dep.ResourceType)
		}
	}

	// Kahn's algorithm for topological sort.
	inDegree := make(map[string]int)
	for _, d := range defs {
		inDegree[d.Metadata.ResourceType] = 0
	}
	for rt, deps := range depends {
		_ = rt
		for _, dep := range deps {
			if _, ok := defMap[dep]; ok {
				inDegree[rt]++
			}
		}
	}

	// Seed queue with zero-indegree nodes, sorted for determinism.
	var queue []string
	for rt, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, rt)
		}
	}
	sort.Strings(queue)

	var ordered []*schema.ResourceDefinition
	visited := make(map[string]bool)

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if visited[cur] {
			continue
		}
		visited[cur] = true
		ordered = append(ordered, defMap[cur])

		// Decrement in-degree for types that depend on cur.
		for rt, deps := range depends {
			if visited[rt] {
				continue
			}
			for _, dep := range deps {
				if dep == cur {
					inDegree[rt]--
					if inDegree[rt] == 0 {
						queue = append(queue, rt)
					}
				}
			}
		}
		sort.Strings(queue)
	}

	if len(ordered) != len(defs) {
		// Find unprocessed types for error message.
		var unprocessed []string
		for _, d := range defs {
			if !visited[d.Metadata.ResourceType] {
				unprocessed = append(unprocessed, d.Metadata.ResourceType)
			}
		}
		return nil, fmt.Errorf("circular dependency among resource types: %s", strings.Join(unprocessed, ", "))
	}

	return ordered, nil
}

// assignLabels computes and assigns unique Terraform resource labels to each
// resource. For resource types that allow duplicate labels
// (AllowDuplicateLabels), collisions are disambiguated by appending _2, _3, etc.
// For other types, duplicate labels produce an error.
func assignLabels(resources []*ResourceData, def *schema.ResourceDefinition) error {
	type labelInfo struct {
		indices []int
		names   []string
		ids     []string
	}
	seen := make(map[string]*labelInfo)

	// First pass: compute base labels and detect collisions.
	baseLabels := make([]string, len(resources))
	for i, rd := range resources {
		label := terraformLabel(rd, def)
		baseLabels[i] = label
		if info, exists := seen[label]; exists {
			info.indices = append(info.indices, i)
			info.names = append(info.names, rd.Name)
			info.ids = append(info.ids, rd.ID)
		} else {
			seen[label] = &labelInfo{
				indices: []int{i},
				names:   []string{rd.Name},
				ids:     []string{rd.ID},
			}
		}
	}

	// Check for duplicates.
	var duplicates []string
	for label, info := range seen {
		if len(info.indices) <= 1 {
			continue
		}
		if !def.API.AllowDuplicateLabels {
			duplicates = append(duplicates, fmt.Sprintf(
				"label %q produced by %d resources (names: %s, IDs: %s)",
				label, len(info.indices),
				strings.Join(info.names, ", "),
				strings.Join(info.ids, ", "),
			))
		}
	}

	if len(duplicates) > 0 {
		sort.Strings(duplicates)
		return fmt.Errorf("duplicate resource labels:\n  %s", strings.Join(duplicates, "\n  "))
	}

	// Second pass: assign labels, disambiguating duplicates when allowed.
	usage := make(map[string]int)
	for i, rd := range resources {
		label := baseLabels[i]
		count := usage[label]
		usage[label] = count + 1
		if count > 0 {
			// Append suffix to disambiguate: label_2, label_3, ...
			rd.Label = fmt.Sprintf("%s_%d", label, count+1)
		} else {
			rd.Label = label
		}
	}

	return nil
}

// terraformLabel computes the sanitized Terraform resource label for a processed
// resource. This must match the label the HCL formatter produces so that
// graph-based GenerateTerraformReference returns the correct resource reference.
func terraformLabel(rd *ResourceData, def *schema.ResourceDefinition) string {
	if len(def.API.LabelFields) > 0 {
		keys := make([]string, 0, len(def.API.LabelFields))
		for _, field := range def.API.LabelFields {
			if v, ok := rd.Attributes[field]; ok && v != nil {
				if s, ok := v.(string); ok && s != "" {
					keys = append(keys, s)
				}
			}
		}
		if len(keys) > 0 {
			return utils.SanitizeMultiKeyResourceName(keys...)
		}
	}
	if rd.Name != "" {
		return utils.SanitizeResourceName(rd.Name)
	}
	return rd.ID
}

// applyUpstreamExpansion applies filter + expansion when IncludeUpstream is active.
// It builds pre-filter edges, applies the filter to get a seed set, expands via
// graph walk, and returns the expanded results.
func (o *ExportOrchestrator) applyUpstreamExpansion(
	results []*ExportedResourceData,
	depGraph *graph.DependencyGraph,
	resourceFilter *filter.ResourceFilter,
	excludedIDs map[string]bool,
) []*ExportedResourceData {
	// Build edges from all processed (unfiltered) resources so the graph is complete
	// before we apply filtering.
	o.buildPreFilterEdges(results, depGraph)

	// Collect all nodes as seeds by checking which match the filter.
	var seeds []graph.ResourceNode
	for _, erd := range results {
		for _, rd := range erd.Resources {
			address := erd.ResourceType + "." + rd.Label
			if resourceFilter.Allow(address) {
				seeds = append(seeds, graph.ResourceNode{
					ResourceType: erd.ResourceType,
					ID:           rd.ID,
				})
			}
		}
	}

	// If no seeds match, return empty.
	if len(seeds) == 0 {
		return []*ExportedResourceData{}
	}

	// Expand seeds via graph to get all reachable upstream resources.
	expanded := depGraph.WalkDependencies(seeds)

	// Build set of expanded nodes for quick lookup.
	expandedSet := make(map[string]bool) // key: "type:id"
	for _, node := range expanded {
		expandedSet[node.ResourceType+":"+node.ID] = true
	}

	// Check which expanded resources are explicitly excluded.
	// Resources that match explicit exclude patterns stay excluded.
	explicitExcludes := make(map[string]bool)
	for _, erd := range results {
		for _, rd := range erd.Resources {
			address := erd.ResourceType + "." + rd.Label
			if resourceFilter.IsExplicitlyExcluded(address) {
				explicitExcludes[erd.ResourceType+":"+rd.ID] = true
			}
		}
	}

	// Rebuild results keeping resources in expandedSet that aren't explicitly excluded.
	var finalResults []*ExportedResourceData
	for _, erd := range results {
		var kept []*ResourceData
		for _, rd := range erd.Resources {
			key := erd.ResourceType + ":" + rd.ID
			if expandedSet[key] && !explicitExcludes[key] {
				// Keep this resource: it's in the expanded set and not explicitly excluded
				kept = append(kept, rd)
			} else if !expandedSet[key] {
				// Mark as excluded if not in expanded set
				excludedIDs[rd.ID] = true
			} else if explicitExcludes[key] {
				// Explicitly excluded (but might be referenced): mark for fallback variables
				excludedIDs[rd.ID] = true
			}
		}

		// Only include types with resources.
		if len(kept) > 0 {
			finalResults = append(finalResults, &ExportedResourceData{
				ResourceType: erd.ResourceType,
				Definition:   erd.Definition,
				Resources:    kept,
			})
		}
	}

	return finalResults
}

// buildPreFilterEdges scans all processed resources and adds edges to the
// dependency graph based on reference_type declarations in the schema.
// This enables graph-based upstream expansion before applying filters.
func (o *ExportOrchestrator) buildPreFilterEdges(results []*ExportedResourceData, g *graph.DependencyGraph) {
	for _, erd := range results {
		for _, rd := range erd.Resources {
			o.scanEdgesFromAttrs(rd.Attributes, erd.Definition.Attributes, erd.ResourceType, rd.ID, g)
		}
	}
}

// scanEdgesFromAttrs recursively walks an attribute map and registers graph edges
// for any reference_type UUID values found. It mirrors the resolveAttrs pattern.
func (o *ExportOrchestrator) scanEdgesFromAttrs(
	attrs map[string]interface{},
	defs []schema.AttributeDefinition,
	fromType, fromID string,
	g *graph.DependencyGraph,
) {
	for _, attrDef := range defs {
		tName := attrDef.TerraformName
		if tName == "" {
			tName = strings.ToLower(attrDef.Name)
		}

		// Direct reference attribute
		if attrDef.ReferencesType != "" && attrDef.ReferencesType != "pingone_environment" {
			val, ok := attrs[tName]
			if !ok || val == nil {
				continue
			}
			uuid, ok := val.(string)
			if !ok || uuid == "" {
				continue
			}
			// Try to add edge (silently skip if target not in graph)
			_ = g.AddEdge(fromType, fromID, attrDef.ReferencesType, uuid, tName, "")
			continue
		}

		// Nested object
		if attrDef.Type == "object" && len(attrDef.NestedAttributes) > 0 {
			if m, ok := attrs[tName].(map[string]interface{}); ok {
				o.scanEdgesFromAttrs(m, attrDef.NestedAttributes, fromType, fromID, g)
			}
			continue
		}

		// Nested list/set
		if (attrDef.Type == "list" || attrDef.Type == "set") && len(attrDef.NestedAttributes) > 0 {
			if slice, ok := attrs[tName].([]interface{}); ok {
				for _, item := range slice {
					if itemMap, ok := item.(map[string]interface{}); ok {
						o.scanEdgesFromAttrs(itemMap, attrDef.NestedAttributes, fromType, fromID, g)
					}
				}
			}
			continue
		}

		// Nested map
		if attrDef.Type == "map" && len(attrDef.NestedAttributes) > 0 {
			if m, ok := attrs[tName].(map[string]interface{}); ok {
				for _, entryVal := range m {
					if entryMap, ok := entryVal.(map[string]interface{}); ok {
						o.scanEdgesFromAttrs(entryMap, attrDef.NestedAttributes, fromType, fromID, g)
					}
				}
			}
		}
	}
}

// resolveDependsOnResources resolves RuntimeDependsOn entries on each resource
// by looking up the ResourceID in the dependency graph to populate the Label field.
// Entries whose ID is not in the graph retain an empty Label; formatters skip those.
func resolveDependsOnResources(resources []*ResourceData, g *graph.DependencyGraph) {
	if g == nil {
		return
	}
	for _, rd := range resources {
		for i := range rd.DependsOnResources {
			dep := &rd.DependsOnResources[i]
			label, err := g.GetReferenceName(dep.ResourceType, dep.ResourceID)
			if err == nil {
				dep.Label = label
			}
		}
	}
}
