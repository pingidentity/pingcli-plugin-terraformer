package pingone

import (
	"context"
	"fmt"
	"strings"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
)

func init() {
	// API client dispatch.
	registerResource("pingone_branding_theme", resourceHandler{
		list: listBrandingThemes,
		get:  getBrandingTheme,
	})

	// Custom transform: derive the provider's use_default_background toggle
	// from the API's BackgroundType enum (no direct SDK field exists for it).
	registerTransform("handleBrandingThemeUseDefaultBackground", handleBrandingThemeUseDefaultBackground)
}

// listBrandingThemes lists all branding themes in the environment using the
// management SDK's paginated iterator.
func listBrandingThemes(ctx context.Context, c *Client, _ string) ([]interface{}, error) {
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}

	var result []interface{}
	iterator := mgmt.BrandingThemesApi.ReadBrandingThemes(ctx, c.environmentID.String()).Execute()
	for cursor, err := range iterator {
		if err != nil {
			return nil, fmt.Errorf("list branding themes: %w", err)
		}
		embedded, ok := cursor.EntityArray.GetEmbeddedOk()
		if !ok || embedded == nil {
			continue
		}
		themes, ok := embedded.GetThemesOk()
		if !ok {
			continue
		}
		for i := range themes {
			result = append(result, &themes[i])
		}
	}
	return result, nil
}

// getBrandingTheme retrieves a single branding theme by ID via the management SDK.
func getBrandingTheme(ctx context.Context, c *Client, _ string, resourceID string) (interface{}, error) {
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}
	theme, _, err := mgmt.BrandingThemesApi.ReadOneBrandingTheme(ctx, c.environmentID.String(), resourceID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get branding theme: %w", err)
	}
	return theme, nil
}

// handleBrandingThemeUseDefaultBackground converts the API's BackgroundType
// enum string ("NONE", "COLOR", "IMAGE", "DEFAULT") into the boolean the
// pingone_branding_theme provider schema expects for use_default_background.
// Only "DEFAULT" maps to true; every other value (and absence) maps to
// false, which the processor's empty-value skip then drops from output
// since the attribute is optional.
func handleBrandingThemeUseDefaultBackground(value interface{}, _ interface{}, _ *schema.AttributeDefinition, _ *schema.ResourceDefinition) (interface{}, error) {
	s, ok := value.(string)
	if !ok {
		return nil, nil
	}
	if strings.EqualFold(s, "DEFAULT") {
		return true, nil
	}
	return nil, nil
}
