package schema

// WalkAttributes recurses attrs up to depth levels and calls fn for each qualifying leaf.
//
// A leaf qualifies if:
//   - Its Transform is not "jsonencode_raw"
//   - Its CustomTransform is empty
//   - It is a scalar type (string, bool, number), OR it is computed (e.g. id), OR it is
//     a container type (object/map/list/set) that has no NestedAttributes and is therefore
//     treated as a terminal value.
//
// Container attributes (object/map/list/set) with NestedAttributes are not passed to fn;
// their children are recursed into when currentDepth < depth.
//
// The path argument accumulates the dot-notation prefix. Pass "" for top-level calls.
// At depth <= 0, no attributes are visited.
func WalkAttributes(attrs []AttributeDefinition, depth int, path string, fn func(attrPath string, attr AttributeDefinition)) {
	if depth <= 0 {
		return
	}
	for _, attr := range attrs {
		if shouldSkip(attr) {
			continue
		}
		full := attrPath(path, attr.TerraformName)
		if isContainer(attr) && len(attr.NestedAttributes) > 0 {
			WalkAttributes(attr.NestedAttributes, depth-1, full, fn)
		} else {
			fn(full, attr)
		}
	}
}

func shouldSkip(attr AttributeDefinition) bool {
	return attr.Transform == "jsonencode_raw" || attr.CustomTransform != ""
}

func isContainer(attr AttributeDefinition) bool {
	switch attr.Type {
	case "object", "map", "list", "set":
		return true
	default:
		return false
	}
}

func attrPath(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}
