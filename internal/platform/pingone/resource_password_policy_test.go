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

func TestPasswordPolicyResourceRegistered(t *testing.T) {
	assert.True(t, isSupported("pingone_password_policy"))
}

func TestPasswordPolicyResourceHandlerFunctions(t *testing.T) {
	h, ok := resourceHandlers["pingone_password_policy"]
	require.True(t, ok)
	assert.NotNil(t, h.list)
	assert.NotNil(t, h.get)
}

func TestListPasswordPolicies_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"_embedded": map[string]interface{}{
				"passwordPolicies": []map[string]interface{}{
					{"id": "pp-1", "name": "Policy One"},
					{"id": "pp-2", "name": "Policy Two"},
				},
			},
		})
	}))
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := listPasswordPolicies(context.Background(), c, "")
	require.NoError(t, err)
	require.Len(t, result, 2)

	p0, ok := result[0].(*management.PasswordPolicy)
	require.True(t, ok)
	assert.Equal(t, "pp-1", p0.GetId())
}

func TestListPasswordPolicies_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{})
	}))
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := listPasswordPolicies(context.Background(), c, "")
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestListPasswordPolicies_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"message": "bad request"})
	}))
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := listPasswordPolicies(context.Background(), c, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list password policies")
	assert.Nil(t, result)
}

func TestListPasswordPolicies_ManagementClientUnavailable(t *testing.T) {
	c := &Client{environmentID: uuid.New()}
	result, err := listPasswordPolicies(context.Background(), c, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}

func TestGetPasswordPolicy_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "pp-1",
			"name": "Policy One",
		})
	}))
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := getPasswordPolicy(context.Background(), c, "", "pp-1")
	require.NoError(t, err)

	p, ok := result.(*management.PasswordPolicy)
	require.True(t, ok)
	assert.Equal(t, "pp-1", p.GetId())
	assert.Equal(t, "Policy One", p.GetName())
}

func TestGetPasswordPolicy_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"message": "not found"})
	}))
	defer srv.Close()

	mgmt := newTestResourceManagementClient(srv.URL)
	c := NewWithManagementClient(nil, mgmt, uuid.New())

	result, err := getPasswordPolicy(context.Background(), c, "", "missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get password policy")
	assert.Nil(t, result)
}

func TestGetPasswordPolicy_ManagementClientUnavailable(t *testing.T) {
	c := &Client{environmentID: uuid.New()}
	result, err := getPasswordPolicy(context.Background(), c, "", "pp-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}
