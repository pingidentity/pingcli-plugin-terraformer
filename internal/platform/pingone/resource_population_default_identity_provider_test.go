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

func TestPopulationDefaultIdentityProviderResourceRegistered(t *testing.T) {
	assert.True(t, isSupported("pingone_population_default_identity_provider"))
}

func TestPopulationDefaultIdentityProviderResourceHandlerFunctions(t *testing.T) {
	h, ok := resourceHandlers["pingone_population_default_identity_provider"]
	assert.True(t, ok)
	assert.NotNil(t, h.list)
	assert.NotNil(t, h.get)
}

// newPopulationDefaultIdpMux serves /populations (list) and
// /populations/{populationID}/defaultIdentityProvider from the same test
// server.
func newPopulationDefaultIdpMux(populationsBody map[string]any, idpByPopulation map[string]any) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/environments/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if populationID, ok := populationIDFromDefaultIdpPath(r.URL.Path); ok {
			body, ok := idpByPopulation[populationID]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]any{"message": "not found"})
				return
			}
			_ = json.NewEncoder(w).Encode(body)
			return
		}
		_ = json.NewEncoder(w).Encode(populationsBody)
	})
	return httptest.NewServer(mux)
}

// populationIDFromDefaultIdpPath extracts the population ID from
// /environments/{envID}/populations/{populationID}/defaultIdentityProvider.
func populationIDFromDefaultIdpPath(path string) (string, bool) {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	for i, seg := range segments {
		if seg == "populations" && i+2 < len(segments) && segments[i+2] == "defaultIdentityProvider" {
			return segments[i+1], true
		}
	}
	return "", false
}

func populationsListBody(ids ...string) map[string]any {
	pops := make([]any, 0, len(ids))
	for _, id := range ids {
		pops = append(pops, map[string]any{"id": id, "name": id})
	}
	return map[string]any{"_embedded": map[string]any{"populations": pops}}
}

func TestListPopulationDefaultIdentityProviders(t *testing.T) {
	populationsBody := populationsListBody("pop-1", "pop-2")
	idpByPopulation := map[string]any{
		"pop-1": map[string]any{"id": "idp-1", "type": "GOOGLE"},
		"pop-2": map[string]any{"type": "PING_ONE"},
	}

	srv := newPopulationDefaultIdpMux(populationsBody, idpByPopulation)
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := listPopulationDefaultIdentityProviders(testCtx(), c, "")
	require.NoError(t, err)
	require.Len(t, result, 2)

	byPopulation := map[string]*populationDefaultIdentityProviderData{}
	for _, item := range result {
		data, ok := item.(*populationDefaultIdentityProviderData)
		require.True(t, ok, "expected *populationDefaultIdentityProviderData, got %T", item)
		byPopulation[data.PopulationID] = data
	}

	pop1 := byPopulation["pop-1"]
	require.NotNil(t, pop1)
	assert.Equal(t, "idp-1", pop1.IdentityProviderID)
	assert.Equal(t, "GOOGLE", pop1.Type)

	pop2 := byPopulation["pop-2"]
	require.NotNil(t, pop2)
	assert.Empty(t, pop2.IdentityProviderID)
	assert.Equal(t, "PING_ONE", pop2.Type)
}

func TestListPopulationDefaultIdentityProviders_SkipsOnError(t *testing.T) {
	populationsBody := populationsListBody("pop-1")
	// pop-1 intentionally omitted from idpByPopulation -> 404 -> should be
	// skipped with a warning rather than failing the whole export.
	srv := newPopulationDefaultIdpMux(populationsBody, map[string]any{})
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := listPopulationDefaultIdentityProviders(testCtx(), c, "")
	require.NoError(t, err)
	assert.Empty(t, result)
	assert.NotEmpty(t, c.Warnings())
}

func TestListPopulationDefaultIdentityProviders_ManagementClientUnavailable(t *testing.T) {
	c := &Client{}
	result, err := listPopulationDefaultIdentityProviders(testCtx(), c, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}

func TestGetPopulationDefaultIdentityProvider(t *testing.T) {
	idpByPopulation := map[string]any{
		"pop-1": map[string]any{"id": "idp-1", "type": "GOOGLE"},
	}
	srv := newPopulationDefaultIdpMux(map[string]any{}, idpByPopulation)
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := getPopulationDefaultIdentityProvider(testCtx(), c, "", "pop-1")
	require.NoError(t, err)

	data, ok := result.(*populationDefaultIdentityProviderData)
	require.True(t, ok)
	assert.Equal(t, "pop-1", data.PopulationID)
	assert.Equal(t, "idp-1", data.IdentityProviderID)
}

func TestGetPopulationDefaultIdentityProvider_NotFound(t *testing.T) {
	srv := newPopulationDefaultIdpMux(map[string]any{}, map[string]any{})
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := getPopulationDefaultIdentityProvider(testCtx(), c, "", "missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get population default identity provider")
	assert.Nil(t, result)
}

func TestGetPopulationDefaultIdentityProvider_ManagementClientUnavailable(t *testing.T) {
	c := &Client{}
	result, err := getPopulationDefaultIdentityProvider(testCtx(), c, "", "pop-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}
