package review

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
)

// diffSections are the definition-level comparison targets.
const (
	sectionMetadata            = "metadata"
	sectionAPI                 = "api"
	sectionAttributes          = "attributes"
	sectionDependencies        = "dependencies"
	sectionVariables           = "variables"
	sectionCustomHandlers      = "custom_handlers"
	sectionConditionalDefaults = "conditional_defaults"
)

// DiffDefinitions compares two parsed definitions and returns the semantic
// changes between them. Attribute-level changes are keyed by terraform_name
// and recurse into nested_attributes; section-level changes are reported
// per YAML field. The result is deterministic (stable order) so reports are
// diffable themselves.
func DiffDefinitions(base, pr *schema.ResourceDefinition) []FieldChange {
	var changes []FieldChange

	changes = append(changes, diffMetadata(&base.Metadata, &pr.Metadata)...)
	changes = append(changes, diffAPI(&base.API, &pr.API)...)
	changes = append(changes, diffAttributes(sectionAttributes, base.Attributes, pr.Attributes)...)
	changes = append(changes, diffDependencies(&base.Dependencies, &pr.Dependencies)...)
	changes = append(changes, diffVariables(&base.Variables, &pr.Variables)...)
	changes = append(changes, diffCustomHandlers(base.CustomHandlers, pr.CustomHandlers)...)
	changes = append(changes, diffConditionalDefaults(base.ConditionalDefaults, pr.ConditionalDefaults)...)

	return changes
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func addChange(changes []FieldChange, section, attr, kind, field, base, pr string) []FieldChange {
	return append(changes, FieldChange{
		Section: section,
		Attr:    attr,
		Kind:    kind,
		Field:   field,
		Base:    base,
		PR:      pr,
	})
}

// diffScalars compares a set of scalar fields by name. Field values are
// pre-formatted strings; empty-string vs missing distinction is handled by
// the caller formatting its values consistently ("" == unset).
func diffScalars(changes []FieldChange, section, attr string, base, pr map[string]string) []FieldChange {
	fields := make([]string, 0, len(base))
	for f := range base {
		fields = append(fields, f)
	}
	sort.Strings(fields)
	for _, f := range fields {
		b, p := base[f], pr[f]
		switch {
		case b == "" && p == "":
			// both unset
		case b == "":
			changes = addChange(changes, section, attr, "added", f, "", p)
		case p == "":
			changes = addChange(changes, section, attr, "removed", f, b, "")
		case b != p:
			changes = addChange(changes, section, attr, "changed", f, b, p)
		}
	}
	return changes
}

func diffMetadata(base, pr *schema.ResourceMetadata) []FieldChange {
	if base.ResourceType != pr.ResourceType {
		return []FieldChange{{
			Section: sectionMetadata,
			Kind:    "changed",
			Field:   "resource_type",
			Base:    base.ResourceType,
			PR:      pr.ResourceType,
		}}
	}
	return diffScalars(nil, sectionMetadata, "", map[string]string{
		"platform":   base.Platform,
		"service":    base.Service,
		"api_type":   base.APIType,
		"name":       base.Name,
		"short_name": base.ShortName,
		"version":    base.Version,
		"enabled":    triState(base.Enabled),
	}, map[string]string{
		"platform":   pr.Platform,
		"service":    pr.Service,
		"api_type":   pr.APIType,
		"name":       pr.Name,
		"short_name": pr.ShortName,
		"version":    pr.Version,
		"enabled":    triState(pr.Enabled),
	})
}

// triState renders the tri-state *bool as "" (unset), "false", or "true".
func triState(b *bool) string {
	if b == nil {
		return ""
	}
	return boolStr(*b)
}

func diffAPI(base, pr *schema.APIDefinition) []FieldChange {
	return diffScalars(nil, sectionAPI, "", map[string]string{
		"sdk_package":            base.SDKPackage,
		"sdk_type":               base.SDKType,
		"list_method":            base.ListMethod,
		"get_method":             base.GetMethod,
		"id_field":               base.IDField,
		"name_field":             base.NameField,
		"label_fields":           joinStrings(base.LabelFields),
		"pagination_type":        base.PaginationType,
		"additional_id_fields":   joinStrings(base.AdditionalIDFields),
		"allow_duplicate_labels": boolStr(base.AllowDuplicateLabels),
	}, map[string]string{
		"sdk_package":            pr.SDKPackage,
		"sdk_type":               pr.SDKType,
		"list_method":            pr.ListMethod,
		"get_method":             pr.GetMethod,
		"id_field":               pr.IDField,
		"name_field":             pr.NameField,
		"label_fields":           joinStrings(pr.LabelFields),
		"pagination_type":        pr.PaginationType,
		"additional_id_fields":   joinStrings(pr.AdditionalIDFields),
		"allow_duplicate_labels": boolStr(pr.AllowDuplicateLabels),
	})
}

func joinStrings(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, s := range ss {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(s)
	}
	return sb.String()
}

// attributeKey identifies an attribute within a (possibly nested) list.
type attributeKey struct {
	tfName  string
	parents string
}

func attrKey(parents string, a *schema.AttributeDefinition) attributeKey {
	return attributeKey{tfName: a.TerraformName, parents: parents}
}

func attrPath(parents, tfName string) string {
	if parents == "" {
		return tfName
	}
	return parents + "." + tfName
}

func attrMap(parents string, attrs []schema.AttributeDefinition) map[attributeKey]*schema.AttributeDefinition {
	m := make(map[attributeKey]*schema.AttributeDefinition, len(attrs))
	for i := range attrs {
		m[attrKey(parents, &attrs[i])] = &attrs[i]
	}
	return m
}

func diffAttributes(section string, base, pr []schema.AttributeDefinition) []FieldChange {
	var changes []FieldChange
	baseMap := attrMap("", base)
	prMap := attrMap("", pr)

	keys := make([]attributeKey, 0, len(baseMap))
	for k := range baseMap {
		keys = append(keys, k)
	}
	for k := range prMap {
		if _, ok := baseMap[k]; !ok {
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].parents != keys[j].parents {
			return keys[i].parents < keys[j].parents
		}
		return keys[i].tfName < keys[j].tfName
	})

	for _, k := range keys {
		path := attrPath(k.parents, k.tfName)
		b, inBase := baseMap[k]
		p, inPR := prMap[k]
		switch {
		case !inBase:
			changes = addChange(changes, section, path, "added", "", "", "")
			changes = append(changes, diffAttributeFields(path, &schema.AttributeDefinition{}, p, false)...)
		case !inPR:
			changes = addChange(changes, section, path, "removed", "", "", "")
		default:
			changes = append(changes, diffAttributeFields(path, b, p, true)...)
		}
	}
	return changes
}

