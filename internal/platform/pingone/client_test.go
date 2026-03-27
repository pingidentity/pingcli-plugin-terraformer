package pingone

import (
	"testing"

	"github.com/google/uuid"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/clients"
	"github.com/stretchr/testify/assert"
)

func TestClientImplementsInterface(t *testing.T) {
	var _ clients.APIClient = (*Client)(nil)
}

func TestPlatform(t *testing.T) {
	c := &Client{}
	assert.Equal(t, "pingone", c.Platform())
}

func TestNewClient(t *testing.T) {
	envID := uuid.New()
	c := New(nil, envID)
	assert.NotNil(t, c)
	assert.Nil(t, c.apiClient)
	assert.Equal(t, envID, c.environmentID)
}
