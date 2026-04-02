package pingone

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sync"

	pingoneGoClient "github.com/pingidentity/pingone-go-client/pingone"
	"golang.org/x/oauth2"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/core"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
)

// errAccessDenied is returned by fetchFlowVariableDeps when the API returns 403.
// Callers should log a warning and continue rather than failing the export.
var errAccessDenied = errors.New("access denied")

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
}

// listFlows implements list-then-get: lists all flows to collect IDs,
// then calls get for each to retrieve full details including graph data,
// settings, input schema, etc. (which the list endpoint may omit).
func listFlows(ctx context.Context, c *Client, _ string) ([]interface{}, error) {
	resp, _, err := c.apiClient.DaVinciFlowsApi.GetFlows(ctx, c.environmentID).Execute()
	if err != nil {
		return nil, fmt.Errorf("list flows: %w", err)
	}
	embedded := resp.GetEmbedded()
	flows := embedded.GetFlows()
	result := make([]interface{}, 0, len(flows))
	for _, flow := range flows {
		detail, _, err := c.apiClient.DaVinciFlowsApi.GetFlowById(ctx, c.environmentID, flow.GetId()).Execute()
		if err != nil {
			return nil, fmt.Errorf("get flow %s: %w", flow.GetId(), err)
		}
		// Fetch version details for variable dependencies.
		if err := fetchFlowVariableDeps(ctx, c, flow.GetId(), fmt.Sprintf("%g", flow.GetCurrentVersion())); err != nil {
			if errors.Is(err, errAccessDenied) {
				c.AddWarning(fmt.Sprintf("Unable to fetch flow variable dependencies for flow %s: %v. "+
					"The flow versions endpoint requires a role with higher privileges than Read Only. "+
					"Flow will be exported without depends_on references to DaVinci variables.", flow.GetId(), err))
			} else {
				return nil, fmt.Errorf("fetch flow variable deps for %s: %w", flow.GetId(), err)
			}
		}

		result = append(result, detail)
	}
	return result, nil
}

func getFlow(ctx context.Context, c *Client, _ string, resourceID string) (interface{}, error) {
	detail, _, err := c.apiClient.DaVinciFlowsApi.GetFlowById(ctx, c.environmentID, resourceID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get flow: %w", err)
	}

	// Fetch version details for variable dependencies.
	if cv, ok := detail.GetCurrentVersionOk(); ok && cv != nil {
		if err := fetchFlowVariableDeps(ctx, c, detail.GetId(), fmt.Sprintf("%g", *cv)); err != nil {
			if errors.Is(err, errAccessDenied) {
				c.AddWarning(fmt.Sprintf("Unable to fetch flow variable dependencies for flow %s: %v. "+
					"The flow versions endpoint requires a role with higher privileges than Read Only. "+
					"Flow will be exported without depends_on references to DaVinci variables.", detail.GetId(), err))
			} else {
				return nil, fmt.Errorf("fetch flow variable deps for %s: %w", detail.GetId(), err)
			}
		}
	}

	return detail, nil
}

// fetchFlowVariableDeps calls the flow versions endpoint via raw HTTP POST
// and caches variable dependency info for the given flow.
// Endpoint: POST /environments/{envID}/flows/{flowID}/versions/{versionID}
// Returns an error if the API call fails. If no variables are returned, processing continues quietly.
func fetchFlowVariableDeps(ctx context.Context, c *Client, flowID string, versionID string) error {
	cfg := c.apiClient.GetConfig()

	basePath, err := cfg.ServerURLWithContext(ctx, "DaVinciFlowVersionsApiService.GetVersionByIdUsingFlowId")
	if err != nil {
		return fmt.Errorf("resolve base URL: %w", err)
	}

	reqURL := fmt.Sprintf("%s/environments/%s/flows/%s/versions/%s",
		basePath, c.environmentID.String(), flowID, versionID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/vnd.pingidentity.flowversion.export+json")
	req.Header.Set("Accept", "application/json")

	// Extract OAuth2 token from context (same mechanism the SDK uses internally).
	if tok, ok := ctx.Value(pingoneGoClient.ContextOAuth2).(oauth2.TokenSource); ok {
		token, err := tok.Token()
		if err != nil {
			return fmt.Errorf("retrieve OAuth2 token: %w", err)
		}
		token.SetAuthHeader(req)
	} else if accessToken, ok := ctx.Value(pingoneGoClient.ContextAccessToken).(string); ok {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	resp, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%w: status %d: %s", errAccessDenied, resp.StatusCode, string(body))
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return fmt.Errorf("parse response JSON: %w", err)
	}

	deps := extractVariableDeps(parsed)
	if len(deps) > 0 {
		flowVariableDeps.Store(flowID, deps)
	}

	return nil
}

// extractVariableDeps parses the "variables" array from the flow version
// response's AdditionalProperties and returns RuntimeDependsOn entries for
// variables with context "company" or "flowInstance".
func extractVariableDeps(additionalProps map[string]interface{}) []core.RuntimeDependsOn {
	rawVars, ok := additionalProps["variables"]
	if !ok || rawVars == nil {
		return nil
	}

	varSlice, ok := rawVars.([]interface{})
	if !ok {
		return nil
	}

	var deps []core.RuntimeDependsOn
	seen := make(map[string]bool)

	for _, item := range varSlice {
		varMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

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
