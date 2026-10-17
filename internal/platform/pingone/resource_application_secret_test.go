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

func TestApplicationSecretResourceRegistered(t *testing.T) {
	assert.True(t, isSupported("pingone_application_secret"))
}

func TestApplicationSecretResourceHandlerFunctions(t *testing.T) {
	h, ok := resourceHandlers["pingone_application_secret"]
	assert.True(t, ok)
	assert.NotNil(t, h.list)
	assert.NotNil(t, h.get)
}

// newApplicationSecretMux serves /applications (list, via ReadAllApplications)
// and /applications/{id}/secret (via ReadApplicationSecret) from the same
// test server.
func newApplicationSecretMux(applicationsBody map[string]any, secretsByApp map[string]any) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/environments/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if appID, ok := appIDFromSecretPath(r.URL.Path); ok {
			body, ok := secretsByApp[appID]
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

// appIDFromSecretPath extracts the application ID from
// /environments/{envID}/applications/{applicationID}/secret.
func appIDFromSecretPath(path string) (string, bool) {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	for i, seg := range segments {
		if seg == "applications" && i+2 < len(segments) && segments[i+2] == "secret" {
			return segments[i+1], true
		}
	}
	return "", false
}

func oidcApplicationsBody(ids ...string) map[string]any {
	var apps []any
	for _, id := range ids {
		apps = append(apps, map[string]any{
			"id":                      id,
			"name":                    "App " + id,
			"enabled":                 true,
			"protocol":                "OPENID_CONNECT",
			"type":                    "WEB_APP",
			"grantTypes":              []string{"AUTHORIZATION_CODE"},
			"responseTypes":           []string{"CODE"},
			"tokenEndpointAuthMethod": "CLIENT_SECRET_BASIC",
		})
	}
	return map[string]any{
		"_embedded": map[string]any{
			"applications": apps,
		},
	}
}

func TestListApplicationSecrets(t *testing.T) {
	applicationsBody := oidcApplicationsBody("app-1", "app-2")
	secretsByApp := map[string]any{
		"app-1": map[string]any{"secret": "s3cr3t-1"},
		"app-2": map[string]any{"secret": "s3cr3t-2"},
	}

	srv := newApplicationSecretMux(applicationsBody, secretsByApp)
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := listApplicationSecrets(testCtx(), c, "")
	require.NoError(t, err)
	require.Len(t, result, 2)

	var gotIDs []string
	for _, item := range result {
		data, ok := item.(*applicationSecretData)
		require.True(t, ok, "expected *applicationSecretData, got %T", item)
		gotIDs = append(gotIDs, data.ID)
	}
	assert.ElementsMatch(t, []string{"app-1", "app-2"}, gotIDs)
}

func TestListApplicationSecrets_SkipsApplicationsWithoutSecret(t *testing.T) {
	applicationsBody := oidcApplicationsBody("app-1", "app-2")
	secretsByApp := map[string]any{
		"app-1": map[string]any{"secret": "s3cr3t-1"},
		// app-2 intentionally omitted -> 404 -> should be skipped with a warning.
	}

	srv := newApplicationSecretMux(applicationsBody, secretsByApp)
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := listApplicationSecrets(testCtx(), c, "")
	require.NoError(t, err)
	require.Len(t, result, 1)

	data, ok := result[0].(*applicationSecretData)
	require.True(t, ok)
	assert.Equal(t, "app-1", data.ID)
	assert.NotEmpty(t, c.Warnings())
}

func TestListApplicationSecrets_ManagementClientUnavailable(t *testing.T) {
	c := &Client{}
	result, err := listApplicationSecrets(testCtx(), c, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}

func TestGetApplicationSecret(t *testing.T) {
	srv := newApplicationSecretMux(oidcApplicationsBody(), map[string]any{
		"app-1": map[string]any{
			"secret": "s3cr3t-1",
			"previous": map[string]any{
				"secret":    "old-secret",
				"expiresAt": "2026-01-01T00:00:00Z",
				"lastUsed":  "2025-12-01T00:00:00Z",
			},
		},
	})
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := getApplicationSecret(testCtx(), c, "", "app-1")
	require.NoError(t, err)

	data, ok := result.(*applicationSecretData)
	require.True(t, ok)
	assert.Equal(t, "app-1", data.ID)
	assert.Equal(t, "app-1", data.ApplicationID)
	require.NotNil(t, data.Secret)
	assert.Equal(t, "s3cr3t-1", *data.Secret)
	require.NotNil(t, data.Previous)
	require.NotNil(t, data.Previous.Secret)
	assert.Equal(t, "old-secret", *data.Previous.Secret)
	assert.NotEmpty(t, data.Previous.ExpiresAt)
	require.NotNil(t, data.Previous.LastUsed)
}

func TestGetApplicationSecret_NotFound(t *testing.T) {
	srv := newApplicationSecretMux(oidcApplicationsBody(), map[string]any{})
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := getApplicationSecret(testCtx(), c, "", "missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get application secret")
	assert.Nil(t, result)
}

func TestGetApplicationSecret_ManagementClientUnavailable(t *testing.T) {
	c := &Client{}
	result, err := getApplicationSecret(testCtx(), c, "", "app-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}
