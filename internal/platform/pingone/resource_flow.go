package pingone

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"

	pingone "github.com/pingidentity/pingone-go-client/pingone"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/core"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
)

// errAccessDenied is returned by fetchFlowVariableDeps when the API returns 403.
var errAccessDenied = errors.New("access denied")

// jsLinksLegacyHint is appended to errors caused by the legacy jsLinks string format.
const jsLinksLegacyHint = "one or more DaVinci flows contain a legacy jsLinks format (string[] instead of object[]) " +
	"that the PingOne SDK cannot parse. To resolve this, update the affected flow's jsLinks via the DaVinci UI or API. " +
	"See https://github.com/pingidentity/pingcli-plugin-terraformer/issues/10"

// flowVariableDeps caches variable dependency info per flow ID.
// Populated by listFlows/getFlow, consumed by the custom handler.
var flowVariableDeps sync.Map // flowID (string) -> []core.RuntimeDependsOn

func init() {
	// API client dispatch.
	registerResource("pingone_davinci_flow", resourceHandler{
		list: listFlows,
		get:  getFlow,
	})

	// Custom handler: extract variable dependencies from the flow version response.
	registerHandler("handleFlowVariableDependencies", handleFlowVariableDependencies)

	// Embedded reference: subFlowId inside node properties references another flow.
	registerEmbeddedReferenceRule(core.EmbeddedReferenceRule{
		ResourceType:       "pingone_davinci_flow",
		AttributePath:      "graph_data.elements.nodes.*.data.properties",
		TargetResourceType: "pingone_davinci_flow",
		JSONKeyPath:        "subFlowId.value.value",
		ReferenceField:     "id",
	})

	// Embedded reference with fallback: form.value inside node properties
	// references a DaVinci form. The form resource is not yet exported, so
	// the UUID is emitted as a Terraform variable. When pingone_davinci_form
	// is added, the graph lookup will succeed and the variable will be
	// automatically promoted to a resource reference.
	registerEmbeddedReferenceRule(core.EmbeddedReferenceRule{
		ResourceType:       "pingone_davinci_flow",
		AttributePath:      "graph_data.elements.nodes.*.data.properties",
		TargetResourceType: "pingone_davinci_form",
		JSONKeyPath:        "form.value",
		ReferenceField:     "id",
		Strategy:           "reference_with_fallback",
		VariablePrefix:     "davinci_form",
		VariableNamingPath: "nodeTitle.value",
	})

	// Embedded reference with fallback: theme.value inside node properties
	// (showForm capability) references a DaVinci UI template. Mirrors the
	// form.value rule above. theme.value may also hold a non-UUID mode-flag
	// sentinel (e.g. "useThemeId", or the unconfirmed "activeTheme") instead
	// of a direct UUID — those are left untouched by the unconditional
	// UUID-format guard in core.processRawHCLValue, not by an enumerated
	// skip list here.
	registerEmbeddedReferenceRule(core.EmbeddedReferenceRule{
		ResourceType:       "pingone_davinci_flow",
		AttributePath:      "graph_data.elements.nodes.*.data.properties",
		TargetResourceType: "pingone_branding_theme",
		JSONKeyPath:        "theme.value",
		ReferenceField:     "id",
		Strategy:           "reference_with_fallback",
		VariablePrefix:     "davinci_theme",
		VariableNamingPath: "nodeTitle.value",
	})

	// Embedded reference with fallback: themeId.value inside node properties
	// holds the theme UUID rich-text/Slate-wrapped
	// (`[{"children":[{"text":"<uuid>"}]}]`) when theme.value is the mode
	// flag "useThemeId". Gated on the sibling theme.value precondition so it
	// only fires in that indirect mode; theme.value itself is left
	// unmodified by this rule.
	registerEmbeddedReferenceRule(core.EmbeddedReferenceRule{
		ResourceType:        "pingone_davinci_flow",
		AttributePath:       "graph_data.elements.nodes.*.data.properties",
		TargetResourceType:  "pingone_branding_theme",
		JSONKeyPath:         "themeId.value",
		ReferenceField:      "id",
		Strategy:            "reference_with_fallback",
		VariablePrefix:      "davinci_theme",
		VariableNamingPath:  "nodeTitle.value",
		PreconditionKeyPath: "theme.value",
		PreconditionValue:   "useThemeId",
		UnwrapMode:          "rich_text",
	})
}

