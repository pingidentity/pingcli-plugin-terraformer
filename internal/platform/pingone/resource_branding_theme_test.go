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

func TestBrandingThemeResourceRegistered(t *testing.T) {
	assert.True(t, isSupported("pingone_branding_theme"))
}

func TestBrandingThemeResourceHandlerFunctions(t *testing.T) {
	h, ok := resourceHandlers["pingone_branding_theme"]
	require.True(t, ok)
	assert.NotNil(t, h.list)
	assert.NotNil(t, h.get)
}

// ── test helpers ─────────────────────────────────────────────────

// newTestBrandingThemeManagementClient builds a management.APIClient whose requests
// are routed to the given httptest.Server, and a Client wired to use it via
// NewWithManagementClient (avoiding a real OAuth exchange).
func newTestBrandingThemeManagementClient(t *testing.T, srv *httptest.Server, envID uuid.UUID) *Client {
	t.Helper()
	cfg := management.NewConfiguration()
	cfg.Servers = management.ServerConfigurations{
		{URL: srv.URL},
	}
	cfg.HTTPClient = srv.Client()
	mgmtClient := management.NewAPIClient(cfg)
	return NewWithManagementClient(nil, mgmtClient, envID)
}

// ── listBrandingThemes ───────────────────────────────────────────

func TestListBrandingThemes_Success(t *testing.T) {
	envID := uuid.New()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"_embedded": map[string]interface{}{
				"themes": []map[string]interface{}{
					{"id": "theme-1", "template": "default", "configuration": map[string]interface{}{"name": "Theme One"}},
					{"id": "theme-2", "template": "focus", "configuration": map[string]interface{}{"name": "Theme Two"}},
				},
			},
		})
	}))
	defer srv.Close()

	c := newTestBrandingThemeManagementClient(t, srv, envID)

	result, err := listBrandingThemes(context.Background(), c, "")
	require.NoError(t, err)
	require.Len(t, result, 2)

	theme0, ok := result[0].(*management.BrandingTheme)
	require.True(t, ok)
	assert.Equal(t, "theme-1", theme0.GetId())

	theme1, ok := result[1].(*management.BrandingTheme)
	require.True(t, ok)
	assert.Equal(t, "theme-2", theme1.GetId())
}

func TestListBrandingThemes_Empty(t *testing.T) {
	envID := uuid.New()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"_embedded": map[string]interface{}{
				"themes": []map[string]interface{}{},
			},
		})
	}))
	defer srv.Close()

	c := newTestBrandingThemeManagementClient(t, srv, envID)

	result, err := listBrandingThemes(context.Background(), c, "")
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestListBrandingThemes_NoEmbedded(t *testing.T) {
	envID := uuid.New()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{})
	}))
	defer srv.Close()

	c := newTestBrandingThemeManagementClient(t, srv, envID)

	result, err := listBrandingThemes(context.Background(), c, "")
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestListBrandingThemes_APIError(t *testing.T) {
	envID := uuid.New()

	// Use 400 (not 429/500) — the SDK retries those with backoff up to
	// maxRetries times, which would make this test slow/hang-prone.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"message": "bad request"})
	}))
	defer srv.Close()

	c := newTestBrandingThemeManagementClient(t, srv, envID)

	result, err := listBrandingThemes(context.Background(), c, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list branding themes")
	assert.Nil(t, result)
}

func TestListBrandingThemes_ManagementClientError(t *testing.T) {
	// A Client with neither managementClient nor managementCfg configured
	// causes c.management(ctx) to fail before any HTTP call is attempted.
	c := &Client{environmentID: uuid.New()}

	result, err := listBrandingThemes(context.Background(), c, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}

// ── getBrandingTheme ─────────────────────────────────────────────

func TestGetBrandingTheme_Success(t *testing.T) {
	envID := uuid.New()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       "theme-1",
			"template": "default",
			"default":  true,
			"configuration": map[string]interface{}{
				"name":             "Theme One",
				"backgroundType":   "COLOR",
				"backgroundColor":  "#FFFFFF",
				"bodyTextColor":    "#000000",
				"buttonColor":      "#000000",
				"buttonTextColor":  "#FFFFFF",
				"cardColor":        "#FFFFFF",
				"headingTextColor": "#000000",
				"linkTextColor":    "#0000FF",
				"logoType":         "NONE",
			},
		})
	}))
	defer srv.Close()

	c := newTestBrandingThemeManagementClient(t, srv, envID)

	result, err := getBrandingTheme(context.Background(), c, "", "theme-1")
	require.NoError(t, err)

	theme, ok := result.(*management.BrandingTheme)
	require.True(t, ok)
	assert.Equal(t, "theme-1", theme.GetId())
	assert.True(t, theme.GetDefault())
	config := theme.GetConfiguration()
	assert.Equal(t, "Theme One", config.GetName())
}

func TestGetBrandingTheme_NotFound(t *testing.T) {
	envID := uuid.New()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"message": "not found"})
	}))
	defer srv.Close()

	c := newTestBrandingThemeManagementClient(t, srv, envID)

	result, err := getBrandingTheme(context.Background(), c, "", "missing-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get branding theme")
	assert.Nil(t, result)
}

func TestGetBrandingTheme_EmptyResourceID(t *testing.T) {
	envID := uuid.New()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"message": "not found"})
	}))
	defer srv.Close()

	c := newTestBrandingThemeManagementClient(t, srv, envID)

	result, err := getBrandingTheme(context.Background(), c, "", "")
	require.Error(t, err)
	assert.Nil(t, result)
}

func TestGetBrandingTheme_ManagementClientError(t *testing.T) {
	c := &Client{environmentID: uuid.New()}

	result, err := getBrandingTheme(context.Background(), c, "", "theme-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}

// ── handleBrandingThemeUseDefaultBackground ─────────────────────

func TestHandleBrandingThemeUseDefaultBackground(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  interface{}
	}{
		{name: "DEFAULT maps to true", value: "DEFAULT", want: true},
		{name: "lowercase default maps to true", value: "default", want: true},
		{name: "COLOR maps to nil", value: "COLOR", want: nil},
		{name: "IMAGE maps to nil", value: "IMAGE", want: nil},
		{name: "NONE maps to nil", value: "NONE", want: nil},
		{name: "empty string maps to nil", value: "", want: nil},
		{name: "non-string value maps to nil", value: 123, want: nil},
		{name: "nil value maps to nil", value: nil, want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := handleBrandingThemeUseDefaultBackground(tt.value, nil, nil, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
