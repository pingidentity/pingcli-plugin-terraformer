package pingone

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/patrickcping/pingone-go-sdk-v2/management"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplicationFlowPolicyAssignmentResourceRegistered(t *testing.T) {
	assert.True(t, isSupported("pingone_application_flow_policy_assignment"))
}

func TestApplicationFlowPolicyAssignmentResourceHandlerFunctions(t *testing.T) {
	h, ok := resourceHandlers["pingone_application_flow_policy_assignment"]
	assert.True(t, ok)
	assert.NotNil(t, h.list)
	assert.NotNil(t, h.get)
}

// newFlowPolicyAssignmentMux serves /applications (list) and
// /applications/{id}/flowPolicyAssignments[/{assignmentID}] from the same
// test server.
func newFlowPolicyAssignmentMux(applicationsBody map[string]any, assignmentsByApp map[string]any) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/environments/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if appID, ok := appIDFromFlowPolicyAssignmentsPath(r.URL.Path); ok {
			body, ok := assignmentsByApp[appID]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]any{"message": "not found"})
				return
			}
			_ = json.NewEncoder(w).Encode(body)
			return
		}
		_ = json.NewEncoder(w).Encode(applicationsBody)
	})
	return httptest.NewServer(mux)
}

// appIDFromFlowPolicyAssignmentsPath extracts the application ID from
// /environments/{envID}/applications/{applicationID}/flowPolicyAssignments[/{assignmentID}].
func appIDFromFlowPolicyAssignmentsPath(path string) (string, bool) {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	for i, seg := range segments {
		if seg == "applications" && i+2 < len(segments) && segments[i+2] == "flowPolicyAssignments" {
			return segments[i+1], true
		}
	}
	return "", false
}

func TestListApplicationFlowPolicyAssignments(t *testing.T) {
	applicationsBody := oidcApplicationsBody("app-1", "app-2")
	assignmentsByApp := map[string]any{
		"app-1": map[string]any{
			"_embedded": map[string]any{
				"flowPolicyAssignments": []any{
					map[string]any{"id": "assign-1", "priority": 1, "flowPolicy": map[string]any{"id": "policy-1"}},
				},
			},
		},
		"app-2": map[string]any{
			"_embedded": map[string]any{
				"flowPolicyAssignments": []any{
					map[string]any{"id": "assign-2", "priority": 2, "flowPolicy": map[string]any{"id": "policy-2"}},
				},
			},
		},
	}

	srv := newFlowPolicyAssignmentMux(applicationsBody, assignmentsByApp)
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := listApplicationFlowPolicyAssignments(testCtx(), c, "")
	require.NoError(t, err)

	var gotIDs []string
	for _, item := range result {
		assignment, ok := item.(*management.FlowPolicyAssignment)
		require.True(t, ok, "expected *management.FlowPolicyAssignment, got %T", item)
		gotIDs = append(gotIDs, assignment.GetId())
	}
	assert.ElementsMatch(t, []string{"assign-1", "assign-2"}, gotIDs)
}

func TestListApplicationFlowPolicyAssignments_NoAssignments(t *testing.T) {
	applicationsBody := oidcApplicationsBody("app-1")
	assignmentsByApp := map[string]any{
		"app-1": map[string]any{"count": 0},
	}

	srv := newFlowPolicyAssignmentMux(applicationsBody, assignmentsByApp)
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := listApplicationFlowPolicyAssignments(testCtx(), c, "")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestListApplicationFlowPolicyAssignments_ManagementClientUnavailable(t *testing.T) {
	c := &Client{}
	result, err := listApplicationFlowPolicyAssignments(testCtx(), c, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}

func TestGetApplicationFlowPolicyAssignment(t *testing.T) {
	srv := newFlowPolicyAssignmentMux(oidcApplicationsBody(), map[string]any{
		"app-1": map[string]any{"id": "assign-1", "priority": 1, "flowPolicy": map[string]any{"id": "policy-1"}},
	})
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := getApplicationFlowPolicyAssignment(testCtx(), c, "", "app-1/assign-1")
	require.NoError(t, err)

	assignment, ok := result.(*management.FlowPolicyAssignment)
	require.True(t, ok)
	assert.Equal(t, "assign-1", assignment.GetId())
	assert.Equal(t, int32(1), assignment.GetPriority())
}

func TestGetApplicationFlowPolicyAssignment_MalformedCompositeID(t *testing.T) {
	c := &Client{}
	result, err := getApplicationFlowPolicyAssignment(testCtx(), c, "", "no-slash")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resourceID must be applicationID/assignmentID")
	assert.Nil(t, result)
}

func TestGetApplicationFlowPolicyAssignment_ManagementClientUnavailable(t *testing.T) {
	c := &Client{}
	result, err := getApplicationFlowPolicyAssignment(testCtx(), c, "", "app-1/assign-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}
