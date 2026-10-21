package review

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
)

func baseDef() *schema.ResourceDefinition {
	enabled := true
	return &schema.ResourceDefinition{
		Metadata: schema.ResourceMetadata{
			Platform:     "pingone",
			ResourceType: "pingone_widget",
			APIType:      "Widget",
			Name:         "PingOne Widget",
			ShortName:    "widget",
			Version:      "1.0",
			Enabled:      &enabled,
		},
		API: schema.APIDefinition{
			SDKPackage:     "github.com/patrickcping/pingone-go-sdk-v2/management",
			SDKType:        "Widget",
			ListMethod:     "WidgetsApi.ReadAllWidgets",
			GetMethod:      "WidgetsApi.ReadOneWidget",
			IDField:        "id",
			NameField:      "name",
			PaginationType: "cursor",
		},
		Attributes: []schema.AttributeDefinition{
			{Name: "ID", TerraformName: "id", Type: "string", SourcePath: "Id", Computed: true},
			{Name: "EnvironmentID", TerraformName: "environment_id", Type: "string", SourcePath: "Environment.Id",
				Required: true, ReferencesType: "pingone_environment", ReferenceField: "id"},
			{Name: "Name", TerraformName: "name", Type: "string", SourcePath: "Name", Required: true},
			{Name: "Description", TerraformName: "description", Type: "string", SourcePath: "Description"},
			{
				Name: "Settings", TerraformName: "settings", Type: "object", SourcePath: "Settings",
				NestedAttributes: []schema.AttributeDefinition{
					{Name: "Color", TerraformName: "color", Type: "string", SourcePath: "Color"},
				},
			},
		},
		Dependencies: schema.DependencyDefinition{
			ImportIDFormat: "{{env_id}}/{{id}}",
		},
	}
}

func cloneDef(d *schema.ResourceDefinition) *schema.ResourceDefinition {
	// Deep enough copy for tests: mutate via returned pointer.
	cp := *d
	cp.Metadata = d.Metadata
	cp.API = d.API
	cp.Dependencies = d.Dependencies
	cp.Variables = d.Variables
	cp.Attributes = append([]schema.AttributeDefinition(nil), d.Attributes...)
	for i := range cp.Attributes {
		cp.Attributes[i].NestedAttributes = append([]schema.AttributeDefinition(nil), d.Attributes[i].NestedAttributes...)
	}
	return &cp
}

func findChange(changes []FieldChange, section, attr, kind, field string) *FieldChange {
	for i := range changes {
		c := &changes[i]
		if c.Section == section && c.Attr == attr && c.Kind == kind && (field == "" || c.Field == field) {
			return c
		}
	}
	return nil
}

func TestDiffDefinitions_NoChanges(t *testing.T) {
	base := baseDef()
	pr := cloneDef(base)
	changes := DiffDefinitions(base, pr)
	assert.Empty(t, changes)
}

func TestDiffDefinitions_AttributeChanged(t *testing.T) {
	base := baseDef()
	pr := cloneDef(base)
	pr.Attributes[3].Sensitive = true // description becomes sensitive

	changes := DiffDefinitions(base, pr)
	require.Len(t, changes, 1)
	c := findChange(changes, sectionAttributes, "description", "changed", "sensitive")
	require.NotNil(t, c)
	assert.Equal(t, "false", c.Base)
	assert.Equal(t, "true", c.PR)
}

func TestDiffDefinitions_AttributeAddedAndRemoved(t *testing.T) {
	base := baseDef()
	pr := cloneDef(base)

	// Remove description (index 3).
	pr.Attributes = append(pr.Attributes[:3], pr.Attributes[4:]...)
	// Add a new attribute.
	pr.Attributes = append(pr.Attributes, schema.AttributeDefinition{
		Name: "ExternalID", TerraformName: "external_id", Type: "string", SourcePath: "ExternalId",
	})

	changes := DiffDefinitions(base, pr)
	require.NotNil(t, findChange(changes, sectionAttributes, "description", "removed", ""))
	require.NotNil(t, findChange(changes, sectionAttributes, "external_id", "added", ""))

	// The added attribute's set fields are itemized.
	added := findChange(changes, sectionAttributes, "external_id", "added", "source_path")
	require.NotNil(t, added)
	assert.Equal(t, "ExternalId", added.PR)
}

func TestDiffDefinitions_NestedAttributeChanged(t *testing.T) {
	base := baseDef()
	pr := cloneDef(base)
	pr.Attributes[4].NestedAttributes[0].Computed = true

	changes := DiffDefinitions(base, pr)
	c := findChange(changes, sectionAttributes, "settings.color", "changed", "computed")
	require.NotNil(t, c)
	assert.Equal(t, "false", c.Base)
	assert.Equal(t, "true", c.PR)
}

func TestDiffDefinitions_APIChanged(t *testing.T) {
	base := baseDef()
	pr := cloneDef(base)
	pr.API.GetMethod = ""

	changes := DiffDefinitions(base, pr)
	c := findChange(changes, sectionAPI, "", "removed", "get_method")
	require.NotNil(t, c)
	assert.Equal(t, "WidgetsApi.ReadOneWidget", c.Base)
}

func TestDiffDefinitions_ReferenceTypeChanged(t *testing.T) {
	base := baseDef()
	pr := cloneDef(base)
	pr.Attributes[1].ReferencesType = "pingone_organization"

	changes := DiffDefinitions(base, pr)
	c := findChange(changes, sectionAttributes, "environment_id", "changed", "references_type")
	require.NotNil(t, c)
	assert.Equal(t, "pingone_environment", c.Base)
	assert.Equal(t, "pingone_organization", c.PR)
}

func TestDiffDefinitions_DependsOnChanged(t *testing.T) {
	base := baseDef()
	pr := cloneDef(base)
	pr.Dependencies.DependsOn = append(pr.Dependencies.DependsOn, schema.DependencyRule{
		ResourceType: "pingone_population",
		FieldPath:    "population_id",
	})

	changes := DiffDefinitions(base, pr)
	c := findChange(changes, sectionDependencies, "pingone_population", "added", "depends_on")
	require.NotNil(t, c)
	assert.Contains(t, c.PR, "population_id")
}

func TestDiffDefinitions_EnabledTriState(t *testing.T) {
	base := baseDef()
	pr := cloneDef(base)
	falseVal := false
	pr.Metadata.Enabled = &falseVal

	changes := DiffDefinitions(base, pr)
	c := findChange(changes, sectionMetadata, "", "changed", "enabled")
	require.NotNil(t, c)
	assert.Equal(t, "true", c.Base)
	assert.Equal(t, "false", c.PR)
}

func TestDiffDefinitions_DeterministicOrder(t *testing.T) {
	base := baseDef()
	pr := cloneDef(base)
	pr.Attributes[3].Sensitive = true
	pr.Attributes[4].NestedAttributes[0].Computed = true
	pr.API.PaginationType = "none"

	changes := DiffDefinitions(base, pr)
	require.Len(t, changes, 3)
	// Stable ordering: api section sorts before attributes.
	assert.Equal(t, sectionAPI, changes[0].Section)
}
