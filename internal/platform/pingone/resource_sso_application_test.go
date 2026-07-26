package pingone

import (
	"testing"

	"github.com/patrickcping/pingone-go-sdk-v2/management"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── toApplicationData: union unwrapping ─────────────────────────────

func TestToApplicationData_OIDC(t *testing.T) {
	envId := "env-123"
	oidc := management.NewApplicationOIDC(true, "My Web App", management.ENUMAPPLICATIONPROTOCOL_OPENID_CONNECT, management.ENUMAPPLICATIONTYPE_WEB_APP, management.ENUMAPPLICATIONOIDCTOKENAUTHMETHOD_CLIENT_SECRET_BASIC)
	oidc.SetId("app-1")
	oidc.SetEnvironment(management.ObjectEnvironment{Id: &envId})
	oidc.GrantTypes = []management.EnumApplicationOIDCGrantType{management.ENUMAPPLICATIONOIDCGRANTTYPE_AUTHORIZATION_CODE}
	oidc.RedirectUris = []string{"https://example.com/callback"}

	data, ok := toApplicationData(oidc)
	require.True(t, ok)
	require.NotNil(t, data)

	assert.Equal(t, "app-1", data.Id)
	assert.Equal(t, envId, data.EnvironmentId)
	assert.Equal(t, "My Web App", data.Name)
	assert.True(t, data.Enabled)
	require.NotNil(t, data.OIDC)
	assert.Equal(t, "WEB_APP", data.OIDC.Type)
	assert.Equal(t, []string{"AUTHORIZATION_CODE"}, data.OIDC.GrantTypes)
	assert.Equal(t, []string{"https://example.com/callback"}, data.OIDC.RedirectUris)
	assert.Equal(t, "CLIENT_SECRET_BASIC", data.OIDC.TokenEndpointAuthMethod)

	// Only OIDC should be populated; the other 3 variants must stay nil so
	// the formatter emits exactly one *_options block.
	assert.Nil(t, data.SAML)
	assert.Nil(t, data.ExternalLink)
	assert.Nil(t, data.WSFED)
}

func TestToApplicationData_OIDC_MobileAppAndSigning(t *testing.T) {
	oidc := management.NewApplicationOIDC(true, "Native App", management.ENUMAPPLICATIONPROTOCOL_OPENID_CONNECT, management.ENUMAPPLICATIONTYPE_NATIVE_APP, management.ENUMAPPLICATIONOIDCTOKENAUTHMETHOD_NONE)
	bundleID := "com.example.app"
	oidc.Mobile = &management.ApplicationOIDCAllOfMobile{BundleId: &bundleID}
	oidc.Signing = &management.ApplicationOIDCAllOfSigning{
		KeyRotationPolicy: management.ApplicationOIDCAllOfSigningKeyRotationPolicy{Id: "krp-1"},
	}

	data, ok := toApplicationData(oidc)
	require.True(t, ok)
	require.NotNil(t, data.OIDC)
	require.NotNil(t, data.OIDC.MobileApp)
	assert.Equal(t, "com.example.app", *data.OIDC.MobileApp.BundleId)
	require.NotNil(t, data.OIDC.Signing)
	assert.Equal(t, "krp-1", *data.OIDC.Signing.KeyRotationPolicyId)
}

func TestToApplicationData_SAML(t *testing.T) {
	saml := management.NewApplicationSAML(true, "My SAML App", management.ENUMAPPLICATIONPROTOCOL_SAML, management.ENUMAPPLICATIONTYPE_WEB_APP, []string{"https://sp.example.com/acs"}, 3600, "sp:entity:example")
	saml.SetId("app-2")

	data, ok := toApplicationData(saml)
	require.True(t, ok)
	require.NotNil(t, data)
	require.NotNil(t, data.SAML)

	assert.Equal(t, "app-2", data.Id)
	assert.Equal(t, []string{"https://sp.example.com/acs"}, data.SAML.AcsUrls)
	assert.Equal(t, int32(3600), data.SAML.AssertionDuration)
	assert.Equal(t, "sp:entity:example", data.SAML.SpEntityId)

	assert.Nil(t, data.OIDC)
	assert.Nil(t, data.ExternalLink)
	assert.Nil(t, data.WSFED)
}

func TestToApplicationData_SAML_NestedBlocks(t *testing.T) {
	saml := management.NewApplicationSAML(true, "SAML with signing", management.ENUMAPPLICATIONPROTOCOL_SAML, management.ENUMAPPLICATIONTYPE_WEB_APP, []string{"https://sp.example.com/acs"}, 3600, "sp:entity:example")
	saml.IdpSigning = &management.ApplicationSAMLAllOfIdpSigning{
		Key: management.ApplicationSAMLAllOfIdpSigningKey{Id: "key-1"},
	}
	saml.SpVerification = &management.ApplicationSAMLAllOfSpVerification{
		Certificates: []management.ApplicationSAMLAllOfSpVerificationCertificates{{Id: "cert-1"}, {Id: "cert-2"}},
	}
	saml.SpEncryption = &management.ApplicationSAMLAllOfSpEncryption{
		Algorithm:   management.ENUMCERTIFICATEKEYENCRYPTIONALGORITHM_AES_256,
		Certificate: management.ApplicationSAMLAllOfSpEncryptionCertificate{Id: "enc-cert-1"},
	}

	data, ok := toApplicationData(saml)
	require.True(t, ok)
	require.NotNil(t, data.SAML)

	require.NotNil(t, data.SAML.IdpSigningKey)
	assert.Equal(t, "key-1", data.SAML.IdpSigningKey.KeyId)

	require.NotNil(t, data.SAML.SpVerification)
	assert.Equal(t, []string{"cert-1", "cert-2"}, data.SAML.SpVerification.CertificateIds)

	require.NotNil(t, data.SAML.SpEncryption)
	assert.Equal(t, "enc-cert-1", data.SAML.SpEncryption.Certificate.Id)
}

func TestToApplicationData_ExternalLink(t *testing.T) {
	link := management.NewApplicationExternalLink(true, "My Link", management.ENUMAPPLICATIONPROTOCOL_EXTERNAL_LINK, management.ENUMAPPLICATIONTYPE_PORTAL_LINK_APP, "https://example.com")
	link.SetId("app-3")

	data, ok := toApplicationData(link)
	require.True(t, ok)
	require.NotNil(t, data.ExternalLink)
	assert.Equal(t, "https://example.com", data.ExternalLink.HomePageUrl)

	assert.Nil(t, data.OIDC)
	assert.Nil(t, data.SAML)
	assert.Nil(t, data.WSFED)
}

func TestToApplicationData_WSFED(t *testing.T) {
	idpSigning := management.ApplicationWSFEDAllOfIdpSigning{
		Algorithm: management.ENUMAPPLICATIONWSFEDIDPSIGNINGALGORITHM_SHA256WITH_RSA,
		Key:       management.ApplicationWSFEDAllOfIdpSigningKey{Id: "key-1"},
	}
	wsfed := management.NewApplicationWSFED(true, "My WSFed App", management.ENUMAPPLICATIONPROTOCOL_WS_FED, management.ENUMAPPLICATIONTYPE_WEB_APP, "my.domain.com", idpSigning, "https://reply.example.com")
	wsfed.SetId("app-4")

	data, ok := toApplicationData(wsfed)
	require.True(t, ok)
	require.NotNil(t, data.WSFED)
	assert.Equal(t, "my.domain.com", data.WSFED.DomainName)
	assert.Equal(t, "https://reply.example.com", data.WSFED.ReplyUrl)
	assert.Equal(t, "key-1", data.WSFED.IdpSigningKey.KeyId)
	require.NotNil(t, data.WSFED.IdpSigningKey.Algorithm)
	assert.Equal(t, "SHA256withRSA", *data.WSFED.IdpSigningKey.Algorithm)

	assert.Nil(t, data.OIDC)
	assert.Nil(t, data.SAML)
	assert.Nil(t, data.ExternalLink)
}

func TestToApplicationData_WSFED_Kerberos(t *testing.T) {
	idpSigning := management.ApplicationWSFEDAllOfIdpSigning{
		Algorithm: management.ENUMAPPLICATIONWSFEDIDPSIGNINGALGORITHM_SHA256WITH_RSA,
		Key:       management.ApplicationWSFEDAllOfIdpSigningKey{Id: "key-1"},
	}
	wsfed := management.NewApplicationWSFED(true, "WSFed with Kerberos", management.ENUMAPPLICATIONPROTOCOL_WS_FED, management.ENUMAPPLICATIONTYPE_WEB_APP, "my.domain.com", idpSigning, "https://reply.example.com")
	userTypeID := "ut-1"
	wsfed.Kerberos = &management.ApplicationWSFEDAllOfKerberos{
		Gateways: []management.ApplicationWSFEDAllOfKerberosGateways{
			{
				Id:       "gw-1",
				Type:     management.ENUMAPPLICATIONWSFEDKERBEROSGATEWAYTYPE_LDAP,
				UserType: management.ApplicationWSFEDAllOfKerberosUserType{Id: &userTypeID},
			},
		},
	}

	data, ok := toApplicationData(wsfed)
	require.True(t, ok)
	require.NotNil(t, data.WSFED.Kerberos)
	require.Len(t, data.WSFED.Kerberos.Gateways, 1)
	gw := data.WSFED.Kerberos.Gateways[0]
	assert.Equal(t, "gw-1", gw.Id)
	require.NotNil(t, gw.UserType.Id)
	assert.Equal(t, "ut-1", *gw.UserType.Id)
}

// ── toApplicationData: unsupported system application types ─────────

func TestToApplicationData_SkipsAdminConsole(t *testing.T) {
	adminConsole := management.NewApplicationPingOneAdminConsole()
	data, ok := toApplicationData(adminConsole)
	assert.False(t, ok)
	assert.Nil(t, data)
}

func TestToApplicationData_SkipsPortal(t *testing.T) {
	portal := management.NewApplicationPingOnePortal(true, "PingOne Portal", management.ENUMAPPLICATIONPROTOCOL_OPENID_CONNECT, management.ENUMAPPLICATIONTYPE_PING_ONE_PORTAL, management.ENUMAPPLICATIONOIDCTOKENAUTHMETHOD_NONE, true)
	data, ok := toApplicationData(portal)
	assert.False(t, ok)
	assert.Nil(t, data)
}

func TestToApplicationData_SkipsSelfService(t *testing.T) {
	selfService := management.NewApplicationPingOneSelfService(true, "PingOne Self Service", management.ENUMAPPLICATIONPROTOCOL_OPENID_CONNECT, management.ENUMAPPLICATIONTYPE_PING_ONE_SELF_SERVICE, management.ENUMAPPLICATIONOIDCTOKENAUTHMETHOD_NONE, true)
	data, ok := toApplicationData(selfService)
	assert.False(t, ok)
	assert.Nil(t, data)
}

// ── toApplicationData: nil / edge cases ──────────────────────────────

func TestToApplicationData_NilInput(t *testing.T) {
	data, ok := toApplicationData(nil)
	assert.False(t, ok)
	assert.Nil(t, data)
}

func TestToApplicationData_UnknownType(t *testing.T) {
	data, ok := toApplicationData("not an application")
	assert.False(t, ok)
	assert.Nil(t, data)
}

// ── describeSkippedApplication ────────────────────────────────────────

func TestDescribeSkippedApplication(t *testing.T) {
	assert.Equal(t, "(PingOne Admin Console)", describeSkippedApplication(management.NewApplicationPingOneAdminConsole()))

	portal := management.NewApplicationPingOnePortal(true, "Portal", management.ENUMAPPLICATIONPROTOCOL_OPENID_CONNECT, management.ENUMAPPLICATIONTYPE_PING_ONE_PORTAL, management.ENUMAPPLICATIONOIDCTOKENAUTHMETHOD_NONE, true)
	portal.SetId("portal-1")
	assert.Equal(t, "portal-1", describeSkippedApplication(portal))

	assert.Equal(t, "(unknown)", describeSkippedApplication("garbage"))
}

// ── commonFromBase: shared field projection ──────────────────────────

func TestCommonFromBase_TagsAndAccessControl(t *testing.T) {
	tags := []management.EnumApplicationTags{management.ENUMAPPLICATIONTAGS_PING_FED_CONNECTION_INTEGRATION}
	groupID := "group-1"
	ac := &management.ApplicationAccessControl{
		Group: &management.ApplicationAccessControlGroup{
			Type:   management.ENUMAPPLICATIONACCESSCONTROLGROUPTYPE_ANY_GROUP,
			Groups: []management.ApplicationAccessControlGroupGroupsInner{{Id: groupID}},
		},
	}
	icon := &management.ApplicationIcon{Id: "icon-1", Href: "https://example.com/icon.png"}

	data := commonFromBase("id-1", "env-1", "name-1", nil, true, nil, nil, tags, ac, icon)

	assert.Equal(t, []string{"PING_FED_CONNECTION_INTEGRATION"}, data.Tags)
	require.NotNil(t, data.AccessControlGroupOptions)
	assert.Equal(t, "ANY_GROUP", data.AccessControlGroupOptions.Type)
	assert.Equal(t, []string{groupID}, data.AccessControlGroupOptions.Groups)
	require.NotNil(t, data.Icon)
	assert.Equal(t, "icon-1", data.Icon.Id)
}

func TestCommonFromBase_NilAccessControlAndIcon(t *testing.T) {
	data := commonFromBase("id-1", "env-1", "name-1", nil, true, nil, nil, nil, nil, nil)
	assert.Nil(t, data.AccessControlGroupOptions)
	assert.Nil(t, data.AccessControlRoleType)
	assert.Nil(t, data.Icon)
	assert.Empty(t, data.Tags)
}

func TestCommonFromBase_RoleAccessControl(t *testing.T) {
	ac := &management.ApplicationAccessControl{
		Role: &management.ApplicationAccessControlRole{Type: management.ENUMAPPLICATIONACCESSCONTROLTYPE_ADMIN_USERS_ONLY},
	}
	data := commonFromBase("id-1", "env-1", "name-1", nil, true, nil, nil, nil, ac, nil)
	require.NotNil(t, data.AccessControlRoleType)
	assert.Equal(t, "ADMIN_USERS_ONLY", *data.AccessControlRoleType)
	assert.Nil(t, data.AccessControlGroupOptions)
}

// ── corsFromSDK ───────────────────────────────────────────────────────

func TestCorsFromSDK_Nil(t *testing.T) {
	assert.Nil(t, corsFromSDK(nil))
}

func TestCorsFromSDK_WithOrigins(t *testing.T) {
	cors := &management.ApplicationCorsSettings{
		Behavior: management.ENUMAPPLICATIONCORSSETTINGSBEHAVIOR_SPECIFIC_ORIGINS,
		Origins:  []string{"https://example.com"},
	}
	result := corsFromSDK(cors)
	require.NotNil(t, result)
	assert.Equal(t, "ALLOW_SPECIFIC_ORIGINS", result.Behavior)
	assert.Equal(t, []string{"https://example.com"}, result.Origins)
}

// ── dispatch registration ────────────────────────────────────────────

func TestPingOneApplicationRegistered(t *testing.T) {
	assert.True(t, isSupported("pingone_application"), "expected pingone_application to be registered")
}

// TestDaVinciApplicationStillRegistered is a regression guard: adding
// pingone_application (this resource) must not disturb the pre-existing
// pingone_davinci_application registration in resource_application.go.
func TestDaVinciApplicationStillRegistered(t *testing.T) {
	assert.True(t, isSupported("pingone_davinci_application"), "expected pingone_davinci_application to remain registered")
}

func TestSSOAndDaVinciApplicationAreDistinctResourceTypes(t *testing.T) {
	ssoHandler, ok := resourceHandlers["pingone_application"]
	require.True(t, ok)
	davinciHandler, ok := resourceHandlers["pingone_davinci_application"]
	require.True(t, ok)

	assert.NotNil(t, ssoHandler.list)
	assert.NotNil(t, ssoHandler.get)
	assert.NotNil(t, davinciHandler.list)
	assert.NotNil(t, davinciHandler.get)
}
