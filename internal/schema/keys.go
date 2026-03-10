package schema

// CanonicalAttributeKey returns the canonical map key for the given attribute
// definition. This is the single source of truth for how attribute values are
// keyed in the processed data map. Both the processor and the variable
// extractor must use this function to ensure key consistency.
//
// terraform_name is required on all attribute definitions (enforced by the
// validator). If it is empty, this function panics — the definition should
// never have passed validation.
func CanonicalAttributeKey(attr AttributeDefinition) string {
	if attr.TerraformName == "" {
		panic("terraform_name is required but empty — validator should have caught this for attribute: " + attr.Name)
	}
	return attr.TerraformName
}
