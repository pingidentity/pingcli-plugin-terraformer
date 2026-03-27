package core

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
)

// RawHCLValue wraps a string that should be written to HCL without quoting.
// Custom transforms return this for values like json_object that must be
// emitted verbatim (not Go %q-quoted).
type RawHCLValue string

// ResolvedReference represents a cross-resource reference that has been resolved
// through the dependency graph. The orchestrator replaces raw UUID strings with
// ResolvedReference values during post-processing. Formatters use type assertion
// to detect these and render them appropriately for each output format:
//   - HCL: unquoted traversal expression (e.g. pingone_davinci_flow.my_flow.id)
//   - JSON: string value
//   - Variable reference: var.pingone_environment_id
type ResolvedReference struct {
	// ResourceType is the Terraform resource type (e.g. "pingone_davinci_flow").
	ResourceType string

	// ResourceName is the sanitized Terraform resource label (e.g. "pingcli__my_flow").
	ResourceName string

	// Field is the attribute being referenced (e.g. "id").
	Field string

	// IsVariable indicates this reference should be rendered as a Terraform
	// variable (var.{name}) rather than a resource traversal.
	IsVariable bool

	// VariableName is the full variable name when IsVariable is true
	// (e.g. "pingone_environment_id").
	VariableName string

	// OriginalValue is the raw UUID that was resolved, preserved for fallback.
	OriginalValue string
}

// Expression returns the Terraform expression string for this reference.
// For variable references: "var.pingone_environment_id"
// For resource references: "pingone_davinci_flow.pingcli__my_flow.id"
func (r ResolvedReference) Expression() string {
	if r.IsVariable {
		return "var." + r.VariableName
	}
	return r.ResourceType + "." + r.ResourceName + "." + r.Field
}

// Processor converts API responses to intermediate representation using schema definitions.
type Processor struct {
	registry       *schema.Registry
	customHandlers *CustomHandlerRegistry
}

// ProcessorOption configures optional Processor behaviour.
type ProcessorOption func(*Processor)

// WithCustomHandlers attaches a custom handler registry to the processor.
// When set, attributes with transform "custom" and resource-level custom
// handlers declared in YAML are dispatched through this registry.
func WithCustomHandlers(reg *CustomHandlerRegistry) ProcessorOption {
	return func(p *Processor) {
		p.customHandlers = reg
	}
}