// diffAttributeFields compares the scalar fields of one attribute pair and
// recurses into nested attributes. When inBoth is false the base side is a
// zero value and only "added" entries are emitted.
func diffAttributeFields(path string, base, pr *schema.AttributeDefinition, inBoth bool) []FieldChange {
	var changes []FieldChange

	baseScalars := map[string]string{
		"type":              base.Type,
		"source_path":       base.SourcePath,
		"required":          boolStr(base.Required),
		"computed":          boolStr(base.Computed),
		"sensitive":         boolStr(base.Sensitive),
		"variable_eligible": boolStr(base.VariableEligible),
		"references_type":   base.ReferencesType,
		"reference_field":   base.ReferenceField,
		"transform":         base.Transform,
		"custom_transform":  base.CustomTransform,
		"map_key_path":      base.MapKeyPath,
		"override_value":    fmt.Sprintf("%v", base.OverrideValue),
		"nil_value":         base.NilValue,
	}
	prScalars := map[string]string{
		"type":              pr.Type,
		"source_path":       pr.SourcePath,
		"required":          boolStr(pr.Required),
		"computed":          boolStr(pr.Computed),
		"sensitive":         boolStr(pr.Sensitive),
		"variable_eligible": boolStr(pr.VariableEligible),
		"references_type":   pr.ReferencesType,
		"reference_field":   pr.ReferenceField,
		"transform":         pr.Transform,
		"custom_transform":  pr.CustomTransform,
		"map_key_path":      pr.MapKeyPath,
		"override_value":    fmt.Sprintf("%v", pr.OverrideValue),
		"nil_value":         pr.NilValue,
	}

	if inBoth {
		changes = diffScalars(changes, sectionAttributes, path, baseScalars, prScalars)
	} else {
		// For added attributes emit every set field as part of the addition.
		fields := make([]string, 0, len(prScalars))
		for f, v := range prScalars {
			if v != "" && v != "false" {
				fields = append(fields, f)
			}
		}
		sort.Strings(fields)
		for _, f := range fields {
			changes = addChange(changes, sectionAttributes, path, "added", f, "", prScalars[f])
		}
	}

	// Recurse into nested attributes.
	changes = append(changes, diffAttributesAt(path, base.NestedAttributes, pr.NestedAttributes)...)
	return changes
}

