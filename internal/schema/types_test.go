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

func TestResourceMetadataEnabledField(t *testing.T) {
	tests := []struct {
		name          string
		yaml          string
		expectedTrue  bool
		expectedFalse bool
		expectedNil   bool
	}{
		{
			name: "enabled: true",
			yaml: `platform: pingone
resource_type: test_resource
api_type: TestAPI
name: Test
short_name: test
version: 1.0
enabled: true`,
			expectedTrue:  true,
			expectedFalse: false,
			expectedNil:   false,
		},
		{
			name: "enabled: false",
			yaml: `platform: pingone
resource_type: test_resource
api_type: TestAPI
name: Test
short_name: test
version: 1.0
enabled: false`,
			expectedTrue:  false,
			expectedFalse: true,
			expectedNil:   false,
		},
		{
			name: "enabled omitted",
			yaml: `platform: pingone
resource_type: test_resource
api_type: TestAPI
name: Test
short_name: test
version: 1.0`,
			expectedTrue:  false,
			expectedFalse: false,
			expectedNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var metadata ResourceMetadata
			err := yaml.Unmarshal([]byte(tt.yaml), &metadata)
			assert.NoError(t, err)

			if tt.expectedNil {
				assert.Nil(t, metadata.Enabled, "Enabled should be nil when omitted")
			} else if tt.expectedTrue {
				assert.NotNil(t, metadata.Enabled, "Enabled should not be nil")
				assert.True(t, *metadata.Enabled, "Enabled should be true")
			} else if tt.expectedFalse {
				assert.NotNil(t, metadata.Enabled, "Enabled should not be nil")
				assert.False(t, *metadata.Enabled, "Enabled should be false")
			}
		})
	}
}

func TestResourceMetadataEnabledFieldRoundTrip(t *testing.T) {
	tests := []struct {
		name      string
		metadata  ResourceMetadata
		expectNil bool
	}{
		{
			name: "enabled true",
			metadata: ResourceMetadata{
				Platform:     "pingone",
				ResourceType: "test",
				APIType:      "Test",
				Name:         "Test",
				ShortName:    "test",
				Version:      "1.0",
				Enabled:      boolPtr(true),
			},
			expectNil: false,
		},
		{
			name: "enabled false",
			metadata: ResourceMetadata{
				Platform:     "pingone",
				ResourceType: "test",
				APIType:      "Test",
				Name:         "Test",
				ShortName:    "test",
				Version:      "1.0",
				Enabled:      boolPtr(false),
			},
			expectNil: false,
		},
		{
			name: "enabled nil",
			metadata: ResourceMetadata{
				Platform:     "pingone",
				ResourceType: "test",
				APIType:      "Test",
				Name:         "Test",
				ShortName:    "test",
				Version:      "1.0",
				Enabled:      nil,
			},
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to YAML
			data, err := yaml.Marshal(tt.metadata)
			assert.NoError(t, err)

			// Unmarshal back
			var decoded ResourceMetadata
			err = yaml.Unmarshal(data, &decoded)
			assert.NoError(t, err)

			if tt.expectNil {
				assert.Nil(t, decoded.Enabled)
			} else {
				assert.NotNil(t, decoded.Enabled)
				assert.Equal(t, *tt.metadata.Enabled, *decoded.Enabled)
			}
		})
	}
}

// boolPtr is a helper function to create a pointer to a bool
func boolPtr(b bool) *bool {
	return &b
}