// NewProcessor creates a new processor with the given registry.
func NewProcessor(registry *schema.Registry, opts ...ProcessorOption) *Processor {
	p := &Processor{
		registry: registry,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// ExtractedVariable holds variable metadata produced during resource processing.
// Both schema-driven extraction (VariableExtractor) and custom transforms
// (TransformResultWithVariables) produce this type.
type ExtractedVariable struct {
	// AttributePath is the dot-notation path to the source attribute
	// (e.g., "value", "properties.clientId").
	AttributePath string

	// VariableName is the generated Terraform variable name
	// (e.g., "davinci_variable_my_var_value").
	VariableName string

	// VariableType is the Terraform type for the variable ("string", "number", "bool", "any").
	VariableType string

	// CurrentValue is the raw value from the API response. Nil for secrets.
	CurrentValue interface{}

	// Description for the Terraform variable block.
	Description string

	// Sensitive marks the variable as sensitive in Terraform.
	Sensitive bool

	// IsSecret indicates the value should not appear in .auto.tfvars.
	IsSecret bool

	// ResourceType is the full Terraform resource type (e.g., "pingone_davinci_connector_instance").
	ResourceType string

	// ResourceName is the sanitized Terraform resource label.
	ResourceName string

	// ResourceID is the original API resource ID.
	ResourceID string
}

// TransformResultWithVariables wraps a transform output value together with
// variable extraction metadata. Custom transforms return this type when they
// produce HCL containing variable references that need corresponding
// declarations in variables.tf, module.tf, and tfvars.
type TransformResultWithVariables struct {
	Value     interface{}
	Variables []ExtractedVariable
}

// ResourceData represents the intermediate representation of a resource
type ResourceData struct {
	ResourceType string
	ID           string
	Name         string
	// Label is the sanitized, unique Terraform resource label assigned by the
	// orchestrator. All downstream consumers (formatters, import generators)
	// must use this field instead of re-deriving the label from Name/Attributes.
	// Set during Export after processing and uniqueness validation.
	Label              string
	Attributes         map[string]interface{}
	Dependencies       []Dependency
	ExtractedVariables []ExtractedVariable
}

// Dependency represents a resource dependency
type Dependency struct {
	Type   string
	ID     string
	Format string
}

// ProcessResource processes API data into resource data using the schema definition
func (p *Processor) ProcessResource(resourceType string, apiData interface{}) (*ResourceData, error) {
	// Get the schema definition
	def, err := p.registry.Get(resourceType)
	if err != nil {
		return nil, fmt.Errorf("failed to get definition for %s: %w", resourceType, err)
	}

	// Convert API data to reflect.Value for inspection
	val := reflect.ValueOf(apiData)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// Initialize resource data
	result := &ResourceData{
		ResourceType: resourceType,
		Attributes:   make(map[string]interface{}),
		Dependencies: []Dependency{},
	}

	// Process each attribute from the schema
	for _, attrDef := range def.Attributes {
		tfName := schema.CanonicalAttributeKey(attrDef)

		// Handle type_discriminated_block attributes via YAML-driven logic.
		if attrDef.Type == "type_discriminated_block" && attrDef.TypeDiscriminatedBlock != nil {
			blockVal, err := p.processTypeDiscriminatedBlock(apiData, attrDef)
			if err != nil {
				return nil, fmt.Errorf("type_discriminated_block %s: %w", attrDef.Name, err)
			}
			if blockVal != nil {
				result.Attributes[tfName] = blockVal
			}
			continue
		}

		// If override_value is set, use it directly and skip extraction/transforms.
		if attrDef.OverrideValue != nil {
			result.Attributes[tfName] = attrDef.OverrideValue
			continue
		}

		value, err := p.extractAttribute(val, attrDef)
		if err != nil {
			// Skip attributes that can't be extracted (optional fields)
			continue
		}

		// Store in attributes map
		if value != nil {
			// For object types with nested_attributes, recursively extract the
			// nested fields into a map[string]interface{} that the formatter expects.
			if attrDef.Type == "object" && len(attrDef.NestedAttributes) > 0 {
				nestedMap, nErr := p.extractNestedAttributes(value, attrDef.NestedAttributes)
				if nErr != nil {
					// If nested extraction fails, skip the attribute.
					continue
				}
				if len(nestedMap) > 0 {
					result.Attributes[tfName] = nestedMap
				}
				continue
			}

			// For list types with nested_attributes, extract each element
			// through nested attribute processing.
			if (attrDef.Type == "list" || attrDef.Type == "set") && len(attrDef.NestedAttributes) > 0 {
				listVal, lErr := p.extractListOfNestedAttributes(value, attrDef.NestedAttributes)
				if lErr != nil {
					continue
				}
				if len(listVal) > 0 {
					result.Attributes[tfName] = listVal
				}
				continue
			}

			// Apply transform if specified
			if attrDef.Transform == "custom" && attrDef.CustomTransform != "" {
				// Dispatch to custom transform via registry.
				value, err = p.applyCustomTransform(attrDef.CustomTransform, value, apiData, &attrDef, def)
				if err != nil {
					return nil, fmt.Errorf("custom transform %q on attribute %s: %w", attrDef.CustomTransform, attrDef.Name, err)
				}
				// Unwrap TransformResultWithVariables if the custom transform
				// returned variable metadata alongside its value.
				if wrapped, ok := value.(TransformResultWithVariables); ok {
					value = wrapped.Value
					result.ExtractedVariables = append(result.ExtractedVariables, wrapped.Variables...)
				}
			} else if attrDef.Transform != "" {
				transformed, tErr := ApplyTransform(attrDef.Transform, value, &attrDef)
				if tErr != nil {
					return nil, fmt.Errorf("transform %q on attribute %s: %w", attrDef.Transform, attrDef.Name, tErr)
				}
				value = transformed
			}

			result.Attributes[tfName] = value

			// Check for ID field (match against terraform_name, case-insensitive)
			if strings.EqualFold(tfName, def.API.IDField) {
				if idStr, ok := value.(string); ok {
					result.ID = idStr
				}
			}

			// Check for name field (match against terraform_name, case-insensitive)
			if strings.EqualFold(tfName, def.API.NameField) {
				if nameStr, ok := value.(string); ok {
					result.Name = nameStr
				}
			}
		}
	}

	// Process dependencies
	if def.Dependencies.DependsOn != nil {
		for _, depRule := range def.Dependencies.DependsOn {
			dep := Dependency{
				Type:   depRule.ResourceType,
				Format: depRule.ReferenceFormat,
			}
			result.Dependencies = append(result.Dependencies, dep)
		}
	}

	// Run resource-level custom handlers declared in YAML.
	if err := p.runCustomHandlers(def, apiData, result); err != nil {
		return nil, err
	}

	// Evaluate declarative conditional defaults from YAML.
	p.evaluateConditionalDefaults(result.Attributes, def.ConditionalDefaults)

	return result, nil
}

// applyCustomTransform dispatches a single attribute value through the named
// custom transform in the CustomHandlerRegistry.
func (p *Processor) applyCustomTransform(name string, value interface{}, apiData interface{}, attr *schema.AttributeDefinition, def *schema.ResourceDefinition) (interface{}, error) {
	if p.customHandlers == nil {
		// No registry — pass through silently.
		return value, nil
	}
	fn, err := p.customHandlers.GetTransform(name)
	if err != nil {
		// Transform not registered — pass through.
		return value, nil
	}
	return fn(value, apiData, attr, def)
}

// runCustomHandlers invokes resource-level custom handlers declared in the
// YAML definition's custom_handlers block. Handler results are merged into
// the processed resource attributes.
func (p *Processor) runCustomHandlers(def *schema.ResourceDefinition, apiData interface{}, result *ResourceData) error {
	if p.customHandlers == nil || def.CustomHandlers == nil {
		return nil
	}

	// Invoke the transformer handler if declared.
	if name := def.CustomHandlers.Transformer; name != "" {
		fn, err := p.customHandlers.GetHandler(name)
		if err != nil {
			// Not registered — treat as no-op (stubs may not be loaded).
			return nil
		}
		extra, err := fn(apiData, def)
		if err != nil {
			return fmt.Errorf("custom handler %q: %w", name, err)
		}
		for k, v := range extra {
			result.Attributes[k] = v
		}
	}

	// Invoke the HCL generator handler if declared.
	if name := def.CustomHandlers.HCLGenerator; name != "" {
		fn, err := p.customHandlers.GetHandler(name)
		if err != nil {
			return nil
		}
		extra, err := fn(apiData, def)
		if err != nil {
			return fmt.Errorf("custom handler %q: %w", name, err)
		}
		for k, v := range extra {
			result.Attributes[k] = v
		}
	}

	return nil
}

// extractAttribute extracts a single attribute value from the API data
func (p *Processor) extractAttribute(val reflect.Value, attrDef schema.AttributeDefinition) (interface{}, error) {
	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %v", val.Kind())
	}

	// Use source_path when available, fall back to attribute Name
	path := attrDef.Name
	if attrDef.SourcePath != "" {
		path = attrDef.SourcePath
	}

	// Find the field using dot-notation traversal
	field, found := findFieldByPath(val, path)
	if !found || !field.IsValid() {
		return nil, fmt.Errorf("field %s not found", path)
	}

	// Track whether original field was a pointer. Non-pointer fields at zero
	// value are indistinguishable from "not set" and should be suppressed.
	wasPointer := field.Kind() == reflect.Ptr

	// Handle pointer fields
	if field.Kind() == reflect.Ptr {
		if field.IsNil() {
			// Optional field not set
			return nil, nil
		}
		field = field.Elem()
	}

	// Skip zero-value non-pointer fields (empty string, nil slice, nil map).
	// Booleans and numbers are kept even at zero since conditional defaults may need them.
	if !wasPointer && isEmptyValue(field) {
		return nil, nil
	}

	// Convert based on attribute type
	return p.convertValue(field, attrDef.Type)
}

// extractNestedAttributes converts a Go value (typically a struct) into a
// map[string]interface{} keyed by terraform_name, using the nested attribute
// definitions. This bridges struct reflection → the map format the formatter expects.
func (p *Processor) extractNestedAttributes(parent interface{}, nested []schema.AttributeDefinition) (map[string]interface{}, error) {
	rv := reflect.ValueOf(parent)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil, nil
		}
		rv = rv.Elem()
	}

	// Handle map[string]interface{} input (already processed data).
	if rv.Kind() == reflect.Map {
		m, ok := parent.(map[string]interface{})
		if ok {
			return m, nil
		}
	}

	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("extractNestedAttributes: expected struct or map, got %v", rv.Kind())
	}

	result := make(map[string]interface{})
	for _, attr := range nested {
		tfName := schema.CanonicalAttributeKey(attr)

		// If override_value is set, use it directly and skip extraction.
		if attr.OverrideValue != nil {
			result[tfName] = attr.OverrideValue
			continue
		}

		path := attr.Name
		if attr.SourcePath != "" {
			path = attr.SourcePath
		}

		field, found := findFieldByPath(rv, path)
		if !found || !field.IsValid() {
			continue
		}
		// Track whether the original field was behind a pointer. Pointer-typed
		// fields use nil to signal absence, so a non-nil pointer that dereferences
		// to a zero value (e.g. *bool → false) is intentional and must be kept.
		// Non-pointer fields have no nil state; their zero value is indistinguishable
		// from "not set", so we skip them.
		wasPointer := field.Kind() == reflect.Ptr
		for field.Kind() == reflect.Ptr {
			if field.IsNil() {
				break
			}
			field = field.Elem()
		}
		if !field.IsValid() || (field.Kind() == reflect.Ptr && field.IsNil()) {
			continue
		}
		// Skip empty non-pointer fields (empty string, nil slice, nil map).
		if !wasPointer && isEmptyValue(field) {
			continue
		}

		// Slice-to-map keying: when the schema expects a map but the source
		// is a slice, and map_key_path is set, convert the slice to a map
		// keyed by the specified sub-field path.
		if attr.Type == "map" && attr.MapKeyPath != "" && field.Kind() == reflect.Slice {
			mapped, mErr := p.convertSliceToMap(field, attr)
			if mErr != nil {
				continue
			}
			if len(mapped) > 0 {
				result[tfName] = mapped
			}
			continue
		}

		val, err := p.convertValue(field, attr.Type)
		if err != nil {
			continue
		}
		if val != nil {
			// Recursively handle nested objects.
			if attr.Type == "object" && len(attr.NestedAttributes) > 0 {
				sub, subErr := p.extractNestedAttributes(val, attr.NestedAttributes)
				if subErr == nil && len(sub) > 0 {
					result[tfName] = sub
				}
				continue
			}

			// Recursively handle list/set of objects with nested attributes.
			if (attr.Type == "list" || attr.Type == "set") && len(attr.NestedAttributes) > 0 {
				listVal, lErr := p.extractListOfNestedAttributes(val, attr.NestedAttributes)
				if lErr == nil && len(listVal) > 0 {
					result[tfName] = listVal
				}
				continue
			}

			// Apply transform if specified on nested attribute.
			if attr.Transform != "" && attr.Transform != "custom" {
				transformed, tErr := ApplyTransform(attr.Transform, val, &attr)
				if tErr != nil {
					continue
				}
				val = transformed
			}

			result[tfName] = val
		}
	}
	return result, nil
}