// diffAttributesAt diffs nested attribute lists under a parent path.
func diffAttributesAt(parents string, base, pr []schema.AttributeDefinition) []FieldChange {
	if len(base) == 0 && len(pr) == 0 {
		return nil
	}
	baseMap := attrMap(parents, base)
	prMap := attrMap(parents, pr)

	var changes []FieldChange
	keys := make([]attributeKey, 0, len(baseMap))
	for k := range baseMap {
		keys = append(keys, k)
	}
	for k := range prMap {
		if _, ok := baseMap[k]; !ok {
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].parents != keys[j].parents {
			return keys[i].parents < keys[j].parents
		}
		return keys[i].tfName < keys[j].tfName
	})

	for _, k := range keys {
		path := attrPath(k.parents, k.tfName)
		b, inBase := baseMap[k]
		p, inPR := prMap[k]
		switch {
		case !inBase:
			changes = addChange(changes, sectionAttributes, path, "added", "", "", "")
			changes = append(changes, diffAttributeFields(path, &schema.AttributeDefinition{}, p, false)...)
		case !inPR:
			changes = addChange(changes, sectionAttributes, path, "removed", "", "", "")
		default:
			changes = append(changes, diffAttributeFields(path, b, p, true)...)
		}
	}
	return changes
}

func diffDependencies(base, pr *schema.DependencyDefinition) []FieldChange {
	var changes []FieldChange
	if base.ImportIDFormat != pr.ImportIDFormat {
		changes = addChange(changes, sectionDependencies, "", "changed", "import_id_format", base.ImportIDFormat, pr.ImportIDFormat)
	}
	changes = append(changes, diffDependencyRules("depends_on", base.DependsOn, pr.DependsOn)...)
	return changes
}

// dependencyRuleKey identifies a depends_on rule by target type + field path.
func dependencyRuleKey(r *schema.DependencyRule) string {
	return r.ResourceType + "|" + r.FieldPath + "|" + r.LookupField
}

