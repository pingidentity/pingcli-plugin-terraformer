package pingone

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrandingThemeDefaultResourceRegistered(t *testing.T) {
	assert.True(t, isSupported("pingone_branding_theme_default"))
}

func TestBrandingThemeDefaultResourceHandlerFunctions(t *testing.T) {
	h, ok := resourceHandlers["pingone_branding_theme_default"]
	assert.True(t, ok)
	assert.NotNil(t, h.list)
	assert.NotNil(t, h.get)
}

func TestToBrandingThemeDefaultData(t *testing.T) {
	d := toBrandingThemeDefaultData("env-1", "theme-1", true)
	require.NotNil(t, d)
	// The assignment's ID is the owning theme's ID, and branding_theme_id
	// carries the same value so the processor's references_type resolution
	// links the derived resource to its pingone_branding_theme.
	assert.Equal(t, "theme-1", d.Id)
	assert.Equal(t, "theme-1", d.ThemeId)
	assert.Equal(t, "env-1", d.Environment.Id)
	assert.True(t, d.Default)
}
