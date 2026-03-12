package core

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/clients"
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
}

// ProgressFunc is called by the orchestrator to report status.
// message is a human-readable status update.
type ProgressFunc func(message string)

// ExportOrchestrator coordinates the end-to-end schema-driven export pipeline.
type ExportOrchestrator struct {
	registry   *schema.Registry
	processor  *Processor
	client     clients.APIClient
	progressFn ProgressFunc
}

// OrchestratorOption configures optional ExportOrchestrator behaviour.
type OrchestratorOption func(*ExportOrchestrator)

// WithProgressFunc sets a progress reporting callback.
func WithProgressFunc(fn ProgressFunc) OrchestratorOption {
	return func(o *ExportOrchestrator) {
		o.progressFn = fn
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

		// Populate graph nodes with the assigned labels.
		for _, rd := range processed {
			depGraph.AddResource(rd.ResourceType, rd.ID, rd.Label)
		}

		o.progress(fmt.Sprintf("✓ Processed %d %s resources", len(processed), def.Metadata.ShortName))

		results = append(results, &ExportedResourceData{
			ResourceType: resourceType,
			Definition:   def,
			Resources:    processed,
		})
	}

	// 4. Resolve cross-resource references in all processed attributes.
	// This replaces raw UUID strings with ResolvedReference values so that
	// formatters can render them without needing graph access.
	if !opts.SkipDependencies {
		o.resolveReferences(results, depGraph, opts.EnvironmentID)
	}

	// 5. Validate graph.
	if err := depGraph.Validate(); err != nil {
		o.progress(fmt.Sprintf("Warning: dependency graph validation: %v", err))
	}

	o.progress(fmt.Sprintf("✓ Export complete — %d resource types, %d total resources", len(results), o.totalResources(results)))

	return &ExportResult{
		ResourcesByType: results,
		Graph:           depGraph,
		EnvironmentID:   opts.EnvironmentID,
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

// resolveReferences walks all processed resources and replaces raw UUID strings
// with ResolvedReference values for any attribute that declares references_type
// in the schema. This covers:
//   - Top-level attributes (e.g. environment_id, connection_id)
//   - Nested object attributes (e.g. config.flow_ref)
//
// Environment references always resolve to variable references.
// Other references attempt graph lookup; if unknown, fall back to variable references.
func (o *ExportOrchestrator) resolveReferences(results []*ExportedResourceData, g *graph.DependencyGraph, environmentID string) {
	for _, erd := range results {
		for _, rd := range erd.Resources {
			resolveAttrs(rd.Attributes, erd.Definition.Attributes, g, environmentID)
			resolveCorrelatedReferences(rd.Attributes, erd.Definition.Attributes)
		}
	}
}

// resolveAttrs recursively resolves reference attributes in an attribute map
// against the corresponding schema attribute definitions.
func resolveAttrs(attrs map[string]interface{}, defs []schema.AttributeDefinition, g *graph.DependencyGraph, environmentID string) {
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
					attrs[tName] = resolveOneReference(attrDef, environmentID, g)
				}
				continue
			}
			uuid, ok := val.(string)
			if !ok || uuid == "" {
				continue
			}
			attrs[tName] = resolveOneReference(attrDef, uuid, g)
			continue
		}

		// Nested object — recurse into its attributes.
		if attrDef.Type == "object" && len(attrDef.NestedAttributes) > 0 {
			val, ok := attrs[tName]
			if !ok || val == nil {
				continue
			}
			if m, ok := val.(map[string]interface{}); ok {
				resolveAttrs(m, attrDef.NestedAttributes, g, environmentID)
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
						resolveAttrs(entryMap, attrDef.NestedAttributes, g, environmentID)
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
						resolveAttrs(itemMap, attrDef.NestedAttributes, g, environmentID)
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
func resolveOneReference(attrDef schema.AttributeDefinition, uuid string, g *graph.DependencyGraph) ResolvedReference {
	field := "id"
	if attrDef.ReferenceField != "" {
		field = attrDef.ReferenceField
	}

	varName := attrDef.ReferencesType
	if attrDef.ReferenceField != "" {
		varName = attrDef.ReferencesType + "_" + attrDef.ReferenceField
	}

	// Try graph lookup for resource-to-resource references.
	if g != nil {
		name, err := g.GetReferenceName(attrDef.ReferencesType, uuid)
		if err == nil {
			return ResolvedReference{
				ResourceType:  attrDef.ReferencesType,
				ResourceName:  name,
				Field:         field,
				OriginalValue: uuid,
			}
		}
	}

	// Fallback to variable reference when resource not found in graph.
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