// listFlowIDs fetches only flow IDs and names from the list endpoint via raw HTTP
// with ?attributes=id,name.
//
// WORKAROUND: The SDK's GetFlows() unmarshals into DaVinciFlowResponse /
// DaVinciFlowResponseLinks, both of which enforce required-field validation in
// UnmarshalJSON. The list endpoint omits several fields that the SDK requires
// (e.g. the "version" link in _links) regardless of what ?attributes= is passed,
// causing an unmarshal error. Additionally, without ?attributes= the list response
// includes full flow graph data, which can exceed the 10 MB AWS API Gateway limit
// and return HTTP 500 for environments with many or large flows.
//
// Raw HTTP with ?attributes=id,name returns ~120 KB for 100 flows and avoids both
// the size limit and the unmarshal validation.
//
// TODO: Revert to SDK's GetFlows() once it either omits required-field validation
// for sparse responses or the list endpoint returns all SDK-required fields.
type flowID struct{ ID, Name string }

func listFlowIDs(ctx context.Context, c *Client) ([]flowID, error) {
	cfg := c.apiClient.GetConfig()
	scheme := cfg.Scheme
	if scheme == "" {
		scheme = "https"
	}
	return fetchFlowIDs(ctx, scheme, cfg.Host, c.environmentID.String(), cfg.HTTPClient)
}

// fetchFlowIDs is the testable core of listFlowIDs: it takes explicit connection
// parameters so tests can point it at an httptest.Server without needing a real
// SDK client.
func fetchFlowIDs(ctx context.Context, scheme, host, environmentID string, httpClient *http.Client) ([]flowID, error) {
	reqURL := fmt.Sprintf("%s://%s/v1/environments/%s/flows?attributes=id,name",
		scheme, host, environmentID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create list request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute list request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read list response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse list response: %w", err)
	}

	var out []flowID
	if embedded, ok := raw["_embedded"].(map[string]interface{}); ok {
		if items, ok := embedded["flows"].([]interface{}); ok {
			for _, item := range items {
				if m, ok := item.(map[string]interface{}); ok {
					id, _ := m["id"].(string)
					name, _ := m["name"].(string)
					if id != "" {
						out = append(out, flowID{ID: id, Name: name})
					}
				}
			}
		}
	}
	return out, nil
}

// clearInputSchemaIfAuthentication zeroes the InputSchema field on detail if
// the flow's trigger type is AUTHENTICATION. The Terraform provider rejects
// input_schema blocks on AUTHENTICATION flows, so this suppresses the attribute
// before the struct enters the processing pipeline.
//
// The function is a no-op when detail is nil, when no Trigger is set, or when
// the trigger type is anything other than AUTHENTICATION.
func clearInputSchemaIfAuthentication(detail *pingone.DaVinciFlowResponse) {
	if detail == nil {
		return
	}
	trigger, ok := detail.GetTriggerOk()
	if !ok || trigger == nil {
		return
	}
	if trigger.GetType() == pingone.DAVINCIFLOWTRIGGERRESPONSETYPE_AUTHENTICATION {
		detail.InputSchema = nil
	}
}

// listFlows implements list-then-get: fetches flow IDs via a minimal raw HTTP
// request, then calls GetFlowById for each to retrieve full details.
func listFlows(ctx context.Context, c *Client, _ string) ([]interface{}, error) {
	stubs, err := listFlowIDs(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("list flows: %w", err)
	}
	result := make([]interface{}, 0, len(stubs))
	for _, stub := range stubs {
		detail, _, err := c.apiClient.DaVinciFlowsApi.GetFlowById(ctx, c.environmentID, stub.ID).Execute()
		if err != nil {
			return nil, fmt.Errorf("get flow id=%s name=%q: %w", stub.ID, stub.Name, err)
		}
		clearInputSchemaIfAuthentication(detail)
		// Fetch version details for variable dependencies.
		if err := fetchFlowVariableDeps(ctx, c, stub.ID, fmt.Sprintf("%g", detail.GetCurrentVersion())); err != nil {
			// if errors.Is(err, errAccessDenied) {
			// 	c.AddWarning(fmt.Sprintf("Unable to fetch flow variable dependencies for flow %s: %v. "+
			// 		"The flow versions endpoint requires a role with higher privileges than Read Only. "+
			// 		"Flow will be exported without depends_on references to DaVinci variables.", stub.ID, err))
			// } else {
			c.AddWarning(fmt.Sprintf("Unable to fetch flow variable dependencies for flow %s: %v. "+
				"Flow will be exported without depends_on references to DaVinci variables.", stub.ID, err))
			// }
		}

		result = append(result, detail)
	}
	return result, nil
}

