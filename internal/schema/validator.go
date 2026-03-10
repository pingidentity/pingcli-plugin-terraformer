package schema

import (
	"fmt"
	"strings"
)

// Validator validates resource definitions
type Validator struct{}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{}
}

// Validate validates a resource definition
func (v *Validator) Validate(def *ResourceDefinition) error {
	var errors []string

	if err := v.validateMetadata(&def.Metadata); err != nil {
		errors = append(errors, err.Error())
	}

	if err := v.validateAPI(&def.API); err != nil {
		errors = append(errors, err.Error())
	}

	for i, attr := range def.Attributes {
		if err := v.validateAttribute(&attr, fmt.Sprintf("attributes[%d]", i)); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if err := v.validateDependencies(&def.Dependencies); err != nil {
		errors = append(errors, err.Error())
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation errors:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

func (v *Validator) validateMetadata(meta *ResourceMetadata) error {
	if meta.Platform == "" {
		return fmt.Errorf("metadata.platform is required")
	}
	if meta.ResourceType == "" {
		return fmt.Errorf("metadata.resource_type is required")
	}
	if meta.APIType == "" {
		return fmt.Errorf("metadata.api_type is required")
	}
	if meta.Name == "" {
		return fmt.Errorf("metadata.name is required")
	}
	if meta.ShortName == "" {
		return fmt.Errorf("metadata.short_name is required")
	}

	validPlatforms := map[string]bool{
		"pingone":      true,
		"pingfederate": true,
	}
	if !validPlatforms[meta.Platform] {
		return fmt.Errorf("metadata.platform '%s' is not valid", meta.Platform)
	}

	return nil
}

func (v *Validator) validateAPI(api *APIDefinition) error {
	if api.SDKPackage == "" {
		return fmt.Errorf("api.sdk_package is required")
	}
	if api.SDKType == "" {
		return fmt.Errorf("api.sdk_type is required")
	}
	if api.ListMethod == "" {
		return fmt.Errorf("api.list_method is required")
	}
	if api.IDField == "" {
		return fmt.Errorf("api.id_field is required")
	}
	if api.NameField == "" {
		return fmt.Errorf("api.name_field is required")
	}

	validPaginationTypes := map[string]bool{
		"cursor": true,
		"offset": true,
		"none":   true,
	}
	if api.PaginationType != "" && !validPaginationTypes[api.PaginationType] {
		return fmt.Errorf("api.pagination_type '%s' is not valid", api.PaginationType)
	}

	return nil
}

func (v *Validator) validateAttribute(attr *AttributeDefinition, path string) error {
	if attr.Name == "" {
		return fmt.Errorf("%s.name is required", path)
	}
	if attr.TerraformName == "" {
		return fmt.Errorf("%s.terraform_name is required", path)
	}
	if attr.Type == "" {
		return fmt.Errorf("%s.type is required", path)
	}

	validTypes := map[string]bool{
		"string":                   true,
		"number":                   true,
		"bool":                     true,
		"object":                   true,
		"list":                     true,
		"map":                      true,
		"set":                      true,
		"type_discriminated_block": true,
	}
	if !validTypes[attr.Type] {
		return fmt.Errorf("%s.type '%s' is not valid", path, attr.Type)
	}

	if attr.Transform != "" {
		validTransforms := map[string]bool{
			"passthrough":    true,
			"base64_encode":  true,
			"base64_decode":  true,
			"json_encode":    true,
			"json_decode":    true,
			"jsonencode_raw": true,
			"to_string":      true,
			"value_map":      true,
			"custom":         true,
		}
		if !validTransforms[attr.Transform] {
			return fmt.Errorf("%s.transform '%s' is not valid", path, attr.Transform)
		}

		if attr.Transform == "custom" && attr.CustomTransform == "" {
			return fmt.Errorf("%s.transform is 'custom' but custom_transform is not set", path)
		}
	}

	if (attr.Type == "object" || attr.Type == "list" || attr.Type == "map") && len(attr.NestedAttributes) > 0 {
		for i, nested := range attr.NestedAttributes {
			nestedPath := fmt.Sprintf("%s.nested_attributes[%d]", path, i)
			if err := v.validateAttribute(&nested, nestedPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func (v *Validator) validateDependencies(deps *DependencyDefinition) error {

	for i, rule := range deps.DependsOn {
		if rule.ResourceType == "" {
			return fmt.Errorf("dependencies.depends_on[%d].resource_type is required", i)
		}
		if rule.FieldPath == "" {
			return fmt.Errorf("dependencies.depends_on[%d].field_path is required", i)
		}
		if rule.ReferenceFormat == "" {
			return fmt.Errorf("dependencies.depends_on[%d].reference_format is required", i)
		}
	}

	return nil
}