// convertSliceToMap converts a reflect.Slice value into a map[string]interface{}
// keyed by extracting the sub-field at mapKeyPath from each element. When nested
// attributes are defined, each element is recursively processed through
// extractNestedAttributes.
func (p *Processor) convertSliceToMap(field reflect.Value, attr schema.AttributeDefinition) (map[string]interface{}, error) {
	if field.Kind() != reflect.Slice {
		return nil, fmt.Errorf("convertSliceToMap: expected slice, got %v", field.Kind())
	}

	result := make(map[string]interface{})
	for i := 0; i < field.Len(); i++ {
		elem := field.Index(i)
		// Dereference pointers.
		for elem.Kind() == reflect.Ptr {
			if elem.IsNil() {
				break
			}
			elem = elem.Elem()
		}
		if !elem.IsValid() || (elem.Kind() == reflect.Ptr && elem.IsNil()) {
			continue
		}

		// Extract the map key from the element.
		keyField, found := findFieldByPath(elem, attr.MapKeyPath)
		if !found || !keyField.IsValid() {
			continue
		}
		// Dereference the key if it's a pointer.
		for keyField.Kind() == reflect.Ptr {
			if keyField.IsNil() {
				break
			}
			keyField = keyField.Elem()
		}
		if !keyField.IsValid() {
			continue
		}
		key := fmt.Sprintf("%v", keyField.Interface())
		if key == "" {
			continue
		}

		// Process nested attributes if defined.
		if len(attr.NestedAttributes) > 0 {
			nestedMap, err := p.extractNestedAttributes(elem.Interface(), attr.NestedAttributes)
			if err != nil {
				continue
			}
			result[key] = nestedMap
		} else {
			result[key] = elem.Interface()
		}
	}

	return result, nil
}

