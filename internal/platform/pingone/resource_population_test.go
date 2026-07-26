package pingone

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/patrickcping/pingone-go-sdk-v2/management"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── dispatch registration ────────────────────────────────────────

func TestPopulationResourceRegistered(t *testing.T) {
	assert.True(t, isSupported("pingone_population"))
}

func TestPopulationResourceHandlerFunctions(t *testing.T) {
	h, ok := resourceHandlers["pingone_population"]
	require.True(t, ok)
	assert.NotNil(t, h.list)
	assert.NotNil(t, h.get)
}

// ── test helpers ─────────────────────────────────────────────────

// newTestManagementClient builds a management.APIClient whose requests are
// routed to the given httptest.Server, and a Client wired to use it via
// NewWithManagementClient (avoiding a real OAuth exchange).
func newTestManagementClient(t *testing.T, srv *httptest.Server, envID uuid.UUID) *Client {
	t.Helper()
	cfg := management.NewConfiguration()
	cfg.Servers = management.ServerConfigurations{
		{URL: srv.URL},
	}
	cfg.HTTPClient = srv.Client()
	mgmtClient := management.NewAPIClient(cfg)
	return NewWithManagementClient(nil, mgmtClient, envID)
}

// ── listPopulations ──────────────────────────────────────────────

func TestListPopulations_Success(t *testing.T) {
	envID := uuid.New()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"_embedded": map[string]interface{}{
				"populations": []map[string]interface{}{
					{"id": "pop-1", "name": "Population One"},
					{"id": "pop-2", "name": "Population Two"},
				},
			},
		})
	}))
	defer srv.Close()

	c := newTestManagementClient(t, srv, envID)

	result, err := listPopulations(context.Background(), c, "")
	require.NoError(t, err)
	require.Len(t, result, 2)

	pop0, ok := result[0].(*management.Population)
	require.True(t, ok)
	assert.Equal(t, "pop-1", pop0.GetId())
	assert.Equal(t, "Population One", pop0.GetName())

	pop1, ok := result[1].(*management.Population)
	require.True(t, ok)
	assert.Equal(t, "pop-2", pop1.GetId())
}

func TestListPopulations_Empty(t *testing.T) {
	envID := uuid.New()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"_embedded": map[string]interface{}{
				"populations": []map[string]interface{}{},
			},
		})
	}))
	defer srv.Close()

	c := newTestManagementClient(t, srv, envID)

	result, err := listPopulations(context.Background(), c, "")
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestListPopulations_NoEmbedded(t *testing.T) {
	envID := uuid.New()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{})
	}))
	defer srv.Close()

	c := newTestManagementClient(t, srv, envID)

	result, err := listPopulations(context.Background(), c, "")
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestListPopulations_APIError(t *testing.T) {
	envID := uuid.New()

	// Use 400 (not 429/500) — the SDK retries those with backoff up to
	// maxRetries times, which would make this test slow/hang-prone.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"message": "bad request"})
	}))
	defer srv.Close()

	c := newTestManagementClient(t, srv, envID)

	result, err := listPopulations(context.Background(), c, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list populations")
	assert.Nil(t, result)
}

func TestListPopulations_ManagementClientError(t *testing.T) {
	// A Client with neither managementClient nor managementCfg configured
	// causes c.management(ctx) to fail before any HTTP call is attempted.
	c := &Client{environmentID: uuid.New()}

	result, err := listPopulations(context.Background(), c, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}

// ── getPopulation ────────────────────────────────────────────────

func TestGetPopulation_Success(t *testing.T) {
	envID := uuid.New()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          "pop-1",
			"name":        "Population One",
			"description": "A test population",
		})
	}))
	defer srv.Close()

	c := newTestManagementClient(t, srv, envID)

	result, err := getPopulation(context.Background(), c, "", "pop-1")
	require.NoError(t, err)

	pop, ok := result.(*management.Population)
	require.True(t, ok)
	assert.Equal(t, "pop-1", pop.GetId())
	assert.Equal(t, "Population One", pop.GetName())
	assert.Equal(t, "A test population", pop.GetDescription())
}

func TestGetPopulation_NotFound(t *testing.T) {
	envID := uuid.New()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"message": "not found"})
	}))
	defer srv.Close()

	c := newTestManagementClient(t, srv, envID)

	result, err := getPopulation(context.Background(), c, "", "missing-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get population")
	assert.Nil(t, result)
}

func TestGetPopulation_EmptyResourceID(t *testing.T) {
	envID := uuid.New()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"message": "not found"})
	}))
	defer srv.Close()

	c := newTestManagementClient(t, srv, envID)

	result, err := getPopulation(context.Background(), c, "", "")
	require.Error(t, err)
	assert.Nil(t, result)
}

func TestGetPopulation_ManagementClientError(t *testing.T) {
	c := &Client{environmentID: uuid.New()}

	result, err := getPopulation(context.Background(), c, "", "pop-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}
