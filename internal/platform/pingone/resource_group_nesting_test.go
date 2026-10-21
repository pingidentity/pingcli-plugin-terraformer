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

// membership is one entry of _embedded.groupMemberships, the real shape of
// GET /environments/{envID}/groups/{groupID}/memberOfGroups (verified live:
// the embedded key is groupMemberships, not groups).
type membership struct {
	id   string
	typ  string
	name string
}

// newGroupNestingMux serves /groups (list) and
// /groups/{groupID}/memberOfGroups[/{nestedGroupID}] from the same test
// server, mirroring the live API response shapes: the list returns
// _embedded.groupMemberships ([]GroupMembership), the single get returns
// the flat membership body.
func newGroupNestingMux(groupsBody map[string]any, membershipsByGroup map[string][]membership) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/environments/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if groupID, nestedID, ok := parseGroupNestingPath(r.URL.Path); ok {
			members, exists := membershipsByGroup[groupID]
			if !exists {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]any{"message": "not found"})
				return
			}
			if nestedID != "" {
				// Single-get: flat membership body (decoded into GroupNesting).
				for _, m := range members {
					if m.id == nestedID {
						_ = json.NewEncoder(w).Encode(map[string]any{"id": m.id, "type": m.typ})
						return
					}
				}
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]any{"message": "not found"})
				return
			}
			// List: _embedded.groupMemberships.
			items := make([]any, 0, len(members))
			for _, m := range members {
				items = append(items, map[string]any{"id": m.id, "type": m.typ, "name": m.name})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"_embedded": map[string]any{"groupMemberships": items},
			})
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
	membershipsByGroup := map[string][]membership{
		"group-1": {{id: "nested-1", typ: "DIRECT", name: "child group"}},
		"group-2": {},
	}

	srv := newGroupNestingMux(groupsBody, membershipsByGroup)
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
	membershipsByGroup := map[string][]membership{
		"group-1": {{id: "nested-1", typ: "DIRECT", name: "child group"}},
	}
	srv := newGroupNestingMux(map[string]any{}, membershipsByGroup)
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