func getFlow(ctx context.Context, c *Client, _ string, resourceID string) (interface{}, error) {
	detail, _, err := c.apiClient.DaVinciFlowsApi.GetFlowById(ctx, c.environmentID, resourceID).Execute()
	if err != nil {
		if strings.Contains(err.Error(), "jsLinks") {
			return nil, fmt.Errorf("get flow: %s", jsLinksLegacyHint)
		}
		return nil, fmt.Errorf("get flow: %w", err)
	}
	clearInputSchemaIfAuthentication(detail)

	// Fetch version details for variable dependencies.
	if cv, ok := detail.GetCurrentVersionOk(); ok && cv != nil {
		if err := fetchFlowVariableDeps(ctx, c, detail.GetId(), fmt.Sprintf("%g", *cv)); err != nil {
			// if errors.Is(err, errAccessDenied) {
			// 	c.AddWarning(fmt.Sprintf("Unable to fetch flow variable dependencies for flow %s: %v. "+
			// 		"The flow versions endpoint requires a role with higher privileges than Read Only. "+
			// 		"Flow will be exported without depends_on references to DaVinci variables.", detail.GetId(), err))
			// } else {
			c.AddWarning(fmt.Sprintf("Unable to fetch flow variable dependencies for flow %s: %v. "+
				"Flow will be exported without depends_on references to DaVinci variables.", detail.GetId(), err))
			// }
		}
	}

	return detail, nil
}

// fetchFlowVariableDeps calls the flow version export endpoint and caches
// variable dependency info for the given flow.
//
// The export endpoint (POST with Content-Type application/vnd.pingidentity.flowversion.export+json)
// returns the "variables" array that the standard GET endpoint omits.
// Endpoint: POST /environments/{envID}/flows/{flowID}/versions/{versionID}
//
// The SDK's DaVinciFlowVersionsApiService does not expose this endpoint as a
// method (the export POST uses a vendor content-type that is outside the SDK's
// generated paths), so we use the SDK's configured HTTP client directly to
// ensure the correct regional host and OAuth2 transport are used. The response
// is decoded into the SDK's DaVinciExportFlowVersionResponse type.
func fetchFlowVariableDeps(ctx context.Context, c *Client, flowID string, versionID string) error {
	cfg := c.apiClient.GetConfig()

	// Build the request URL using the SDK's configured host and scheme,
	// which reflect the correct regional API domain (e.g. api.pingone.eu).
	scheme := cfg.Scheme
	if scheme == "" {
		scheme = "https"
	}

	variables, err := exportFlowVersion(ctx, scheme, cfg.Host, c.environmentID.String(), flowID, versionID, cfg.HTTPClient)
	if err != nil {
		return err
	}

	deps := extractVariableDeps(variables)
	if len(deps) > 0 {
		flowVariableDeps.Store(flowID, deps)
	}

	return nil
}

// exportFlowVersion is the testable core of fetchFlowVariableDeps. It POSTs to
// the flow version export endpoint and returns the variables array from the
// SDK's DaVinciExportFlowVersionResponse.
func exportFlowVersion(ctx context.Context, scheme, host, environmentID, flowID, versionID string, httpClient *http.Client) ([]map[string]interface{}, error) {
	reqURL := fmt.Sprintf("%s://%s/v1/environments/%s/flows/%s/versions/%s",
		scheme, host, environmentID, flowID, versionID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/vnd.pingidentity.flowversion.export+json")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("%w: status %d: %s", errAccessDenied, resp.StatusCode, string(body))
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var exportResp pingone.DaVinciExportFlowVersionResponse
	if err := json.Unmarshal(body, &exportResp); err != nil {
		return nil, fmt.Errorf("parse response JSON: %w", err)
	}

	return exportResp.GetVariables(), nil
}

// extractVariableDeps returns RuntimeDependsOn entries for the variables in the
// export response that have context "company" or "flowInstance".
func extractVariableDeps(variables []map[string]interface{}) []core.RuntimeDependsOn {
	var deps []core.RuntimeDependsOn
	seen := make(map[string]bool)

	for _, varMap := range variables {
		ctx, _ := varMap["context"].(string)
		if ctx != "company" && ctx != "flowInstance" {
			continue
		}

		id, _ := varMap["id"].(string)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true

		deps = append(deps, core.RuntimeDependsOn{
			ResourceType: "pingone_davinci_variable",
			ResourceID:   id,
		})
	}

	return deps
}

// handleFlowVariableDependencies is the custom handler that retrieves cached
// variable dependencies for a flow and returns them via the __depends_on sentinel.
func handleFlowVariableDependencies(data interface{}, _ *schema.ResourceDefinition) (map[string]interface{}, error) {
	// Extract the flow ID from the API data.
	flowID := extractFlowID(data)
	if flowID == "" {
		return nil, nil
	}

	val, ok := flowVariableDeps.Load(flowID)
	if !ok {
		return nil, nil
	}

	deps, ok := val.([]core.RuntimeDependsOn)
	if !ok || len(deps) == 0 {
		return nil, nil
	}

	return map[string]interface{}{
		"__depends_on": deps,
	}, nil
}

// extractFlowID uses reflection to get the Id field from the API response struct.
func extractFlowID(data interface{}) string {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return ""
	}
	f := v.FieldByName("Id")
	if !f.IsValid() {
		return ""
	}
	if f.Kind() == reflect.String {
		return f.String()
	}
	return ""
}
