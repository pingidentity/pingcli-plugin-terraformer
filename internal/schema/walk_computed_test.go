package schema

import (
	"testing"
)

// TestWalkAttributes_ComputedMetadataBlockEnumerated verifies the #125 audit
// expectation: computed nested attributes are enumerated as output paths.
func TestWalkAttributes_ComputedMetadataBlockEnumerated(t *testing.T) {
	attrs := []AttributeDefinition{
		{Name: "ID", TerraformName: "id", Type: "string", Computed: true},
		{Name: "Metadata", TerraformName: "metadata", Type: "object", Computed: true, NestedAttributes: []AttributeDefinition{
			{Name: "Type", TerraformName: "type", Type: "string", Computed: true},
			{Name: "Colors", TerraformName: "colors", Type: "object", Computed: true, NestedAttributes: []AttributeDefinition{
				{Name: "Canvas", TerraformName: "canvas", Type: "string", Computed: true},
			}},
		}},
	}
	var got []string
	WalkAttributes(attrs, 5, "", func(p string, a AttributeDefinition) {
		got = append(got, p)
	})
	want := []string{"id", "metadata.type", "metadata.colors.canvas"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
