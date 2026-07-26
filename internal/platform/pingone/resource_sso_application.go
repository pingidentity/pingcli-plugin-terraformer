package pingone

import (
	"context"
	"fmt"

	"github.com/patrickcping/pingone-go-sdk-v2/management"
)

// pingone_application is a genuinely complex resource: the management SDK's
// ReadOneApplication200Response (and the EntityArrayEmbedded.Applications list
// item type) is a discriminated union with SEVEN possible variants
// (ApplicationOIDC, ApplicationSAML, ApplicationExternalLink, ApplicationWSFED,
// ApplicationPingOneAdminConsole, ApplicationPingOnePortal,
// ApplicationPingOneSelfService) — exactly one field is non-nil per actual
// application, unwrapped via GetActualInstance().
//
// Scope decision (see issue #67 and the PR description for full rationale):
// the pingone_application Terraform resource's schema only supports FOUR
// declarable application types — oidc_options, saml_options,
// external_link_options, wsfed_options (confirmed via the provider docs:
// https://registry.terraform.io/providers/pingidentity/pingone/latest/docs/resources/application).
// ApplicationPingOneAdminConsole, ApplicationPingOnePortal, and
// ApplicationPingOneSelfService are built-in PingOne system applications
// that are provisioned automatically per environment and have no
// corresponding nested schema block in the provider — they cannot be
// declaratively created or imported via pingone_application. Rather than
// silently emitting a resource block with no application-type-specific
// attributes (which the provider would then reject with "exactly one of
// ... must be defined"), this handler skips them and records a warning via
// c.AddWarning so the omission is visible to the operator, not silent.
//
// This handler performs the union-unwrapping and projects the result into a
// flat applicationData struct with one nilable pointer per supported variant
// (OIDC, SAML, ExternalLink, WSFED). The YAML definition then models each
// variant as a plain "object" attribute with nested_attributes — no
// type_discriminated_block is needed because type_discriminated_block (as
// implemented in internal/core) only maps a runtime value to a single
// primitive/JSON-encoded key, not to a fully-typed nested attribute tree.
// Because exactly one variant pointer is non-nil, the generic engine's
// existing "object without nested_attributes populated" skip (nil map field)
// naturally emits only the one matching options block.
func init() {
	registerResource("pingone_application", resourceHandler{
		list: listSSOApplications,
		get:  getSSOApplication,
	})
}

// applicationData is the projection handed to the processor. Field names
// match the source_path values in definitions/pingone/sso/application.yaml.
type applicationData struct {
	Id                        string
	EnvironmentId             string
	Name                      string
	Description               *string
	Enabled                   bool
	HiddenFromAppPortal       *bool
	LoginPageUrl              *string
	Tags                      []string
	AccessControlGroupOptions *accessControlGroupOptionsData
	AccessControlRoleType     *string
	Icon                      *applicationIconData

	OIDC         *oidcOptionsData
	SAML         *samlOptionsData
	ExternalLink *externalLinkOptionsData
	WSFED        *wsfedOptionsData
}

type applicationIconData struct {
	Id   string
	Href string
}

type accessControlGroupOptionsData struct {
	Type   string
	Groups []string
}

type corsSettingsData struct {
	Behavior string
	Origins  []string
}