// extractListOfNestedAttributes converts a slice value where each element is
// processed through extractNestedAttributes. Returns []interface{} where each
// element is a map[string]interface{} from nested attribute extraction.
func (p *Processor) extractListOfNestedAttributes(value interface{}, nested []schema.AttributeDefinition) ([]interface{}, error) {
	rv := reflect.ValueOf(value)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil, nil
		}
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil, fmt.Errorf("extractListOfNestedAttributes: expected slice, got %v", rv.Kind())
	}

	result := make([]interface{}, 0, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		elem := rv.Index(i)
		for elem.Kind() == reflect.Ptr {
			if elem.IsNil() {
				break
			}
			elem = elem.Elem()
		}
		if !elem.IsValid() || (elem.Kind() == reflect.Ptr && elem.IsNil()) {
			continue
		}

		mapped, err := p.extractNestedAttributes(elem.Interface(), nested)
		if err != nil {
			continue
		}
		if len(mapped) > 0 {
			result = append(result, mapped)
		}
	}

	return result, nil
}

// isEmptyValue returns true for values that represent "not set" when the field
// is a non-pointer type: empty strings, nil slices, nil maps. Booleans and
// numbers are NOT considered empty even at their zero values.
func isEmptyValue(field reflect.Value) bool {
	switch field.Kind() {
	case reflect.String:
		return field.Len() == 0
	case reflect.Slice, reflect.Map:
		return field.IsNil()
	default:
		return false
	}
}

