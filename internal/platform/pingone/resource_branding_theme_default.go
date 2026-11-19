package pingone

import (
	"context"
	"fmt"

	"github.com/patrickcping/pingone-go-sdk-v2/management"
)

// pingone_branding_theme_default is derived from the branding themes list:
// the environment's default theme is the `default` flag on the theme itself
// (PUT .../branding/themes/{themeID}/default), which the provider models as
// a separate one-per-environment resource. The derived instance reuses the
// theme's struct — the processor reads Id (twice: the assignment key and
// branding_theme_id) and Default from it, with Environment.Id synthesized
// by the wrapper below.
func init() {
	registerResource("pingone_branding_theme_default", resourceHandler{
		list: listBrandingThemeDefault,
		get:  getBrandingThemeDefault,
	})
}

// brandingThemeDefaultData projects a default-flagged theme onto the
// branding_theme_default.yaml source paths. Field names must match the
// YAML source_paths exactly (findStructField matches case-sensitively):
// Id (assignment key + branding_theme_id), Environment.Id, Default.
type brandingThemeDefaultData struct {
	Environment struct {
		Id string
	}
	Id      string
	ThemeId string
	Default bool
}

func toBrandingThemeDefaultData(envID, themeID string, def bool) *brandingThemeDefaultData {
	d := &brandingThemeDefaultData{
		Id:      themeID,
		ThemeId: themeID,
		Default: def,
	}
	d.Environment.Id = envID
	return d
}

// listBrandingThemeDefault scans the themes list for the one flagged
// default and projects it.
func listBrandingThemeDefault(ctx context.Context, c *Client, _ string) ([]interface{}, error) {
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}

	envID := c.environmentID.String()
	var result []interface{}
	iterator := mgmt.BrandingThemesApi.ReadBrandingThemes(ctx, envID).Execute()
	for cursor, err := range iterator {
		if err != nil {
			return nil, fmt.Errorf("list branding theme default: %w", err)
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
			if themes[i].GetDefault() {
				result = append(result, toBrandingThemeDefaultData(envID, themes[i].GetId(), true))
			}
		}
	}
	return result, nil
}

// getBrandingThemeDefault reads the default flag for one theme via the
// dedicated default endpoint. resourceID is the theme ID.
func getBrandingThemeDefault(ctx context.Context, c *Client, _ string, resourceID string) (interface{}, error) {
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}
	defResp, _, err := mgmt.BrandingThemesApi.ReadBrandingThemeDefault(ctx, c.environmentID.String(), resourceID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get branding theme default: %w", err)
	}
	return toBrandingThemeDefaultData(c.environmentID.String(), resourceID, defResp.GetDefault()), nil
}

// compile-time interface check mirroring the sibling handlers' use of the
// SDK types (keeps the management import honest if the projection changes).
var _ = management.NewBrandingThemeDefault