type oidcOptionsData struct {
	Type                                          string
	GrantTypes                                    []string
	ResponseTypes                                 []string
	TokenEndpointAuthMethod                       string
	ClientId                                      *string
	HomePageUrl                                   *string
	AdditionalRefreshTokenReplayProtectionEnabled *bool
	AllowWildcardInRedirectUris                   *bool
	CorsSettings                                  *corsSettingsData
	DeviceCustomVerificationUri                   *string
	DevicePathId                                  *string
	DevicePollingInterval                         *int32
	DeviceTimeout                                 *int32
	IdpSignoff                                    *bool
	IncludeX5t                                    *bool
	InitiateLoginUri                              *string
	Jwks                                          *string
	JwksUrl                                       *string
	MobileApp                                     *mobileAppData
	OpSessionCheckEnabled                         *bool
	ParRequirement                                *string
	ParTimeout                                    *int32
	PkceEnforcement                               *string
	PostLogoutRedirectUris                        []string
	RedirectUris                                  []string
	RefreshTokenDuration                          *int32
	RefreshTokenRollingDuration                   *int32
	RefreshTokenRollingGracePeriodDuration        *int32
	RefreshTokenType                              *string
	RequestScopesForMultipleResourcesEnabled      *bool
	RequireSignedRequestObject                    *bool
	Signing                                       *signingData
	SupportUnsignedRequestObject                  *bool
	TargetLinkUri                                 *string
}

type mobileAppData struct {
	BundleId               *string
	PackageName            *string
	HuaweiAppId            *string
	HuaweiPackageName      *string
	PasscodeGracePeriod    *int32
	PasscodeRefreshSeconds *int32
	UniversalAppLink       *string
	IntegrityDetection     *integrityDetectionData
}

type integrityDetectionData struct {
	Enabled           *bool
	ExcludedPlatforms []string
	CacheDuration     *cacheDurationData
	GooglePlay        *googlePlayData
}

type cacheDurationData struct {
	Amount *int32
	Units  *string
}

type googlePlayData struct {
	VerificationType              *string
	DecryptionKey                 *string
	VerificationKey               *string
	ServiceAccountCredentialsJson *string
}

type signingData struct {
	KeyRotationPolicyId *string
}

type samlOptionsData struct {
	Type                        *string
	AcsUrls                     []string
	AssertionDuration           int32
	AssertionSignedEnabled      *bool
	CorsSettings                *corsSettingsData
	DefaultTargetUrl            *string
	EnableRequestedAuthnContext *bool
	HomePageUrl                 *string
	IdpSigningKey               *idpSigningKeyData
	NameIdFormat                *string
	ResponseIsSigned            *bool
	SessionNotOnOrAfterDuration *int32
	SloBinding                  *string
	SloEndpoint                 *string
	SloResponseEndpoint         *string
	SloWindow                   *int32
	SpEncryption                *spEncryptionData
	SpEntityId                  string
	SpVerification              *spVerificationData
	VirtualServerIdSettings     *virtualServerIdSettingsData
}

type idpSigningKeyData struct {
	KeyId     string
	Algorithm *string
}

type spEncryptionData struct {
	Algorithm   string
	Certificate certificateRefData
}

type certificateRefData struct {
	Id string
}

type spVerificationData struct {
	CertificateIds     []string
	AuthnRequestSigned *bool
}

type virtualServerIdSettingsData struct {
	Enabled          *bool
	VirtualServerIds []virtualServerIdEntryData
}

type virtualServerIdEntryData struct {
	VsId    string
	Default *bool
}

type externalLinkOptionsData struct {
	HomePageUrl string
}

type wsfedOptionsData struct {
	Type                        string
	DomainName                  string
	ReplyUrl                    string
	IdpSigningKey               idpSigningKeyData
	AudienceRestriction         *string
	CorsSettings                *corsSettingsData
	SloEndpoint                 *string
	SubjectNameIdentifierFormat *string
	Kerberos                    *wsfedKerberosData
}

type wsfedKerberosData struct {
	Gateways []wsfedKerberosGatewayData
}

type wsfedKerberosGatewayData struct {
	Id       string
	Type     *string
	UserType userTypeRefData
}

type userTypeRefData struct {
	Id *string
}

