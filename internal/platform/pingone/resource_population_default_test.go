package pingone

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/patrickcping/pingone-go-sdk-v2/management"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPopulationDefaultResourceRegistered(t *testing.T) {
	assert.True(t, isSupported("pingone_population_default"))
}

func TestPopulationDefaultResourceHandlerFunctions(t *testing.T) {
	h, ok := resourceHandlers["pingone_population_default"]
	assert.True(t, ok)
	assert.NotNil(t, h.list)
	assert.NotNil(t, h.get)
}

func TestListPopulationDefault(t *testing.T) {
	tests := []struct {
		name         string
		responseBody map[string]any
		wantLen      int
		wantID       string
	}{
		{
			name: "default population found among several",
			responseBody: map[string]any{
				"_embedded": map[string]any{
					"populations": []any{
						map[string]any{"id": "pop-1", "name": "Non-default"},
						map[string]any{"id": "pop-2", "name": "Default Pop", "default": true},
					},
				},
			},
			wantLen: 1,
			wantID:  "pop-2",
		},
		{
			name: "no default population returns empty",
			responseBody: map[string]any{
				"_embedded": map[string]any{
					"populations": []any{
						map[string]any{"id": "pop-1", "name": "Non-default"},
					},
				},
			},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(tt.responseBody)
			}))
			defer srv.Close()

			mgmt := newTestResourceManagementClient(srv.URL)
			c := NewWithManagementClient(nil, mgmt, uuid.New())

			result, err := listPopulationDefault(testCtx(), c, "")
			require.NoError(t, err)
			require.Len(t, result, tt.wantLen)

			if tt.wantLen > 0 {
				pop, ok := result[0].(*management.Population)
				require.True(t, ok, "expected *management.Population, got %T", result[0])
				assert.Equal(t, tt.wantID, pop.GetId())
			}
		})
	}
}

func TestListPopulationDefault_ManagementClientUnavailable(t *testing.T) {
	c := &Client{}
	result, err := listPopulationDefault(testCtx(), c, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}

func TestGetPopulationDefault(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"_embedded": map[string]any{
				"populations": []any{
					map[string]any{"id": "pop-1", "name": "Default Pop", "default": true},
				},
			},
		})
	}))
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := getPopulationDefault(testCtx(), c, "", "")
	require.NoError(t, err)

	pop, ok := result.(*management.Population)
	require.True(t, ok)
	assert.Equal(t, "pop-1", pop.GetId())
}

func TestGetPopulationDefault_NoneFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"_embedded": map[string]any{
				"populations": []any{
					map[string]any{"id": "pop-1", "name": "Non-default"},
				},
			},
		})
	}))
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := getPopulationDefault(testCtx(), c, "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no default population found")
	assert.Nil(t, result)
}

func TestGetPopulationDefault_ManagementClientUnavailable(t *testing.T) {
	c := &Client{}
	result, err := getPopulationDefault(testCtx(), c, "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}
