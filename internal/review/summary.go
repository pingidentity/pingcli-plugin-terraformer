package review

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/core"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
)

// SummaryInput carries everything the summary renderer needs.
type SummaryInput struct {
	// Defs maps resource_type to its definition for every reviewed type.
	Defs map[string]*schema.ResourceDefinition

	// AllTypes is the set of types in the review closure's registry — used to
	// mark reference targets as exported vs fallback-variable.
	AllTypes map[string]bool

	// Changes is the detected change set (nil for a full review without a
	// base diff, e.g. explicit --resource-types without detection).
	Changes []TypeChange

	// EmbeddedRules filters to the types under review. Nil means enumerate
	// from the platform registry (the tool populates this; tests pass it).
	EmbeddedRules []core.EmbeddedReferenceRule
}

// RenderSummary renders the definition-driven summary as GitHub-flavored
// markdown, one section per reviewed type followed by the changed-fields
// report when detection ran.
func RenderSummary(in SummaryInput) string {
	var sb strings.Builder

	sb.WriteString("# Resource Review Summary\n\n")
	sb.WriteString("Generated from the resource definitions under review. ")
	sb.WriteString("This is the single source of truth for what the export will emit; the export and snapshot sections of the bundle show it in action.\n\n")

	types := make([]string, 0, len(in.Defs))
	for rt := range in.Defs {
		types = append(types, rt)
	}
	sort.Strings(types)

	for _, rt := range types {
		writeTypeSummary(&sb, rt, in.Defs[rt], in)
	}

	if len(in.Changes) > 0 {
		writeChangesSection(&sb, in.Changes)
	}

	return sb.String()
}

func writeTypeSummary(sb *strings.Builder, rt string, def *schema.ResourceDefinition, in SummaryInput) {
	m := def.Metadata
	fmt.Fprintf(sb, "## %s\n\n", rt)

	// Metadata line.
	enabled := "enabled"
	if m.Enabled != nil && !*m.Enabled {
		enabled = "**disabled** (`enabled: false`)"
	}
	fmt.Fprintf(sb, "%s · %s · version %s · %s\n\n", m.Platform, m.APIType, m.Version, enabled)
	if m.Service != "" {
		fmt.Fprintf(sb, "Requires BOM service `%s` (see #163).\n\n", m.Service)
	}

	writeAPITable(sb, &def.API)
	writeAttributeTable(sb, def.Attributes)
	writeDependencyTables(sb, def, in)
	writeEmbeddedRules(sb, rt, in.EmbeddedRules)
	writeCustomHandlers(sb, def)
	writeVariables(sb, &def.Variables)
}

func writeAPITable(sb *strings.Builder, api *schema.APIDefinition) {
	sb.WriteString("### API surface\n\n")
	sb.WriteString("| Field | Value |\n|---|---|\n")
	fmt.Fprintf(sb, "| sdk_package | `%s` |\n", api.SDKPackage)
	fmt.Fprintf(sb, "| sdk_type | `%s` |\n", api.SDKType)
	fmt.Fprintf(sb, "| list_method | `%s` |\n", api.ListMethod)
	if api.GetMethod != "" {
		fmt.Fprintf(sb, "| get_method | `%s` |\n", api.GetMethod)
	}
	fmt.Fprintf(sb, "| pagination | `%s` |\n", api.PaginationType)
	fmt.Fprintf(sb, "| id_field | `%s` |\n", api.IDField)
	if api.NameField != "" {
		fmt.Fprintf(sb, "| name_field | `%s` |\n", api.NameField)
	}
	if len(api.LabelFields) > 0 {
		fmt.Fprintf(sb, "| label_fields | `%s` |\n", joinStrings(api.LabelFields))
	}
	if len(api.AdditionalIDFields) > 0 {
		fmt.Fprintf(sb, "| additional_id_fields | `%s` |\n", joinStrings(api.AdditionalIDFields))
	}
	if api.AllowDuplicateLabels {
		sb.WriteString("| allow_duplicate_labels | `true` |\n")
	}
	sb.WriteString("\n")
}

// attributeRow is one rendered row of the attribute table.
type attributeRow struct {
	Name      string
	Type      string
	Flags     string
	Refs      string
	Transform string
	Source    string
}