// toApplicationData unwraps the discriminated union and projects the actual
// instance into applicationData. Returns (nil, false) for the three PingOne
// system application types that have no corresponding Terraform schema block
// (see the doc comment above) — callers should skip these with a warning
// rather than emit a resource block the provider will reject.
func toApplicationData(actual interface{}) (*applicationData, bool) {
	switch v := actual.(type) {
	case *management.ApplicationOIDC:
		return fromOIDC(v), true
	case *management.ApplicationSAML:
		return fromSAML(v), true
	case *management.ApplicationExternalLink:
		return fromExternalLink(v), true
	case *management.ApplicationWSFED:
		return fromWSFED(v), true
	default:
		// ApplicationPingOneAdminConsole, ApplicationPingOnePortal,
		// ApplicationPingOneSelfService, or nil (unrecognized/empty union).
		return nil, false
	}
}

func commonFromBase(id, envId, name string, description *string, enabled bool, hiddenFromAppPortal *bool, loginPageUrl *string, tags []management.EnumApplicationTags, ac *management.ApplicationAccessControl, icon *management.ApplicationIcon) applicationData {
	data := applicationData{
		Id:                  id,
		EnvironmentId:       envId,
		Name:                name,
		Description:         description,
		Enabled:             enabled,
		HiddenFromAppPortal: hiddenFromAppPortal,
		LoginPageUrl:        loginPageUrl,
	}

	if len(tags) > 0 {
		data.Tags = make([]string, 0, len(tags))
		for _, t := range tags {
			data.Tags = append(data.Tags, string(t))
		}
	}

	if ac != nil {
		if role, ok := ac.GetRoleOk(); ok && role != nil {
			roleType := string(role.GetType())
			data.AccessControlRoleType = &roleType
		}
		if group, ok := ac.GetGroupOk(); ok && group != nil {
			groups := group.GetGroups()
			ids := make([]string, 0, len(groups))
			for _, g := range groups {
				ids = append(ids, g.GetId())
			}
			groupType := string(group.GetType())
			data.AccessControlGroupOptions = &accessControlGroupOptionsData{
				Type:   groupType,
				Groups: ids,
			}
		}
	}

	if icon != nil {
		data.Icon = &applicationIconData{Id: icon.GetId(), Href: icon.GetHref()}
	}

	return data
}

func corsFromSDK(c *management.ApplicationCorsSettings) *corsSettingsData {
	if c == nil {
		return nil
	}
	return &corsSettingsData{
		Behavior: string(c.GetBehavior()),
		Origins:  c.GetOrigins(),
	}
}