// findFieldByPath resolves a dot-notation path (e.g. "Parent.Field") against
// a reflect.Value representing a struct. Each segment is matched by exact
// struct field name. Pointer fields are dereferenced automatically at each step.
func findFieldByPath(val reflect.Value, path string) (reflect.Value, bool) {
	parts := strings.Split(path, ".")
	current := val

	for _, part := range parts {
		// Dereference pointers at each level
		for current.Kind() == reflect.Ptr {
			if current.IsNil() {
				return reflect.Value{}, false
			}
			current = current.Elem()
		}

		if current.Kind() != reflect.Struct {
			return reflect.Value{}, false
		}

		field := findStructField(current, part)
		if !field.IsValid() {
			return reflect.Value{}, false
		}
		current = field
	}

	return current, true
}

// findStructField finds a field by exact name in a struct value.
func findStructField(val reflect.Value, name string) reflect.Value {
	t := val.Type()
	for i := 0; i < val.NumField(); i++ {
		if t.Field(i).Name == name {
			return val.Field(i)
		}
	}
	return reflect.Value{}
}

// convertValue converts a reflect.Value to the appropriate Go type based on schema type
func (p *Processor) convertValue(field reflect.Value, attrType string) (interface{}, error) {
	// Choice wrapper unwrapping: when the schema expects a primitive type but
	// the reflect value is a struct (common with PingOne SDK choice wrappers
	// like DaVinciFlowSettingsResponseCustomErrorShowFooter{Bool *bool, Object
	// *map[string]interface{}}), attempt to unwrap to the single non-nil
	// pointer field.
	if field.Kind() == reflect.Struct && isPrimitiveAttrType(attrType) {
		unwrapped, ok := unwrapUnionStruct(field)
		if ok {
			field = reflect.ValueOf(unwrapped)
		} else {
			// All fields nil — treat as absent.
			return nil, nil
		}
	}

	switch attrType {
	case "string":
		if field.Kind() == reflect.String {
			return field.String(), nil
		}
		// Handle fmt.Stringer (e.g., uuid.UUID) and any other named types.
		if field.CanInterface() {
			if s, ok := field.Interface().(fmt.Stringer); ok {
				return s.String(), nil
			}
		}
		return fmt.Sprintf("%v", field.Interface()), nil

	case "number":
		switch field.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return field.Int(), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return int64(field.Uint()), nil
		case reflect.Float32, reflect.Float64:
			return field.Float(), nil
		}
		// Handle struct types with a Float64() method (e.g., math/big.Float
		// unwrapped from SDK wrapper types like BigFloatUnquoted).
		if field.Kind() == reflect.Struct {
			ptr := reflect.New(field.Type())
			ptr.Elem().Set(field)
			if m := ptr.MethodByName("Float64"); m.IsValid() {
				results := m.Call(nil)
				if len(results) >= 1 && results[0].CanFloat() {
					return results[0].Float(), nil
				}
			}
		}
		return nil, fmt.Errorf("cannot convert %v to number", field.Kind())

	case "bool":
		if field.Kind() == reflect.Bool {
			return field.Bool(), nil
		}
		// Handle string "true"/"false" from SDK choice wrapper union types
		// (e.g., Choice2 variant returns a named string type).
		if field.Kind() == reflect.String {
			switch strings.ToLower(field.String()) {
			case "true":
				return true, nil
			case "false":
				return false, nil
			}
		}
		return nil, fmt.Errorf("cannot convert %v to bool", field.Kind())

	case "object":
		// Return as-is for object types
		return field.Interface(), nil

	case "list":
		if field.Kind() == reflect.Slice || field.Kind() == reflect.Array {
			result := make([]interface{}, field.Len())
			for i := 0; i < field.Len(); i++ {
				elem := field.Index(i)
				// Coerce typed string aliases (e.g., DaVinciApplicationResponseOAuthGrantType)
				// to plain strings so the formatter doesn't encounter unknown types.
				if elem.Kind() == reflect.String {
					result[i] = elem.String()
				} else if elem.CanInterface() {
					if s, ok := elem.Interface().(fmt.Stringer); ok {
						result[i] = s.String()
					} else {
						result[i] = elem.Interface()
					}
				} else {
					result[i] = elem.Interface()
				}
			}
			return result, nil
		}
		return nil, fmt.Errorf("cannot convert %v to list", field.Kind())

	case "map":
		if field.Kind() == reflect.Map {
			return field.Interface(), nil
		}
		return nil, fmt.Errorf("cannot convert %v to map", field.Kind())

	case "set":
		// Sets are similar to lists in the intermediate representation
		if field.Kind() == reflect.Slice || field.Kind() == reflect.Array {
			result := make([]interface{}, field.Len())
			for i := 0; i < field.Len(); i++ {
				elem := field.Index(i)
				// Coerce typed string aliases to plain strings (same as list case).
				if elem.Kind() == reflect.String {
					result[i] = elem.String()
				} else if elem.CanInterface() {
					if s, ok := elem.Interface().(fmt.Stringer); ok {
						result[i] = s.String()
					} else {
						result[i] = elem.Interface()
					}
				} else {
					result[i] = elem.Interface()
				}
			}
			return result, nil
		}
		return nil, fmt.Errorf("cannot convert %v to set", field.Kind())

	default:
		return nil, fmt.Errorf("unknown attribute type: %s", attrType)
	}
}

