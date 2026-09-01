package pingone

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupNestingResourceRegistered(t *testing.T) {
	assert.True(t, isSupported("pingone_group_nesting"))
}

func TestGroupNestingResourceHandlerFunctions(t *testing.T) {
	h, ok := resourceHandlers["pingone_group_nesting"]
	assert.True(t, ok)
	assert.NotNil(t, h.list)
	assert.NotNil(t, h.get)
}

// newGroupNestingMux serves /groups (list) and
// /groups/{groupID}/memberOfGroups[/{nestedGroupID}] from the same test
// server.
func newGroupNestingMux(groupsBody map[string]any, nestingsByGroup map[string]any) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/environments/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if groupID, nestedID, ok := parseGroupNestingPath(r.URL.Path); ok {
			body, exists := nestingsByGroup[groupID]
			if !exists {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]any{"message": "not found"})
				return
			}
			if nestedID != "" {
				// Single-get: unwrap embedded.groups[0] as the flat GroupNesting body.
				embedded, _ := body.(map[string]any)["_embedded"].(map[string]any)
				groups, _ := embedded["groups"].([]any)
				for _, g := range groups {
					gm := g.(map[string]any)
					if gm["id"] == nestedID {
						_ = json.NewEncoder(w).Encode(gm)
						return
					}
				}
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]any{"message": "not found"})
				return
			}
			_ = json.NewEncoder(w).Encode(body)
			return
		}
		_ = json.NewEncoder(w).Encode(groupsBody)
	})
	return httptest.NewServer(mux)
}

// parseGroupNestingPath extracts (groupID, nestedGroupID, ok) from
// /environments/{envID}/groups/{groupID}/memberOfGroups[/{nestedGroupID}].
// nestedGroupID is "" for the list path.
func parseGroupNestingPath(path string) (string, string, bool) {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	for i, seg := range segments {
		if seg == "groups" && i+2 < len(segments) && segments[i+2] == "memberOfGroups" {
			if i+3 < len(segments) {
				return segments[i+1], segments[i+3], true
			}
			return segments[i+1], "", true
		}
	}
	return "", "", false
}

func groupsListBody(ids ...string) map[string]any {
	groups := make([]any, 0, len(ids))
	for _, id := range ids {
		groups = append(groups, map[string]any{"id": id, "name": id})
	}
	return map[string]any{"_embedded": map[string]any{"groups": groups}}
}

func TestListGroupNestings(t *testing.T) {
	groupsBody := groupsListBody("group-1", "group-2")
	nestingsByGroup := map[string]any{
		"group-1": map[string]any{
			"_embedded": map[string]any{
				"groups": []any{
					map[string]any{"id": "nested-1", "type": "DIRECT"},
				},
			},
		},
		"group-2": map[string]any{
			"_embedded": map[string]any{"groups": []any{}},
		},
	}

	srv := newGroupNestingMux(groupsBody, nestingsByGroup)
	defer srv.Close()

	mgmt := newTestGroupManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := listGroupNestings(testCtx(), c, "")
	require.NoError(t, err)
	require.Len(t, result, 1)

	data, ok := result[0].(*groupNestingData)
	require.True(t, ok, "expected *groupNestingData, got %T", result[0])
	assert.Equal(t, "group-1", data.GroupID)
	assert.Equal(t, "nested-1", data.NestedGroupID)
	assert.Equal(t, "nested-1", data.ID)
}

func TestListGroupNestings_ManagementClientUnavailable(t *testing.T) {
	c := &Client{}
	result, err := listGroupNestings(testCtx(), c, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}

func TestGetGroupNesting(t *testing.T) {
	nestingsByGroup := map[string]any{
		"group-1": map[string]any{
			"_embedded": map[string]any{
				"groups": []any{
					map[string]any{"id": "nested-1", "type": "DIRECT"},
				},
			},
		},
	}
	srv := newGroupNestingMux(map[string]any{}, nestingsByGroup)
	defer srv.Close()

	mgmt := newTestGroupManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := getGroupNesting(testCtx(), c, "", "group-1/nested-1")
	require.NoError(t, err)

	data, ok := result.(*groupNestingData)
	require.True(t, ok)
	assert.Equal(t, "group-1", data.GroupID)
	assert.Equal(t, "nested-1", data.NestedGroupID)
}

func TestGetGroupNesting_MalformedCompositeID(t *testing.T) {
	c := &Client{}
	result, err := getGroupNesting(testCtx(), c, "", "no-slash")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resourceID must be groupID/nestedGroupID")
	assert.Nil(t, result)
}

func TestGetGroupNesting_ManagementClientUnavailable(t *testing.T) {
	c := &Client{}
	result, err := getGroupNesting(testCtx(), c, "", "group-1/nested-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}