func diffDependencyRules(field string, base, pr []schema.DependencyRule) []FieldChange {
	var changes []FieldChange
	baseMap := make(map[string]*schema.DependencyRule, len(base))
	for i := range base {
		baseMap[dependencyRuleKey(&base[i])] = &base[i]
	}
	prMap := make(map[string]*schema.DependencyRule, len(pr))
	for i := range pr {
		prMap[dependencyRuleKey(&pr[i])] = &pr[i]
	}

	keys := make([]string, 0, len(baseMap))
	for k := range baseMap {
		keys = append(keys, k)
	}
	for k := range prMap {
		if _, ok := baseMap[k]; !ok {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	for _, k := range keys {
		b, inBase := baseMap[k]
		p, inPR := prMap[k]
		switch {
		case !inBase:
			changes = addChange(changes, sectionDependencies, p.ResourceType, "added", field, "", formatDependencyRule(p))
		case !inPR:
			changes = addChange(changes, sectionDependencies, b.ResourceType, "removed", field, formatDependencyRule(b), "")
		default:
			if b.Optional != p.Optional || b.ReferenceFormat != p.ReferenceFormat {
				changes = addChange(changes, sectionDependencies, p.ResourceType, "changed", field, formatDependencyRule(b), formatDependencyRule(p))
			}
		}
	}
	return changes
}

func formatDependencyRule(r *schema.DependencyRule) string {
	out := r.FieldPath
	if r.LookupField != "" {
		out += " lookup=" + r.LookupField
	}
	if r.ReferenceFormat != "" {
		out += " fmt=" + r.ReferenceFormat
	}
	if r.Optional {
		out += " optional"
	}
	return out
}

func diffVariables(base, pr *schema.VariableDefinition) []FieldChange {
	var changes []FieldChange
	if base.VariablePrefix != pr.VariablePrefix {
		changes = addChange(changes, sectionVariables, "", "changed", "variable_prefix", base.VariablePrefix, pr.VariablePrefix)
	}
	changes = append(changes, diffVariableRules("eligible_attributes", base.EligibleAttributes, pr.EligibleAttributes)...)
	return changes
}

func variableRuleKey(r *schema.VariableRule) string {
	return r.AttributePath + "|" + r.VariablePrefix
}

func diffVariableRules(field string, base, pr []schema.VariableRule) []FieldChange {
	var changes []FieldChange
	baseMap := make(map[string]*schema.VariableRule, len(base))
	for i := range base {
		baseMap[variableRuleKey(&base[i])] = &base[i]
	}
	prMap := make(map[string]*schema.VariableRule, len(pr))
	for i := range pr {
		prMap[variableRuleKey(&pr[i])] = &pr[i]
	}

	keys := make([]string, 0, len(baseMap))
	for k := range baseMap {
		keys = append(keys, k)
	}
	for k := range prMap {
		if _, ok := baseMap[k]; !ok {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	for _, k := range keys {
		b, inBase := baseMap[k]
		p, inPR := prMap[k]
		switch {
		case !inBase:
			changes = addChange(changes, sectionVariables, p.AttributePath, "added", field, "", formatVariableRule(p))
		case !inPR:
			changes = addChange(changes, sectionVariables, b.AttributePath, "removed", field, formatVariableRule(b), "")
		default:
			if b.IsSecret != p.IsSecret || fmt.Sprintf("%v", b.DefaultValue) != fmt.Sprintf("%v", p.DefaultValue) || b.Description != p.Description {
				changes = addChange(changes, sectionVariables, p.AttributePath, "changed", field, formatVariableRule(b), formatVariableRule(p))
			}
		}
	}
	return changes
}

func formatVariableRule(r *schema.VariableRule) string {
	out := r.VariablePrefix
	if r.IsSecret {
		out += " secret"
	}
	if r.Description != "" {
		out += " desc=" + r.Description
	}
	if r.DefaultValue != nil {
		out += fmt.Sprintf(" default=%v", r.DefaultValue)
	}
	return out
}

func diffCustomHandlers(base, pr *schema.CustomHandlerDefinition) []FieldChange {
	if base == nil && pr == nil {
		return nil
	}
	if base == nil {
		return []FieldChange{{Section: sectionCustomHandlers, Kind: "added", PR: formatCustomHandlers(pr)}}
	}
	if pr == nil {
		return []FieldChange{{Section: sectionCustomHandlers, Kind: "removed", Base: formatCustomHandlers(base)}}
	}
	if *base != *pr {
		return []FieldChange{{Section: sectionCustomHandlers, Kind: "changed", Base: formatCustomHandlers(base), PR: formatCustomHandlers(pr)}}
	}
	return nil
}

func formatCustomHandlers(h *schema.CustomHandlerDefinition) string {
	out := ""
	if h.HCLGenerator != "" {
		out += "hcl=" + h.HCLGenerator + " "
	}
	if h.Transformer != "" {
		out += "transform=" + h.Transformer + " "
	}
	if h.Validator != "" {
		out += "validate=" + h.Validator
	}
	return out
}

func diffConditionalDefaults(base, pr []schema.ConditionalDefault) []FieldChange {
	if len(base) == 0 && len(pr) == 0 {
		return nil
	}
	baseStr := fmt.Sprintf("%v", base)
	prStr := fmt.Sprintf("%v", pr)
	if baseStr != prStr {
		return []FieldChange{{Section: sectionConditionalDefaults, Kind: "changed", Base: baseStr, PR: prStr}}
	}
	return nil
}