// collectAttrRows flattens an attribute tree into ordered rows, indenting
// nested attribute names.
func collectAttrRows(rows []attributeRow, attrs []schema.AttributeDefinition, indent string) []attributeRow {
	for i := range attrs {
		a := &attrs[i]

		var flags []string
		if a.Required {
			flags = append(flags, "required")
		}
		if a.Computed {
			flags = append(flags, "computed")
		}
		if a.Sensitive {
			flags = append(flags, "sensitive")
		}
		if a.VariableEligible {
			flags = append(flags, "var-eligible")
		}

		transform := a.Transform
		if a.CustomTransform != "" {
			if transform == "" || transform == "custom" {
				transform = "custom:" + a.CustomTransform
			} else {
				transform += "+custom:" + a.CustomTransform
			}
		}
		if a.MaskedSecret != nil {
			transform = joinFlag(transform, "masked_secret")
		}
		if a.OverrideValue != nil {
			transform = joinFlag(transform, fmt.Sprintf("override=%v", a.OverrideValue))
		}

		refs := ""
		if a.ReferencesType != "" {
			refs = "→ `" + a.ReferencesType + "`"
			if a.ReferenceField != "" {
				refs += "." + a.ReferenceField
			}
		}

		rows = append(rows, attributeRow{
			Name:      indent + "`" + a.TerraformName + "`",
			Type:      a.Type,
			Flags:     joinStrings(flags),
			Refs:      refs,
			Transform: transform,
			Source:    a.SourcePath,
		})

		rows = collectAttrRows(rows, a.NestedAttributes, indent+"  ")
	}
	return rows
}

func joinFlag(existing, flag string) string {
	if existing == "" {
		return flag
	}
	return existing + " " + flag
}

func writeAttributeTable(sb *strings.Builder, attrs []schema.AttributeDefinition) {
	rows := collectAttrRows(nil, attrs, "")
	if len(rows) == 0 {
		sb.WriteString("### Attributes\n\n_(none declared)_\n\n")
		return
	}

	sb.WriteString("### Attributes\n\n")
	sb.WriteString("| Attribute | Type | Flags | References | Transform | source_path |\n")
	sb.WriteString("|---|---|---|---|---|---|\n")
	for _, r := range rows {
		fmt.Fprintf(sb, "| %s | %s | %s | %s | %s | `%s` |\n",
			r.Name, r.Type, r.Flags, r.Refs, r.Transform, r.Source)
	}
	sb.WriteString("\n")
}

func writeDependencyTables(sb *strings.Builder, def *schema.ResourceDefinition, in SummaryInput) {
	var lines []string

	for _, rule := range def.Dependencies.DependsOn {
		optional := ""
		if rule.Optional {
			optional = " (optional)"
		}
		lines = append(lines, fmt.Sprintf("- `%s` via `%s`%s", rule.ResourceType, rule.FieldPath, optional))
	}

	// references_type targets with export status.
	targets := map[string]bool{}
	collectRefTargets(def.Attributes, targets)
	sortedTargets := make([]string, 0, len(targets))
	for t := range targets {
		sortedTargets = append(sortedTargets, t)
	}
	sort.Strings(sortedTargets)
	for _, t := range sortedTargets {
		if _, isDependsOn := anyDependsOn(def, t); isDependsOn {
			continue // already listed above
		}
		status := "`fallback-variable` (target not in review set)"
		if in.AllTypes[t] {
			status = "exported (resolved reference)"
		}
		if t == "pingone_environment" {
			status = "variable `pingone_environment_id` (always)"
		}
		lines = append(lines, fmt.Sprintf("- `%s` — %s", t, status))
	}

	if len(lines) == 0 {
		return
	}

	sb.WriteString("### Dependencies\n\n")
	for _, l := range lines {
		sb.WriteString(l + "\n")
	}
	sb.WriteString("\n")
}

func anyDependsOn(def *schema.ResourceDefinition, targetType string) (bool, bool) {
	for _, rule := range def.Dependencies.DependsOn {
		if rule.ResourceType == targetType {
			return true, rule.Optional
		}
	}
	return false, false
}

func collectRefTargets(attrs []schema.AttributeDefinition, out map[string]bool) {
	for i := range attrs {
		a := &attrs[i]
		if a.ReferencesType != "" {
			out[a.ReferencesType] = true
		}
		collectRefTargets(a.NestedAttributes, out)
	}
}

func writeEmbeddedRules(sb *strings.Builder, rt string, rules []core.EmbeddedReferenceRule) {
	var mine []core.EmbeddedReferenceRule
	for _, r := range rules {
		if r.ResourceType == rt {
			mine = append(mine, r)
		}
	}
	if len(mine) == 0 {
		return
	}

	sb.WriteString("### Embedded reference rules\n\n")
	sb.WriteString("| Attribute path | Target | Key path | Strategy | Variable prefix | Notes |\n")
	sb.WriteString("|---|---|---|---|---|---|\n")
	for _, r := range mine {
		notes := ""
		if r.PlainStringPattern != "" {
			notes = "plain-string mode"
		}
		if r.UnwrapMode != "" {
			notes = joinFlag(notes, "unwrap="+r.UnwrapMode)
		}
		if r.PreconditionKeyPath != "" {
			notes = joinFlag(notes, fmt.Sprintf("precondition %s==%s", r.PreconditionKeyPath, r.PreconditionValue))
		}
		fmt.Fprintf(sb, "| `%s` | `%s` | `%s` | %s | `%s` | %s |\n",
			r.AttributePath, r.TargetResourceType, r.JSONKeyPath, r.Strategy, r.VariablePrefix, notes)
	}
	sb.WriteString("\n")
}