func fromOIDC(v *management.ApplicationOIDC) *applicationData {
	envId := ""
	if env, ok := v.GetEnvironmentOk(); ok && env != nil {
		envId = env.GetId()
	}
	data := commonFromBase(v.GetId(), envId, v.GetName(), v.Description, v.GetEnabled(), v.HiddenFromAppPortal, v.LoginPageUrl, v.Tags, v.AccessControl, v.Icon)

	// Build as nil (not empty non-nil) when the source is empty, so the
	// processor's isEmptyValue check (which only treats nil slices as
	// "not set") correctly omits the attribute instead of emitting an
	// empty [] that would produce a perpetual plan diff.
	var grantTypes []string
	for _, g := range v.GrantTypes {
		grantTypes = append(grantTypes, string(g))
	}
	var responseTypes []string
	for _, r := range v.ResponseTypes {
		responseTypes = append(responseTypes, string(r))
	}

	oidc := &oidcOptionsData{
		Type:                    string(v.Type),
		GrantTypes:              grantTypes,
		ResponseTypes:           responseTypes,
		TokenEndpointAuthMethod: string(v.TokenEndpointAuthMethod),
		ClientId:                v.ClientId,
		HomePageUrl:             v.HomePageUrl,
		AdditionalRefreshTokenReplayProtectionEnabled: v.AdditionalRefreshTokenReplayProtectionEnabled,
		AllowWildcardInRedirectUris:                   v.AllowWildcardInRedirectUris,
		CorsSettings:                                  corsFromSDK(v.CorsSettings),
		DeviceCustomVerificationUri:                   v.DeviceCustomVerificationUri,
		DevicePathId:                                  v.DevicePathId,
		DevicePollingInterval:                         v.DevicePollingInterval,
		DeviceTimeout:                                 v.DeviceTimeout,
		IdpSignoff:                                    v.IdpSignoff,
		IncludeX5t:                                    v.IncludeX5t,
		InitiateLoginUri:                              v.InitiateLoginUri,
		Jwks:                                          v.Jwks,
		JwksUrl:                                       v.JwksUrl,
		OpSessionCheckEnabled:                         v.OpSessionCheckEnabled,
		PostLogoutRedirectUris:                        v.PostLogoutRedirectUris,
		RedirectUris:                                  v.RedirectUris,
		RefreshTokenDuration:                          v.RefreshTokenDuration,
		RefreshTokenRollingDuration:                   v.RefreshTokenRollingDuration,
		RefreshTokenRollingGracePeriodDuration:        v.RefreshTokenRollingGracePeriodDuration,
		RequestScopesForMultipleResourcesEnabled:      v.RequestScopesForMultipleResourcesEnabled,
		RequireSignedRequestObject:                    v.RequireSignedRequestObject,
		SupportUnsignedRequestObject:                  v.SupportUnsignedRequestObject,
		TargetLinkUri:                                 v.TargetLinkUri,
		ParTimeout:                                    v.ParTimeout,
	}

	if v.ParRequirement != nil {
		s := string(*v.ParRequirement)
		oidc.ParRequirement = &s
	}
	if v.PkceEnforcement != nil {
		s := string(*v.PkceEnforcement)
		oidc.PkceEnforcement = &s
	}
	if v.RefreshTokenType != nil {
		s := string(*v.RefreshTokenType)
		oidc.RefreshTokenType = &s
	}

	if v.Mobile != nil {
		m := v.Mobile
		mobile := &mobileAppData{
			BundleId:          m.BundleId,
			PackageName:       m.PackageName,
			HuaweiAppId:       m.HuaweiAppId,
			HuaweiPackageName: m.HuaweiPackageName,
			UniversalAppLink:  m.UriPrefix,
		}
		if m.PasscodeGracePeriod != nil {
			mobile.PasscodeGracePeriod = m.PasscodeGracePeriod
		}
		if m.PasscodeRefreshDuration != nil {
			mobile.PasscodeRefreshSeconds = &m.PasscodeRefreshDuration.Duration
		}
		if id := m.IntegrityDetection; id != nil {
			detect := &integrityDetectionData{}
			if id.Mode != nil {
				enabled := *id.Mode == management.ENUMENABLEDSTATUS_ENABLED
				detect.Enabled = &enabled
			}
			for _, p := range id.ExcludedPlatforms {
				detect.ExcludedPlatforms = append(detect.ExcludedPlatforms, string(p))
			}
			if id.CacheDuration != nil {
				cd := &cacheDurationData{Amount: id.CacheDuration.Amount}
				if id.CacheDuration.Units != nil {
					u := string(*id.CacheDuration.Units)
					cd.Units = &u
				}
				detect.CacheDuration = cd
			}
			if id.GooglePlay != nil {
				gp := &googlePlayData{
					DecryptionKey:                 id.GooglePlay.DecryptionKey,
					VerificationKey:               id.GooglePlay.VerificationKey,
					ServiceAccountCredentialsJson: id.GooglePlay.ServiceAccountCredentials,
				}
				if id.GooglePlay.VerificationType != nil {
					vt := string(*id.GooglePlay.VerificationType)
					gp.VerificationType = &vt
				}
				detect.GooglePlay = gp
			}
			mobile.IntegrityDetection = detect
		}
		oidc.MobileApp = mobile
	}

	if v.Signing != nil {
		id := v.Signing.KeyRotationPolicy.Id
		oidc.Signing = &signingData{KeyRotationPolicyId: &id}
	}

	data.OIDC = oidc
	return &data
}

