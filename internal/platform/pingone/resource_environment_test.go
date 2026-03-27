package pingone

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pingidentity/pingone-go-client/pingone"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnvironmentResourceRegistered verifies the environment handler is in the dispatch table.
func TestEnvironmentResourceRegistered(t *testing.T) {
	assert.True(t, isSupported("pingone_environment"))
}

// TestEnvironmentResourceHandlerFunctions verifies list and get functions are set.
func TestEnvironmentResourceHandlerFunctions(t *testing.T) {
	h, ok := resourceHandlers["pingone_environment"]
	assert.True(t, ok)
	assert.NotNil(t, h.list)
	assert.NotNil(t, h.get)
}

// --- toEnvironmentData projection tests using real SDK types ---

// newTestEnvironmentResponse creates a minimal EnvironmentResponse for testing.
func newTestEnvironmentResponse(id uuid.UUID, name string) *pingone.EnvironmentResponse {
	orgID := uuid.New()
	now := time.Now()
	return pingone.NewEnvironmentResponse(
		name,
		pingone.EnvironmentRegionCode("NA"),
		pingone.EnvironmentTypeValue("PRODUCTION"),
		now,
		now,
		id,
		*pingone.NewResourceRelationshipReadOnly(orgID),
	)
}

func TestToEnvironmentData_FullConversion(t *testing.T) {
	envID := uuid.New()
	orgID := uuid.New()
	licID := uuid.New()
	deployID := uuid.New()
	now := time.Now()
	desc := "Test environment"
	solType := "CUSTOMER"
	consoleHref := "https://console.example.com"

	envResp := pingone.NewEnvironmentResponse(
		"Full Test Env",
		pingone.EnvironmentRegionCode("EU"),
		pingone.EnvironmentTypeValue("SANDBOX"),
		now, now,
		envID,
		*pingone.NewResourceRelationshipReadOnly(orgID),
	)
	envResp.Description = &desc
	envResp.License = pingone.NewEnvironmentLicense(licID)

	bomResp := pingone.NewEnvironmentBillOfMaterialsResponse()
	bomResp.SolutionType = &solType
	bomResp.Products = []pingone.EnvironmentBillOfMaterialsProduct{
		{
			Type: pingone.EnvironmentBillOfMaterialsProductType("PING_ONE_BASE"),
			Console: &pingone.EnvironmentBillOfMaterialsProductConsole{
				Href: &consoleHref,
			},
			Deployment: pingone.NewResourceRelationshipReadOnly(deployID),
			Bookmarks: []pingone.EnvironmentBillOfMaterialsProductBookmark{
				{Href: "https://bm.example.com", Name: "Portal"},
			},
			Tags: []string{"core", "base"},
		},
	}

	result := toEnvironmentData(envResp, bomResp)

	require.NotNil(t, result)
	assert.Equal(t, envID.String(), result.Id)
	assert.Equal(t, "Full Test Env", result.Name)
	assert.Equal(t, "Test environment", *result.Description)
	assert.Equal(t, "SANDBOX", result.Type)
	assert.Equal(t, "EU", result.Region)
	assert.Equal(t, licID.String(), result.License.Id)
	assert.Equal(t, orgID.String(), result.Organization.Id)

	require.NotNil(t, result.BillOfMaterials)
	assert.Equal(t, "CUSTOMER", *result.BillOfMaterials.SolutionType)
	require.Len(t, result.BillOfMaterials.Products, 1)

	prod := result.BillOfMaterials.Products[0]
	assert.Equal(t, "PING_ONE_BASE", prod.Type)
	require.NotNil(t, prod.Console)
	assert.Equal(t, consoleHref, *prod.Console.Href)
	require.NotNil(t, prod.Deployment)
	assert.Equal(t, deployID.String(), prod.Deployment.Id)
	require.Len(t, prod.Bookmarks, 1)
	assert.Equal(t, "Portal", prod.Bookmarks[0].Name)
	assert.Equal(t, "https://bm.example.com", prod.Bookmarks[0].Href)
	require.Len(t, prod.Tags, 2)
	assert.Equal(t, "core", prod.Tags[0])
	assert.Equal(t, "base", prod.Tags[1])
}

func TestToEnvironmentData_NilLicense(t *testing.T) {
	envID := uuid.New()
	envResp := newTestEnvironmentResponse(envID, "No License")
	envResp.License = nil

	result := toEnvironmentData(envResp, nil)

	require.NotNil(t, result)
	assert.Equal(t, envID.String(), result.Id)
	assert.Equal(t, "", result.License.Id, "nil License should produce empty License.Id")
}

func TestToEnvironmentData_NilDescription(t *testing.T) {
	envResp := newTestEnvironmentResponse(uuid.New(), "No Desc")
	envResp.Description = nil

	result := toEnvironmentData(envResp, nil)

	require.NotNil(t, result)
	assert.Nil(t, result.Description)
}

func TestToEnvironmentData_NilBOM(t *testing.T) {
	envResp := newTestEnvironmentResponse(uuid.New(), "No BOM")

	result := toEnvironmentData(envResp, nil)

	require.NotNil(t, result)
	assert.Nil(t, result.BillOfMaterials)
}