func writeCustomHandlers(sb *strings.Builder, def *schema.ResourceDefinition) {
	var lines []string
	if def.CustomHandlers != nil {
		h := def.CustomHandlers
		if h.HCLGenerator != "" {
			lines = append(lines, fmt.Sprintf("- HCL generator: `%s`", h.HCLGenerator))
		}
		if h.Transformer != "" {
			lines = append(lines, fmt.Sprintf("- Transformer: `%s`", h.Transformer))
		}
		if h.Validator != "" {
			lines = append(lines, fmt.Sprintf("- Validator: `%s`", h.Validator))
		}
	}
	collectCustomTransforms(def.Attributes, &lines)
	if len(lines) == 0 {
		return
	}

	sb.WriteString("### Custom code paths\n\n")
	for _, l := range lines {
		sb.WriteString(l + "\n")
	}
	sb.WriteString("\n_Implementations live in `internal/platform/pingone/` (dispatched by name at runtime)._\n\n")
}

func collectCustomTransforms(attrs []schema.AttributeDefinition, out *[]string) {
	seen := map[string]bool{}
	for i := range attrs {
		a := &attrs[i]
		if a.CustomTransform != "" && !seen[a.CustomTransform] {
			seen[a.CustomTransform] = true
			*out = append(*out, fmt.Sprintf("- Custom transform `%s` (attribute `%s`)", a.CustomTransform, a.TerraformName))
		}
		collectCustomTransforms(a.NestedAttributes, out)
	}
}

func writeVariables(sb *strings.Builder, v *schema.VariableDefinition) {
	if v.VariablePrefix == "" && len(v.EligibleAttributes) == 0 {
		return
	}
	sb.WriteString("### Variables\n\n")
	if v.VariablePrefix != "" {
		fmt.Fprintf(sb, "Variable prefix: `%s`\n\n", v.VariablePrefix)
	}
	if len(v.EligibleAttributes) > 0 {
		sb.WriteString("| Attribute path | Prefix | Secret | Default | Description |\n|---|---|---|---|---|\n")
		for _, r := range v.EligibleAttributes {
			secret := ""
			if r.IsSecret {
				secret = "yes"
			}
			def := ""
			if r.DefaultValue != nil {
				def = fmt.Sprintf("%v", r.DefaultValue)
			}
			fmt.Fprintf(sb, "| `%s` | `%s` | %s | %s | %s |\n",
				r.AttributePath, r.VariablePrefix, secret, def, r.Description)
		}
		sb.WriteString("\n")
	}
}

func writeChangesSection(sb *strings.Builder, changes []TypeChange) {
	sb.WriteString("---\n\n# Changed fields (base → this PR)\n\n")
	for _, tc := range changes {
		label := "`" + tc.ResourceType + "`"
		switch {
		case tc.IsNew:
			label += " — **new resource**"
		case tc.Removed:
			label += " — **removed**"
		}
		extra := ""
		if tc.HandlerChanged {
			extra = " · handler changed"
		}
		fmt.Fprintf(sb, "## %s%s\n\n", label, extra)

		if tc.DefinitionPath != "" {
			fmt.Fprintf(sb, "_Definition: `%s`_\n\n", tc.DefinitionPath)
		}

		if len(tc.FieldChanges) == 0 {
			if tc.IsNew {
				sb.WriteString("Entire definition is new — see the summary section above.\n\n")
			} else {
				sb.WriteString("No definition-level changes detected.\n\n")
			}
			continue
		}

		sb.WriteString("| Section | Attribute | Change | Field | Base | This PR |\n")
		sb.WriteString("|---|---|---|---|---|---|\n")
		for _, fc := range tc.FieldChanges {
			attr := fc.Attr
			if attr != "" {
				attr = "`" + attr + "`"
			}
			fmt.Fprintf(sb, "| %s | %s | %s | `%s` | `%s` | `%s` |\n",
				fc.Section, attr, fc.Kind, fc.Field, fc.Base, fc.PR)
		}
		sb.WriteString("\n")
	}
}
