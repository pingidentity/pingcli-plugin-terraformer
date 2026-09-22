package core_test

import (
	"testing"

	"github.com/google/uuid"
	pingone "github.com/pingidentity/pingone-go-client/pingone"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/core"
	hclformatter "github.com/pingidentity/pingcli-plugin-terraformer/internal/formatters/hcl"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
)

// Tests in this file validate the issue #125 audit: every computed attribute
// the Terraform provider exposes and the API returns must be declared in the
// YAML definition, so list-outputs/--output-attribute can enumerate it.
// Computed-only attributes must never render into emitted config (the
// computed-skip guard), so each test also asserts the HCL stays clean.

// davinciRegistry loads the real definitions directory.
func davinciRegistry(t *testing.T) *schema.Registry {
	t.Helper()
	registry := schema.NewRegistry()
	require.NoError(t, registry.LoadPlatform("../../definitions", "pingone"))
	return registry
}

// formatComputedProbe renders a processed resource through the HCL formatter
// with dependencies skipped, returning the HCL string.
func formatComputedProbe(t *testing.T, registry *schema.Registry, resourceType string, result *core.ResourceData) string {
	t.Helper()
	def, err := registry.Get(resourceType)
	require.NoError(t, err)
	formatter := hclformatter.NewFormatter()
	hcl, err := formatter.Format(result, def, hclformatter.FormatOptions{SkipDependencies: true, EnvironmentID: "00000000-0000-0000-0000-000000000001"})
	require.NoError(t, err)
	return hcl
}

// environmentUUID is the fake environment ID used by these tests.
var environmentUUID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// testHALLink builds a JSONHALLink with an href.
func testHALLink(href string) pingone.JSONHALLink {
	return *pingone.NewJSONHALLink(href)
}

// TestDaVinciApplicationComputedOAuthClientSecret verifies that
// oauth.client_secret — computed + sensitive on the provider, returned by the
// API in the oauth response — is extracted into the processed attributes and
// enumerable as an output path, yet never rendered into the emitted HCL
// configuration (computed-skip guard). See #125.
func TestDaVinciApplicationComputedOAuthClientSecret(t *testing.T) {
	registry := davinciRegistry(t)
	p := core.NewProcessor(registry)

	app := pingone.NewDaVinciApplicationResponse(
		*pingone.NewDaVinciApplicationResponseLinks(
			testHALLink("https://example.com/self"),
			testHALLink("https://example.com/environment"),
			testHALLink("https://example.com/flowPolicies"),
			testHALLink("https://example.com/rotateKey"),
			testHALLink("https://example.com/rotateSecret"),
		),
		*pingone.NewDaVinciApplicationResponseApiKey(true, "ak-value-123"),
		*pingone.NewResourceRelationshipReadOnly(environmentUUID),
		"app-1",
		"Test App",
		func() pingone.DaVinciApplicationResponseOAuth {
			o := pingone.NewDaVinciApplicationResponseOAuth("secret-value-456")
			o.SetGrantTypes([]pingone.DaVinciApplicationResponseOAuthGrantType{})
			return *o
		}(),
	)

	result, err := p.ProcessResource("pingone_davinci_application", app)
	require.NoError(t, err)

	oauth, ok := result.Attributes["oauth"].(map[string]interface{})
	require.True(t, ok, "oauth should be a map")
	assert.Equal(t, "secret-value-456", oauth["client_secret"], "client_secret must be extracted so output paths can resolve it")

	// HCL rendering: client_secret is computed-only, so it must NOT appear in
	// the emitted configuration.
	hcl := formatComputedProbe(t, registry, "pingone_davinci_application", result)
	assert.NotContains(t, hcl, "client_secret", "computed-only client_secret must be skipped by the HCL formatter")
}

// TestDaVinciConnectorInstanceComputedMetadata verifies that the entire
// connector instance metadata block (colors/logos/type/vendor) — read-only on
// the provider, returned by the API — is extracted and enumerable as output
// paths, yet never rendered into the emitted HCL configuration. An entirely
// computed object must not even produce an empty "metadata = {}" block. See
// #125.
func TestDaVinciConnectorInstanceComputedMetadata(t *testing.T) {
	registry := davinciRegistry(t)
	p := core.NewProcessor(registry)

	metadata := pingone.NewDaVinciConnectorInstanceResponseMetadata()
	metadata.SetType("authentication")
	metadata.SetVendor("PingIdentity")
	colors := pingone.NewDaVinciConnectorInstanceResponseMetadataColors()
	colors.SetCanvas("#ffffff")
	colors.SetCanvasText("#000000")
	colors.SetDark("#101010")
	metadata.SetColors(*colors)

	instance := pingone.NewDaVinciConnectorInstanceResponse(
		*pingone.NewDaVinciConnectorInstanceResponseLinks(
			testHALLink("https://example.com/environment"),
			testHALLink("https://example.com/self"),
			testHALLink("https://example.com/clone"),
		),
		*pingone.NewResourceRelationshipDaVinciReadOnly("connector-1"),
		*pingone.NewResourceRelationshipReadOnly(environmentUUID),
		"inst-1",
		"Test Instance",
	)
	instance.SetMetadata(*metadata)

	result, err := p.ProcessResource("pingone_davinci_connector_instance", instance)
	require.NoError(t, err)

	md, ok := result.Attributes["metadata"].(map[string]interface{})
	require.True(t, ok, "metadata should be a map")
	assert.Equal(t, "authentication", md["type"])
	assert.Equal(t, "PingIdentity", md["vendor"])
	colorsOut, ok := md["colors"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "#ffffff", colorsOut["canvas"])

	// HCL rendering: no metadata attribute may appear, and no empty
	// metadata block either.
	hcl := formatComputedProbe(t, registry, "pingone_davinci_connector_instance", result)
	assert.NotContains(t, hcl, "metadata", "computed-only metadata block must be skipped entirely by the HCL formatter")
}
