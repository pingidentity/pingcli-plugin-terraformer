package schema

// ResourceDefinition represents a complete resource definition loaded from YAML
type ResourceDefinition struct {
	Metadata            ResourceMetadata         `yaml:"metadata"`
	API                 APIDefinition            `yaml:"api"`
	Attributes          []AttributeDefinition    `yaml:"attributes"`
	Dependencies        DependencyDefinition     `yaml:"dependencies"`
	Variables           VariableDefinition       `yaml:"variables"`
	CustomHandlers      *CustomHandlerDefinition `yaml:"custom_handlers,omitempty"`
	ConditionalDefaults []ConditionalDefault     `yaml:"conditional_defaults,omitempty"`
}

// ResourceMetadata contains resource identification information
type ResourceMetadata struct {
	Platform     string `yaml:"platform"`
	Service      string `yaml:"service,omitempty"`
	ResourceType string `yaml:"resource_type"`
	APIType      string `yaml:"api_type"`
	Name         string `yaml:"name"`
	ShortName    string `yaml:"short_name"`
	Version      string `yaml:"version"`
	// Enabled is a tri-state flag: nil = enabled by default, false = disabled, true = explicitly enabled.
	Enabled *bool `yaml:"enabled,omitempty"`
}

// APIDefinition defines API interaction configuration
type APIDefinition struct {
	SDKPackage string `yaml:"sdk_package"`
	SDKType    string `yaml:"sdk_type"`
	ListMethod string `yaml:"list_method"`
	GetMethod  string `yaml:"get_method,omitempty"`
	IDField    string `yaml:"id_field"`
	NameField  string `yaml:"name_field"`
	// LabelFields specifies the ordered list of terraform attribute names whose
	// values are combined via SanitizeMultiKeyResourceName to produce the HCL
	// resource label.  When empty the formatter falls back to SanitizeName(data.Name).
	LabelFields        []string `yaml:"label_fields,omitempty"`
	PaginationType     string   `yaml:"pagination_type"`
	AdditionalIDFields []string `yaml:"additional_id_fields,omitempty"`
	// AllowDuplicateLabels skips the unique label validation for resource types
	// where the upstream system permits duplicate names (e.g., flow policies).
	// When true, the orchestrator will not error on duplicate Terraform labels.
	AllowDuplicateLabels bool `yaml:"allow_duplicate_labels,omitempty"`
}

// AttributeDefinition defines a single resource attribute
type AttributeDefinition struct {
	Name             string                `yaml:"name"`
	TerraformName    string                `yaml:"terraform_name"`
	Type             string                `yaml:"type"`
	SourcePath       string                `yaml:"source_path,omitempty"`
	Required         bool                  `yaml:"required"`
	Computed         bool                  `yaml:"computed"`
	Sensitive        bool                  `yaml:"sensitive"`
	VariableEligible bool                  `yaml:"variable_eligible"`
	VariableDefault  interface{}           `yaml:"variable_default,omitempty"`
	ReferencesType   string                `yaml:"references_type,omitempty"`
	ReferenceField   string                `yaml:"reference_field,omitempty"`
	Transform        string                `yaml:"transform,omitempty"`
	CustomTransform  string                `yaml:"custom_transform,omitempty"`
	NestedAttributes []AttributeDefinition `yaml:"nested_attributes,omitempty"`
	Lifecycle        *LifecycleConfig      `yaml:"lifecycle,omitempty"`

	// MapKeyPath specifies the sub-field path used to key slice elements when
	// converting an API slice to a Terraform map. Only used when Type == "map"
	// and the source value is a slice. Dot-notation for nested paths (e.g. "Data.Id").
	MapKeyPath string `yaml:"map_key_path,omitempty"`

	// ValueMap maps API values to Terraform-accepted equivalents. Used with
	// the "value_map" transform.
	ValueMap map[string]string `yaml:"value_map,omitempty"`

	// ValueMapDefault is the fallback value when the API value is not found in
	// ValueMap. If empty, the original value is preserved.
	ValueMapDefault string `yaml:"value_map_default,omitempty"`

	// MaskedSecret configures masked secret detection and variable substitution.
	// Used with the "masked_secret" transform.
	MaskedSecret *MaskedSecretConfig `yaml:"masked_secret,omitempty"`

	// TypeDiscriminatedBlock configuration for type_discriminated_block attributes.
	// When Type == "type_discriminated_block", the processor uses this config to
	// map the runtime Go type of the source value to a Terraform block key.
	TypeDiscriminatedBlock *TypeDiscriminatedBlockConfig `yaml:"type_discriminated_block,omitempty"`

	// OverrideValue forces this attribute to the given constant value regardless
	// of the API response. The value bypasses extraction, transforms, and all
	// other processing. Useful when the Terraform provider expects a fixed
	// sentinel (e.g. version = -1) that differs from what the API returns.
	OverrideValue interface{} `yaml:"override_value,omitempty"`

	// NilValue controls how the processor handles nil values for this attribute.
	// Options: "keep_empty" (emit empty string for nil) or "omit" (skip, current behavior).
	// When not specified, defaults to "omit" for backward compatibility.
	NilValue string `yaml:"nil_value,omitempty"`
}