// processTypeDiscriminatedBlock converts a runtime-typed API field into a
// single-key map whose key is determined by the Go type of the value. The
// mapping is declared in the YAML definition's type_discriminated_block config.
func (p *Processor) processTypeDiscriminatedBlock(apiData interface{}, attrDef schema.AttributeDefinition) (map[string]interface{}, error) {
	cfg := attrDef.TypeDiscriminatedBlock

	// Check skip conditions against apiData struct fields.
	for _, sc := range cfg.SkipConditions {
		fieldVal := ReadStringField(apiData, sc.SourceField)
		if strings.EqualFold(fieldVal, sc.Equals) {
			return nil, nil
		}
	}

	// Read the raw value from apiData via source_path, using findFieldByPath
	// for dot-notation support.
	sourcePath := attrDef.SourcePath
	if sourcePath == "" {
		sourcePath = attrDef.Name
	}

	val := reflect.ValueOf(apiData)
	for val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil, nil
		}
		val = val.Elem()
	}

	field, found := findFieldByPath(val, sourcePath)
	if !found || !field.IsValid() {
		return nil, nil
	}

	// Dereference pointers.
	for field.Kind() == reflect.Ptr {
		if field.IsNil() {
			return nil, nil
		}
		field = field.Elem()
	}

	// Unwrap union structs: if the value is a struct where all exported fields
	// are pointers, find the first non-nil field and use its dereferenced value.
	// This handles SDK union types like DaVinciVariableResponseValue.
	var rawValue interface{}
	if field.Kind() == reflect.Struct {
		unwrapped, ok := unwrapUnionStruct(field)
		if ok {
			rawValue = unwrapped
		} else {
			rawValue = field.Interface()
		}
	} else if field.Kind() == reflect.Interface {
		if field.IsNil() {
			return nil, nil
		}
		rawValue = field.Interface()
	} else {
		rawValue = field.Interface()
	}

	if rawValue == nil {
		return nil, nil
	}

	// Check for empty values.
	if str, ok := rawValue.(string); ok && str == "" {
		return nil, nil
	}

	// Determine Go runtime type name.
	typeName := goRuntimeTypeName(rawValue)

	// Look up block key in TypeKeyMap.
	blockKey, ok := cfg.TypeKeyMap[typeName]
	if !ok {
		return nil, nil
	}

	// Check if this key requires JSON encoding.
	needsJSON := false
	for _, jk := range cfg.JSONEncodeKeys {
		if jk == blockKey {
			needsJSON = true
			break
		}
	}

	if needsJSON {
		jsonBytes, err := json.Marshal(rawValue)
		if err != nil {
			return nil, fmt.Errorf("json marshal for key %q: %w", blockKey, err)
		}
		if len(jsonBytes) <= 2 { // {} or []
			return nil, nil
		}
		return map[string]interface{}{blockKey: RawHCLValue(string(jsonBytes))}, nil
	}

	// For float64, check if it's an integer and emit int64.
	if f, ok := rawValue.(float64); ok {
		if f == float64(int64(f)) {
			return map[string]interface{}{blockKey: int64(f)}, nil
		}
		return map[string]interface{}{blockKey: f}, nil
	}

	// For float32, convert to float64 for consistent downstream handling.
	if f, ok := rawValue.(float32); ok {
		f64 := float64(f)
		if f64 == float64(int64(f64)) {
			return map[string]interface{}{blockKey: int64(f64)}, nil
		}
		return map[string]interface{}{blockKey: f64}, nil
	}

	// For int, emit as int64.
	if i, ok := rawValue.(int); ok {
		return map[string]interface{}{blockKey: int64(i)}, nil
	}

	return map[string]interface{}{blockKey: rawValue}, nil
}

