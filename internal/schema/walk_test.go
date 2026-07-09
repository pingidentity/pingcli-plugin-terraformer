package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWalkAttributes(t *testing.T) {
	tests := []struct {
		name     string
		attrs    []AttributeDefinition
		depth    int
		expected []string // collected attrPath values
	}{
		{
			name:     "depth 0 visits nothing",
			attrs:    []AttributeDefinition{{Name: "ID", TerraformName: "id", Type: "string"}},
			depth:    0,
			expected: []string{},
		},
		{
			name: "depth 1 flat scalars",
			attrs: []AttributeDefinition{
				{Name: "ID", TerraformName: "id", Type: "string", Computed: true},
				{Name: "Name", TerraformName: "name", Type: "string"},
				{Name: "Enabled", TerraformName: "enabled", Type: "bool"},
			},
			depth:    1,
			expected: []string{"id", "name", "enabled"},
		},
		{
			name: "depth 1 does not descend into nested object",
			attrs: []AttributeDefinition{
				{Name: "Name", TerraformName: "name", Type: "string"},
				{
					Name:          "Settings",
					TerraformName: "settings",
					Type:          "object",
					NestedAttributes: []AttributeDefinition{
						{Name: "CSP", TerraformName: "csp", Type: "string"},
					},
				},
			},
			depth:    1,
			expected: []string{"name"},
		},
		{
			name: "depth 2 descends one level into nested object",
			attrs: []AttributeDefinition{
				{Name: "Name", TerraformName: "name", Type: "string"},
				{
					Name:          "Settings",
					TerraformName: "settings",
					Type:          "object",
					NestedAttributes: []AttributeDefinition{
						{Name: "CSP", TerraformName: "csp", Type: "string"},
						{Name: "Sandbox", TerraformName: "sandbox", Type: "bool"},
					},
				},
			},
			depth:    2,
			expected: []string{"name", "settings.csp", "settings.sandbox"},
		},
		{
			name: "depth 2 does not go three levels",
			attrs: []AttributeDefinition{
				{
					Name:          "Outer",
					TerraformName: "outer",
					Type:          "object",
					NestedAttributes: []AttributeDefinition{
						{
							Name:          "Middle",
							TerraformName: "middle",
							Type:          "object",
							NestedAttributes: []AttributeDefinition{
								{Name: "Inner", TerraformName: "inner", Type: "string"},
							},
						},
					},
				},
			},
			depth:    2,
			expected: []string{},
		},
		{
			name: "computed scalar is included",
			attrs: []AttributeDefinition{
				{Name: "ID", TerraformName: "id", Type: "string", Computed: true},
			},
			depth:    1,
			expected: []string{"id"},
		},
		{
			name: "jsonencode_raw transform is skipped",
			attrs: []AttributeDefinition{
				{Name: "Name", TerraformName: "name", Type: "string"},
				{Name: "Data", TerraformName: "graph_data", Type: "object", Transform: "jsonencode_raw"},
			},
			depth:    1,
			expected: []string{"name"},
		},
		{
			name: "custom_transform is skipped",
			attrs: []AttributeDefinition{
				{Name: "Name", TerraformName: "name", Type: "string"},
				{Name: "Special", TerraformName: "special", Type: "string", CustomTransform: "someHandler"},
			},
			depth:    1,
			expected: []string{"name"},
		},
		{
			name:     "empty attrs returns nothing",
			attrs:    []AttributeDefinition{},
			depth:    1,
			expected: []string{},
		},
		{
			name: "container with no nested attrs is treated as leaf",
			attrs: []AttributeDefinition{
				{Name: "Tags", TerraformName: "tags", Type: "map"},
			},
			depth:    1,
			expected: []string{"tags"},
		},
		{
			name: "list nested attrs descended at depth 2",
			attrs: []AttributeDefinition{
				{
					Name:          "Items",
					TerraformName: "items",
					Type:          "list",
					NestedAttributes: []AttributeDefinition{
						{Name: "Value", TerraformName: "value", Type: "string"},
					},
				},
			},
			depth:    2,
			expected: []string{"items.value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			WalkAttributes(tt.attrs, tt.depth, "", func(p string, _ AttributeDefinition) {
				got = append(got, p)
			})
			if got == nil {
				got = []string{}
			}
			assert.Equal(t, tt.expected, got)
		})
	}
}