func fromSAML(v *management.ApplicationSAML) *applicationData {
	envId := ""
	if env, ok := v.GetEnvironmentOk(); ok && env != nil {
		envId = env.GetId()
	}
	data := commonFromBase(v.GetId(), envId, v.GetName(), v.Description, v.GetEnabled(), v.HiddenFromAppPortal, v.LoginPageUrl, nil, v.AccessControl, v.Icon)

	saml := &samlOptionsData{
		AcsUrls:                     v.AcsUrls,
		AssertionDuration:           v.AssertionDuration,
		AssertionSignedEnabled:      v.AssertionSigned,
		CorsSettings:                corsFromSDK(v.CorsSettings),
		DefaultTargetUrl:            v.DefaultTargetUrl,
		EnableRequestedAuthnContext: v.EnableRequestedAuthnContext,
		HomePageUrl:                 v.HomePageUrl,
		NameIdFormat:                v.NameIdFormat,
		ResponseIsSigned:            v.ResponseSigned,
		SessionNotOnOrAfterDuration: v.SessionNotOnOrAfterDuration,
		SloEndpoint:                 v.SloEndpoint,
		SloResponseEndpoint:         v.SloResponseEndpoint,
		SloWindow:                   v.SloWindow,
		SpEntityId:                  v.SpEntityId,
	}

	if v.SloBinding != nil {
		s := string(*v.SloBinding)
		saml.SloBinding = &s
	}

	typ := string(v.Type)
	saml.Type = &typ

	if v.IdpSigning != nil {
		key := &idpSigningKeyData{KeyId: v.IdpSigning.Key.Id}
		if v.IdpSigning.Algorithm != nil {
			a := string(*v.IdpSigning.Algorithm)
			key.Algorithm = &a
		}
		saml.IdpSigningKey = key
	}

	if v.SpEncryption != nil {
		saml.SpEncryption = &spEncryptionData{
			Algorithm:   string(v.SpEncryption.Algorithm),
			Certificate: certificateRefData{Id: v.SpEncryption.Certificate.Id},
		}
	}

	if v.SpVerification != nil {
		ids := make([]string, 0, len(v.SpVerification.Certificates))
		for _, c := range v.SpVerification.Certificates {
			ids = append(ids, c.Id)
		}
		saml.SpVerification = &spVerificationData{
			CertificateIds:     ids,
			AuthnRequestSigned: v.SpVerification.AuthnRequestSigned,
		}
	}

	if v.VirtualServerIdSettings != nil {
		vs := v.VirtualServerIdSettings
		entries := make([]virtualServerIdEntryData, 0, len(vs.VirtualServerIds))
		for _, e := range vs.VirtualServerIds {
			entries = append(entries, virtualServerIdEntryData{VsId: e.VsId, Default: e.Default})
		}
		saml.VirtualServerIdSettings = &virtualServerIdSettingsData{
			Enabled:          vs.Enabled,
			VirtualServerIds: entries,
		}
	}

	data.SAML = saml
	return &data
}

func fromExternalLink(v *management.ApplicationExternalLink) *applicationData {
	envId := ""
	if env, ok := v.GetEnvironmentOk(); ok && env != nil {
		envId = env.GetId()
	}
	data := commonFromBase(v.GetId(), envId, v.GetName(), v.Description, v.GetEnabled(), v.HiddenFromAppPortal, v.LoginPageUrl, nil, v.AccessControl, v.Icon)
	data.ExternalLink = &externalLinkOptionsData{HomePageUrl: v.HomePageUrl}
	return &data
}

