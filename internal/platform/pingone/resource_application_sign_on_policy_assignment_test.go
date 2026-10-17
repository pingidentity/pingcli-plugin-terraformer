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

func TestApplicationSignOnPolicyAssignmentResourceRegistered(t *testing.T) {
	assert.True(t, isSupported("pingone_application_sign_on_policy_assignment"))
}

func TestApplicationSignOnPolicyAssignmentResourceHandlerFunctions(t *testing.T) {
	h, ok := resourceHandlers["pingone_application_sign_on_policy_assignment"]
	assert.True(t, ok)
	assert.NotNil(t, h.list)
	assert.NotNil(t, h.get)
}

// newSignOnPolicyAssignmentMux serves /applications (list) and
// /applications/{id}/signOnPolicyAssignments[/{assignmentID}] from the same
// test server.
func newSignOnPolicyAssignmentMux(applicationsBody map[string]any, assignmentsByApp map[string]any) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/environments/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if appID, ok := appIDFromSignOnPolicyAssignmentsPath(r.URL.Path); ok {
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

// appIDFromSignOnPolicyAssignmentsPath extracts the application ID from
// /environments/{envID}/applications/{applicationID}/signOnPolicyAssignments[/{assignmentID}].
func appIDFromSignOnPolicyAssignmentsPath(path string) (string, bool) {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	for i, seg := range segments {
		if seg == "applications" && i+2 < len(segments) && segments[i+2] == "signOnPolicyAssignments" {
			return segments[i+1], true
		}
	}
	return "", false
}

func TestListApplicationSignOnPolicyAssignments(t *testing.T) {
	applicationsBody := oidcApplicationsBody("app-1", "app-2")
	assignmentsByApp := map[string]any{
		"app-1": map[string]any{
			"_embedded": map[string]any{
				"signOnPolicyAssignments": []any{
					map[string]any{"id": "assign-1", "priority": 1, "signOnPolicy": map[string]any{"id": "sop-1"}},
				},
			},
		},
		"app-2": map[string]any{
			"_embedded": map[string]any{
				"signOnPolicyAssignments": []any{
					map[string]any{"id": "assign-2", "priority": 2, "signOnPolicy": map[string]any{"id": "sop-2"}},
				},
			},
		},
	}

	srv := newSignOnPolicyAssignmentMux(applicationsBody, assignmentsByApp)
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := listApplicationSignOnPolicyAssignments(testCtx(), c, "")
	require.NoError(t, err)

	var gotIDs []string
	for _, item := range result {
		assignment, ok := item.(*management.SignOnPolicyAssignment)
		require.True(t, ok, "expected *management.SignOnPolicyAssignment, got %T", item)
		gotIDs = append(gotIDs, assignment.GetId())
	}
	assert.ElementsMatch(t, []string{"assign-1", "assign-2"}, gotIDs)
}

func TestListApplicationSignOnPolicyAssignments_NoAssignments(t *testing.T) {
	applicationsBody := oidcApplicationsBody("app-1")
	assignmentsByApp := map[string]any{
		"app-1": map[string]any{"count": 0},
	}

	srv := newSignOnPolicyAssignmentMux(applicationsBody, assignmentsByApp)
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := listApplicationSignOnPolicyAssignments(testCtx(), c, "")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestListApplicationSignOnPolicyAssignments_ManagementClientUnavailable(t *testing.T) {
	c := &Client{}
	result, err := listApplicationSignOnPolicyAssignments(testCtx(), c, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}

func TestGetApplicationSignOnPolicyAssignment(t *testing.T) {
	srv := newSignOnPolicyAssignmentMux(oidcApplicationsBody(), map[string]any{
		"app-1": map[string]any{"id": "assign-1", "priority": 1, "signOnPolicy": map[string]any{"id": "sop-1"}},
	})
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := getApplicationSignOnPolicyAssignment(testCtx(), c, "", "app-1/assign-1")
	require.NoError(t, err)

	assignment, ok := result.(*management.SignOnPolicyAssignment)
	require.True(t, ok)
	assert.Equal(t, "assign-1", assignment.GetId())
	assert.Equal(t, int32(1), assignment.GetPriority())
}

func TestGetApplicationSignOnPolicyAssignment_MalformedCompositeID(t *testing.T) {
	c := &Client{}
	result, err := getApplicationSignOnPolicyAssignment(testCtx(), c, "", "no-slash")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resourceID must be applicationID/assignmentID")
	assert.Nil(t, result)
}

func TestGetApplicationSignOnPolicyAssignment_ManagementClientUnavailable(t *testing.T) {
	c := &Client{}
	result, err := getApplicationSignOnPolicyAssignment(testCtx(), c, "", "app-1/assign-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}