func TestToEnvironmentData_EmptyBOM(t *testing.T) {
	envResp := newTestEnvironmentResponse(uuid.New(), "Empty BOM")
	bomResp := pingone.NewEnvironmentBillOfMaterialsResponse()

	result := toEnvironmentData(envResp, bomResp)

	require.NotNil(t, result)
	require.NotNil(t, result.BillOfMaterials)
	assert.Len(t, result.BillOfMaterials.Products, 0)
}

func TestToEnvironmentData_UUIDConversion(t *testing.T) {
	envID := uuid.New()
	orgID := uuid.New()
	licID := uuid.New()
	now := time.Now()

	envResp := pingone.NewEnvironmentResponse(
		"UUID Test",
		pingone.EnvironmentRegionCode("AP"),
		pingone.EnvironmentTypeValue("PRODUCTION"),
		now, now,
		envID,
		*pingone.NewResourceRelationshipReadOnly(orgID),
	)
	envResp.License = pingone.NewEnvironmentLicense(licID)

	result := toEnvironmentData(envResp, nil)

	assert.Equal(t, envID.String(), result.Id, "env UUID should convert to string")
	assert.Equal(t, orgID.String(), result.Organization.Id, "org UUID should convert to string")
	assert.Equal(t, licID.String(), result.License.Id, "license UUID should convert to string")
}

func TestToEnvironmentData_EnumConversion(t *testing.T) {
	now := time.Now()
	envResp := pingone.NewEnvironmentResponse(
		"Enum Test",
		pingone.EnvironmentRegionCode("CA"),
		pingone.EnvironmentTypeValue("SANDBOX"),
		now, now,
		uuid.New(),
		*pingone.NewResourceRelationshipReadOnly(uuid.New()),
	)

	result := toEnvironmentData(envResp, nil)

	assert.Equal(t, "SANDBOX", result.Type, "EnvironmentTypeValue should convert to string")
	assert.Equal(t, "CA", result.Region, "EnvironmentRegionCode should convert to string")
}

func TestToEnvironmentData_MultipleProducts(t *testing.T) {
	envResp := newTestEnvironmentResponse(uuid.New(), "Multi Product")
	deployID := uuid.New()

	bomResp := pingone.NewEnvironmentBillOfMaterialsResponse()
	bomResp.Products = []pingone.EnvironmentBillOfMaterialsProduct{
		{Type: pingone.EnvironmentBillOfMaterialsProductType("PING_ONE_BASE")},
		{
			Type:       pingone.EnvironmentBillOfMaterialsProductType("PING_ONE_DAVINCI"),
			Deployment: pingone.NewResourceRelationshipReadOnly(deployID),
		},
		{
			Type: pingone.EnvironmentBillOfMaterialsProductType("PING_ONE_MFA"),
			Tags: []string{"mfa", "security"},
		},
	}

	result := toEnvironmentData(envResp, bomResp)

	require.NotNil(t, result.BillOfMaterials)
	require.Len(t, result.BillOfMaterials.Products, 3)
	assert.Equal(t, "PING_ONE_BASE", result.BillOfMaterials.Products[0].Type)
	assert.Nil(t, result.BillOfMaterials.Products[0].Console)
	assert.Nil(t, result.BillOfMaterials.Products[0].Deployment)

	assert.Equal(t, "PING_ONE_DAVINCI", result.BillOfMaterials.Products[1].Type)
	assert.Equal(t, deployID.String(), result.BillOfMaterials.Products[1].Deployment.Id)

	assert.Equal(t, "PING_ONE_MFA", result.BillOfMaterials.Products[2].Type)
	assert.Len(t, result.BillOfMaterials.Products[2].Tags, 2)
}

func TestToEnvironmentData_NilConsoleAndDeployment(t *testing.T) {
	envResp := newTestEnvironmentResponse(uuid.New(), "Nil Ptrs")
	bomResp := pingone.NewEnvironmentBillOfMaterialsResponse()
	bomResp.Products = []pingone.EnvironmentBillOfMaterialsProduct{
		{
			Type:       pingone.EnvironmentBillOfMaterialsProductType("PING_ONE_PROTECT"),
			Console:    nil,
			Deployment: nil,
		},
	}

	result := toEnvironmentData(envResp, bomResp)

	require.Len(t, result.BillOfMaterials.Products, 1)
	assert.Nil(t, result.BillOfMaterials.Products[0].Console)
	assert.Nil(t, result.BillOfMaterials.Products[0].Deployment)
}

func TestToEnvironmentData_EmptyBookmarksAndTags(t *testing.T) {
	envResp := newTestEnvironmentResponse(uuid.New(), "Empty Collections")
	bomResp := pingone.NewEnvironmentBillOfMaterialsResponse()
	bomResp.Products = []pingone.EnvironmentBillOfMaterialsProduct{
		{
			Type:      pingone.EnvironmentBillOfMaterialsProductType("PING_ONE_VERIFY"),
			Bookmarks: []pingone.EnvironmentBillOfMaterialsProductBookmark{},
			Tags:      []string{},
		},
	}

	result := toEnvironmentData(envResp, bomResp)

	require.Len(t, result.BillOfMaterials.Products, 1)
	assert.Len(t, result.BillOfMaterials.Products[0].Bookmarks, 0)
	assert.Len(t, result.BillOfMaterials.Products[0].Tags, 0)
}
