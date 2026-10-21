package pingone

import (
	"context"
	"fmt"
	"strings"

	"github.com/patrickcping/pingone-go-sdk-v2/management"
)

// pingone_resource_attribute is a child of pingone_resource. Attributes can
// only be created for CUSTOM and OPENID_CONNECT resources (per provider
// docs), so listing enumerates CUSTOM resources (shared listCustomResourceIDs
// helper) plus the single built-in OPENID_CONNECT resource (shared
// findBuiltInResourceID helper), then lists attributes per resource.
//
// The SDK's ResourceAttribute.Resource only carries {Id} — no resource_type
// discriminator — so each resource requires a supplemental
// ResourcesApi.ReadOneResource lookup to resolve its Type, reusing
// resolveResourceType from resource_application_resource_grant.go.
//
// Every resource (CUSTOM or OPENID_CONNECT) automatically carries CORE and
// PREDEFINED attributes (e.g. "sub", "given_name") that exist without ever
// being explicitly created — the provider computes their value client-side
// from a hardcoded default table rather than reading it from the API, so a
// plain GET never returns a real value for them (confirmed against
// terraform-provider-pingone's resource_resource_attribute.go Read/Update
// handlers). Only ResourceAttribute.Type == CUSTOM reflects a genuinely
// user-authored attribute with a real, exportable value — listing filters
// to that type only, regardless of which resource type it lives on.
func init() {
	registerResource("pingone_resource_attribute", resourceHandler{
		list: listResourceAttributes,
		get:  getResourceAttribute,
	})
}

type resourceAttributeData struct {
	ID               string
	EnvironmentID    string
	ResourceType     string
	CustomResourceID string
	ResourceID       string
	Name             string
	Value            string
	Type             string
	IdToken          *bool
	UserInfo         *bool
}

func toResourceAttributeData(environmentID string, v *management.ResourceAttribute, resourceType management.EnumResourceType) *resourceAttributeData {
	resourceID := ""
	if res, ok := v.GetResourceOk(); ok && res != nil {
		resourceID = res.GetId()
	}
	data := &resourceAttributeData{
		ID:            v.GetId(),
		EnvironmentID: environmentID,
		ResourceType:  string(resourceType),
		ResourceID:    resourceID,
		Name:          v.GetName(),
		Value:         v.GetValue(),
	}
	if t, ok := v.GetTypeOk(); ok && t != nil {
		data.Type = string(*t)
	}
	if idToken, ok := v.GetIdTokenOk(); ok {
		data.IdToken = idToken
	}
	if userInfo, ok := v.GetUserInfoOk(); ok {
		data.UserInfo = userInfo
	}
	if resourceType == management.ENUMRESOURCETYPE_CUSTOM {
		data.CustomResourceID = resourceID
	}
	return data
}

// listResourceAttributes implements list-then-scan: lists all CUSTOM
// resources plus the built-in OPENID_CONNECT resource, then lists attributes
// for each, keeping only genuinely user-authored (Type == CUSTOM) attributes.
func listResourceAttributes(ctx context.Context, c *Client, _ string) ([]interface{}, error) {
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}

	resourceIDs, resourceTypes, err := listAttributeEligibleResources(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("list resource attributes: %w", err)
	}

	var result []interface{}
	for _, resourceID := range resourceIDs {
		resourceType := resourceTypes[resourceID]
		iterator := mgmt.ResourceAttributesApi.ReadAllResourceAttributes(ctx, c.environmentID.String(), resourceID).Execute()
		for cursor, err := range iterator {
			if err != nil {
				return nil, fmt.Errorf("list resource attributes for resource %s: %w", resourceID, err)
			}
			embedded, ok := cursor.EntityArray.GetEmbeddedOk()
			if !ok || embedded == nil {
				continue
			}
			attrs, ok := embedded.GetAttributesOk()
			if !ok {
				continue
			}
			for i := range attrs {
				attr, ok := attrs[i].GetActualInstance().(*management.ResourceAttribute)
				if !ok || attr == nil {
					continue
				}
				if attrType, ok := attr.GetTypeOk(); !ok || attrType == nil || *attrType != management.ENUMRESOURCEATTRIBUTETYPE_CUSTOM {
					continue
				}
				result = append(result, toResourceAttributeData(c.environmentID.String(), attr, resourceType))
			}
		}
	}
	return result, nil
}

// getResourceAttribute retrieves a single resource attribute. resourceID is
// a composite "resourceID/attributeID" string.
func getResourceAttribute(ctx context.Context, c *Client, _ string, resourceID string) (interface{}, error) {
	parts := strings.SplitN(resourceID, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("get resource attribute: resourceID must be resourceID/attributeID, got: %s", resourceID)
	}
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}
	attr, _, err := mgmt.ResourceAttributesApi.ReadOneResourceAttribute(ctx, c.environmentID.String(), parts[0], parts[1]).Execute()
	if err != nil {
		return nil, fmt.Errorf("get resource attribute: %w", err)
	}
	resourceType, err := resolveResourceType(ctx, c, mgmt, make(map[string]management.EnumResourceType), parts[0])
	if err != nil {
		return nil, fmt.Errorf("get resource attribute: %w", err)
	}
	return toResourceAttributeData(c.environmentID.String(), attr, resourceType), nil
}

// listAttributeEligibleResources returns the IDs of every resource that can
// have attributes — all CUSTOM resources plus the single built-in
// OPENID_CONNECT resource — along with a map of resource ID to resolved type.
func listAttributeEligibleResources(ctx context.Context, c *Client) ([]string, map[string]management.EnumResourceType, error) {
	customIDs, err := listCustomResourceIDs(ctx, c)
	if err != nil {
		return nil, nil, err
	}

	types := make(map[string]management.EnumResourceType, len(customIDs)+1)
	for _, id := range customIDs {
		types[id] = management.ENUMRESOURCETYPE_CUSTOM
	}

	openIDConnectID, err := findBuiltInResourceID(ctx, c, management.ENUMRESOURCETYPE_OPENID_CONNECT)
	if err != nil {
		return nil, nil, err
	}
	types[openIDConnectID] = management.ENUMRESOURCETYPE_OPENID_CONNECT

	ids := append(customIDs, openIDConnectID)
	return ids, types, nil
}