// isPrimitiveAttrType returns true for schema types that map to Go primitives.
// Used to trigger choice wrapper unwrapping when a struct is encountered but
// the schema expects a primitive.
func isPrimitiveAttrType(attrType string) bool {
	switch attrType {
	case "string", "number", "bool":
		return true
	}
	return false
}

// goRuntimeTypeName returns the Go runtime type name for use in TypeKeyMap lookups.
func goRuntimeTypeName(v interface{}) string {
	switch v.(type) {
	case string:
		return "string"
	case bool:
		return "bool"
	case float32:
		return "float32"
	case float64:
		return "float64"
	case int:
		return "int"
	case int64:
		return "int64"
	case map[string]interface{}:
		return "map"
	case []interface{}:
		return "slice"
	default:
		// Handle named types with underlying primitive kinds (e.g., SDK enum types).
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.String:
			return "string"
		case reflect.Bool:
			return "bool"
		case reflect.Float32:
			return "float32"
		case reflect.Float64:
			return "float64"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return "int"
		case reflect.Map:
			return "map"
		case reflect.Slice:
			return "slice"
		}
		return reflect.TypeOf(v).String()
	}
}

// unwrapUnionStruct detects SDK union-type structs (e.g., DaVinciVariableResponseValue)
// where all exported fields are pointers and exactly one is non-nil. Returns the
// dereferenced value of the first non-nil field, or (nil, false) if the struct
// does not match the union pattern or has no non-nil fields.
func unwrapUnionStruct(val reflect.Value) (interface{}, bool) {
	if val.Kind() != reflect.Struct {
		return nil, false
	}
	t := val.Type()
	allPointers := true
	var firstNonNil reflect.Value

	for i := 0; i < val.NumField(); i++ {
		sf := t.Field(i)
		// Skip unexported fields and AdditionalProperties.
		if !sf.IsExported() || sf.Name == "AdditionalProperties" {
			continue
		}
		fv := val.Field(i)
		if fv.Kind() != reflect.Ptr {
			allPointers = false
			break
		}
		if !fv.IsNil() && !firstNonNil.IsValid() {
			firstNonNil = fv.Elem()
		}
	}

	if !allPointers || !firstNonNil.IsValid() {
		return nil, false
	}

	return firstNonNil.Interface(), true
}