// DependencyDefinition defines resource relationship configuration
type DependencyDefinition struct {
	DependsOn      []DependencyRule `yaml:"depends_on,omitempty"`
	ProvidedBy     string           `yaml:"provided_by,omitempty"`
	ImportIDFormat string           `yaml:"import_id_format"`
}

// DependencyRule defines a single dependency relationship
type DependencyRule struct {
	ResourceType    string `yaml:"resource_type"`
	FieldPath       string `yaml:"field_path"`
	LookupField     string `yaml:"lookup_field"`
	ReferenceFormat string `yaml:"reference_format"`
	Optional        bool   `yaml:"optional"`
}

// VariableDefinition defines variable extraction configuration
type VariableDefinition struct {
	VariablePrefix     string         `yaml:"variable_prefix,omitempty"`
	EligibleAttributes []VariableRule `yaml:"eligible_attributes,omitempty"`
}

// VariableRule defines a single variable extraction rule
type VariableRule struct {
	AttributePath  string      `yaml:"attribute_path"`
	VariablePrefix string      `yaml:"variable_prefix"`
	IsSecret       bool        `yaml:"is_secret"`
	DefaultValue   interface{} `yaml:"default_value,omitempty"`
	Description    string      `yaml:"description,omitempty"`
}

// CustomHandlerDefinition defines optional custom processing
type CustomHandlerDefinition struct {
	HCLGenerator string `yaml:"hcl_generator,omitempty"`
	Transformer  string `yaml:"transformer,omitempty"`
	Validator    string `yaml:"validator,omitempty"`
}

// LifecycleConfig defines Terraform lifecycle block configuration
type LifecycleConfig struct {
	IgnoreChanges []string `yaml:"ignore_changes,omitempty"`
}

// TypeDiscriminatedBlockConfig declares how to map a runtime-typed API value
// to a Terraform block whose key depends on the Go type of the value.
//
// Example: DaVinci variable "value" block — the Terraform provider expects
// exactly ONE key inside the block: string, bool, float32, or json_object.
// Which key to use depends on the Go runtime type of the Variable.Value field.
type TypeDiscriminatedBlockConfig struct {
	// TypeKeyMap maps Go runtime type names to Terraform block key names.
	// Keys: string, bool, float64, int, map, slice
	// Values: the Terraform attribute name inside the block
	TypeKeyMap map[string]string `yaml:"type_key_map"`

	// JSONEncodeKeys lists block keys whose values should be json.Marshal'd
	// and emitted as RawHCLValue (unquoted HCL).
	JSONEncodeKeys []string `yaml:"json_encode_keys,omitempty"`

	// SkipConditions lists conditions under which the entire block is suppressed.
	// All conditions are OR'd — if any matches, the block is skipped.
	SkipConditions []SkipCondition `yaml:"skip_conditions,omitempty"`
}

// SkipCondition defines a condition for suppressing an attribute.
type SkipCondition struct {
	// SourceField is the apiData struct field to check (read via reflection).
	SourceField string `yaml:"source_field"`

	// Equals is the value that triggers suppression (case-insensitive string match).
	Equals string `yaml:"equals"`
}

// MaskedSecretConfig configures sentinel detection and variable substitution
// for API fields that return masked secret values.
type MaskedSecretConfig struct {
	// Sentinel is the masked value returned by the API (e.g. "******").
	Sentinel string `yaml:"sentinel"`

	// VariableNameParts defines ordered segments used to build the Terraform
	// variable name. Each part specifies a source ("resource_name",
	// "attribute_key", or "literal") and an optional value for literals.
	VariableNameParts []VariableNamePart `yaml:"variable_name_parts"`
}

// VariableNamePart defines a single segment of a generated variable name.
type VariableNamePart struct {
	// Source determines where the value comes from:
	//   "resource_name" — ResourceData.Name
	//   "attribute_key" — the current map key or attribute name being processed
	//   "literal"       — uses the Value field directly
	Source string `yaml:"source"`

	// Value is used when Source == "literal".
	Value string `yaml:"value,omitempty"`
}

// ConditionalDefault defines a post-processing attribute override rule.
// Evaluated after all attributes are processed, before formatting.
type ConditionalDefault struct {
	// TargetAttribute is the terraform_name of the attribute to override.
	TargetAttribute string `yaml:"target_attribute"`

	// SetValue is the value to assign when all conditions are met.
	SetValue interface{} `yaml:"set_value"`

	// WhenAll lists conditions that must ALL be true for the override to apply.
	WhenAll []DefaultCondition `yaml:"when_all"`
}

// DefaultCondition is a single condition in a ConditionalDefault rule.
type DefaultCondition struct {
	// AttributeEmpty triggers when the named processed attribute is nil or empty string.
	AttributeEmpty string `yaml:"attribute_empty,omitempty"`

	// AttributeEquals triggers when an attribute matches a specific value.
	AttributeEquals *AttributeValueCondition `yaml:"attribute_equals,omitempty"`
}

// AttributeValueCondition checks a processed attribute's value.
type AttributeValueCondition struct {
	Name  string      `yaml:"name"`
	Value interface{} `yaml:"value"`
}