func fromWSFED(v *management.ApplicationWSFED) *applicationData {
	envId := ""
	if env, ok := v.GetEnvironmentOk(); ok && env != nil {
		envId = env.GetId()
	}
	data := commonFromBase(v.GetId(), envId, v.GetName(), v.Description, v.GetEnabled(), v.HiddenFromAppPortal, v.LoginPageUrl, nil, v.AccessControl, v.Icon)

	wsfed := &wsfedOptionsData{
		Type:                string(v.Type),
		DomainName:          v.DomainName,
		ReplyUrl:            v.ReplyUrl,
		IdpSigningKey:       idpSigningKeyData{KeyId: v.IdpSigning.Key.Id, Algorithm: strPtr(string(v.IdpSigning.Algorithm))},
		AudienceRestriction: v.AudienceRestriction,
		CorsSettings:        corsFromSDK(v.CorsSettings),
		SloEndpoint:         v.SloEndpoint,
	}

	if v.SubjectNameIdentifierFormat != nil {
		s := string(*v.SubjectNameIdentifierFormat)
		wsfed.SubjectNameIdentifierFormat = &s
	}

	if v.Kerberos != nil {
		gateways := make([]wsfedKerberosGatewayData, 0, len(v.Kerberos.Gateways))
		for _, g := range v.Kerberos.Gateways {
			entry := wsfedKerberosGatewayData{Id: g.Id}
			typ := string(g.Type)
			entry.Type = &typ
			if id, ok := g.UserType.GetIdOk(); ok && id != nil {
				entry.UserType = userTypeRefData{Id: id}
			}
			gateways = append(gateways, entry)
		}
		wsfed.Kerberos = &wsfedKerberosData{Gateways: gateways}
	}

	data.WSFED = wsfed
	return &data
}

func strPtr(s string) *string { return &s }

// listSSOApplications lists all pingone_application resources in the target
// environment, skipping the three built-in PingOne system application types
// (Admin Console, Portal, Self Service) that have no Terraform schema
// representation — see the doc comment above toApplicationData.
func listSSOApplications(ctx context.Context, c *Client, _ string) ([]interface{}, error) {
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}

	var result []interface{}
	iterator := mgmt.ApplicationsApi.ReadAllApplications(ctx, c.environmentID.String()).Execute()
	for cursor, err := range iterator {
		if err != nil {
			return nil, fmt.Errorf("list applications: %w", err)
		}
		embedded, ok := cursor.EntityArray.GetEmbeddedOk()
		if !ok || embedded == nil {
			continue
		}
		apps, ok := embedded.GetApplicationsOk()
		if !ok {
			continue
		}
		for i := range apps {
			actual := apps[i].GetActualInstance()
			if actual == nil {
				continue
			}
			data, ok := toApplicationData(actual)
			if !ok {
				c.AddWarning(fmt.Sprintf("skipping application %s: PingOne system application types "+
					"(admin console, portal, self service) are not exportable via pingone_application "+
					"(no corresponding Terraform schema block)", describeSkippedApplication(actual)))
				continue
			}
			result = append(result, data)
		}
	}
	return result, nil
}

// getSSOApplication fetches a single pingone_application by ID.
func getSSOApplication(ctx context.Context, c *Client, _ string, resourceID string) (interface{}, error) {
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}
	resp, _, err := mgmt.ApplicationsApi.ReadOneApplication(ctx, c.environmentID.String(), resourceID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get application: %w", err)
	}
	actual := resp.GetActualInstance()
	if actual == nil {
		return nil, fmt.Errorf("get application %s: empty response", resourceID)
	}
	data, ok := toApplicationData(actual)
	if !ok {
		return nil, fmt.Errorf("get application %s: PingOne system application types are not exportable via pingone_application", resourceID)
	}
	return data, nil
}

// describeSkippedApplication returns a best-effort identifier for a skipped
// system application, for inclusion in the warning message.
func describeSkippedApplication(actual interface{}) string {
	switch v := actual.(type) {
	case *management.ApplicationPingOneAdminConsole:
		_ = v
		return "(PingOne Admin Console)"
	case *management.ApplicationPingOnePortal:
		return v.GetId()
	case *management.ApplicationPingOneSelfService:
		return v.GetId()
	default:
		return "(unknown)"
	}
}