// evaluateConditionalDefaults applies declarative post-processing overrides
// to the processed attributes. Each rule's WhenAll conditions must all be true
// for the override to apply.
func (p *Processor) evaluateConditionalDefaults(attrs map[string]interface{}, defaults []schema.ConditionalDefault) {
	for _, cd := range defaults {
		if p.allConditionsMet(attrs, cd.WhenAll) {
			attrs[cd.TargetAttribute] = cd.SetValue
		}
	}
}

// allConditionsMet checks whether all conditions in a ConditionalDefault rule
// are satisfied by the current attribute map.
func (p *Processor) allConditionsMet(attrs map[string]interface{}, conditions []schema.DefaultCondition) bool {
	for _, cond := range conditions {
		if cond.AttributeEmpty != "" {
			val, exists := attrs[cond.AttributeEmpty]
			if exists && val != nil {
				if s, ok := val.(string); ok && s == "" {
					// empty string counts as empty
				} else {
					return false
				}
			}
			// nil or not-present counts as empty — condition passes
		}

		if cond.AttributeEquals != nil {
			val, exists := attrs[cond.AttributeEquals.Name]
			if !exists {
				return false
			}
			if !valuesEqual(val, cond.AttributeEquals.Value) {
				return false
			}
		}
	}
	return true
}

// valuesEqual compares two values for equality, handling type coercion between
// bool and interface{} from YAML unmarshaling.
func valuesEqual(a, b interface{}) bool {
	if a == b {
		return true
	}
	// YAML unmarshals booleans as bool, but processor may store them differently.
	if ab, ok := a.(bool); ok {
		if bb, ok := b.(bool); ok {
			return ab == bb
		}
	}
	// Compare as strings for mixed-type edge cases.
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// ProcessResourceList processes a list of API resources
func (p *Processor) ProcessResourceList(resourceType string, apiDataList interface{}) ([]*ResourceData, error) {
	val := reflect.ValueOf(apiDataList)
	if val.Kind() != reflect.Slice && val.Kind() != reflect.Array {
		return nil, fmt.Errorf("expected slice or array, got %v", val.Kind())
	}

	results := make([]*ResourceData, 0, val.Len())
	for i := 0; i < val.Len(); i++ {
		item := val.Index(i).Interface()
		resourceData, err := p.ProcessResource(resourceType, item)
		if err != nil {
			return nil, fmt.Errorf("failed to process resource at index %d: %w", i, err)
		}
		results = append(results, resourceData)
	}

	return results, nil
}
