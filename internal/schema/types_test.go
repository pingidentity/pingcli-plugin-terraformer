package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestResourceDefinitionMarshal(t *testing.T) {
	def := &ResourceDefinition{
		Metadata: ResourceMetadata{
			Platform:     "pingone",
			Service:      "davinci",
			ResourceType: "pingone_davinci_variable",
			APIType:      "Variable",
			Name:         "DaVinci Variable",
			ShortName:    "variable",
			Version:      "1.0",
		},
		API: APIDefinition{
			SDKPackage:     "github.com/pingidentity/pingone-go-sdk-v2/davinci",
			SDKType:        "Variable",
			ListMethod:     "EnvironmentVariables.GetAll",
			IDField:        "id",
			NameField:      "name",
			PaginationType: "cursor",
		},
		Attributes: []AttributeDefinition{
			{
				Name:          "ID",
				TerraformName: "id",
				Type:          "string",
				Computed:      true,
			},
			{
				Name:             "Name",
				TerraformName:    "name",
				Type:             "string",
				Required:         true,
				VariableEligible: false,
			},
		},
		Dependencies: DependencyDefinition{
			ImportIDFormat: "{env_id}/{resource_id}",
		},
		Variables: VariableDefinition{
			EligibleAttributes: []VariableRule{
				{
					AttributePath:  "value",
					VariablePrefix: "davinci_variable_",
					IsSecret:       false,
					Description:    "Value for DaVinci variable",
				},
			},
		},
	}

	// Marshal to YAML
	data, err := yaml.Marshal(def)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	// Unmarshal back
	var decoded ResourceDefinition
	err = yaml.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, "pingone", decoded.Metadata.Platform)
	assert.Equal(t, "davinci", decoded.Metadata.Service)
	assert.Equal(t, "pingone_davinci_variable", decoded.Metadata.ResourceType)
	assert.Len(t, decoded.Attributes, 2)
}

func TestAttributeDefinitionTypes(t *testing.T) {
	tests := []struct {
		name     string
		attrType string
		valid    bool
	}{
		{"string type", "string", true},
		{"number type", "number", true},
		{"bool type", "bool", true},
		{"object type", "object", true},
		{"list type", "list", true},
		{"map type", "map", true},
		{"set type", "set", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attr := AttributeDefinition{
				Name: "test",
				Type: tt.attrType,
			}
			assert.Equal(t, tt.attrType, attr.Type)
		})
	}
}

func TestDependencyRuleMarshaling(t *testing.T) {
	rule := DependencyRule{
		ResourceType:    "pingone_davinci_flow",
		FieldPath:       "flow_id",
		LookupField:     "id",
		ReferenceFormat: "pingone_davinci_flow.{resource_name}.id",
		Optional:        true,
	}

	data, err := yaml.Marshal(rule)
	assert.NoError(t, err)

	var decoded DependencyRule
	err = yaml.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, "pingone_davinci_flow", decoded.ResourceType)
	assert.True(t, decoded.Optional)
}
