package davinci

import (
	"context"
	"fmt"

	"github.com/pingidentity/pingone-go-client/pingone"
)

// environmentData projects EnvironmentResponse + BillOfMaterials into the shape
// expected by the environment.yaml source_path values.
type environmentData struct {
	Id              string
	Name            string
	Description     *string
	Type            string
	Region          string
	License         environmentLicenseData
	Organization    environmentOrganizationData
	BillOfMaterials *environmentBOMData
}

type environmentLicenseData struct {
	Id string
}

type environmentOrganizationData struct {
	Id string
}

type environmentBOMData struct {
	SolutionType *string
	Products     []environmentProductData
}

type environmentProductData struct {
	Type       string
	Console    *environmentConsoleData
	Deployment *environmentDeploymentData
	Bookmarks  []environmentBookmarkData
	Tags       []string
}

type environmentConsoleData struct {
	Href *string
}

type environmentDeploymentData struct {
	Id string
}

type environmentBookmarkData struct {
	Name string
	Href string
}

func init() {
	registerResource("pingone_environment", resourceHandler{
		list: listEnvironments,
		get:  getEnvironment,
	})
}

// toEnvironmentData projects an EnvironmentResponse and optional BillOfMaterialsResponse
// into environmentData, matching the source_path structure in environment.yaml.
func toEnvironmentData(envResp *pingone.EnvironmentResponse, bomResp *pingone.EnvironmentBillOfMaterialsResponse) *environmentData {
	data := &environmentData{
		Id:          envResp.Id.String(),
		Name:        envResp.Name,
		Description: envResp.Description,
		Type:        string(envResp.Type),
		Region:      string(envResp.Region),
	}

	// License
	if envResp.License != nil {
		data.License = environmentLicenseData{
			Id: envResp.License.Id.String(),
		}
	}

	// Organization
	data.Organization = environmentOrganizationData{
		Id: envResp.Organization.Id.String(),
	}

	// BillOfMaterials
	if bomResp != nil {
		bom := &environmentBOMData{
			SolutionType: bomResp.SolutionType,
		}

		// Convert products
		if len(bomResp.Products) > 0 {
			bom.Products = make([]environmentProductData, 0, len(bomResp.Products))
			for _, product := range bomResp.Products {
				prod := environmentProductData{
					Type: string(product.Type),
				}

				// Console
				if product.Console != nil {
					prod.Console = &environmentConsoleData{
						Href: product.Console.Href,
					}
				}

				// Deployment
				if product.Deployment != nil {
					prod.Deployment = &environmentDeploymentData{
						Id: product.Deployment.Id.String(),
					}
				}

				// Bookmarks
				if len(product.Bookmarks) > 0 {
					prod.Bookmarks = make([]environmentBookmarkData, len(product.Bookmarks))
					for i, bm := range product.Bookmarks {
						prod.Bookmarks[i] = environmentBookmarkData{
							Name: bm.Name,
							Href: bm.Href,
						}
					}
				}

				// Tags
				if len(product.Tags) > 0 {
					prod.Tags = make([]string, len(product.Tags))
					copy(prod.Tags, product.Tags)
				}

				bom.Products = append(bom.Products, prod)
			}
		}

		data.BillOfMaterials = bom
	}

	return data
}

// listEnvironments fetches the target environment and returns it as a single-item slice.
func listEnvironments(ctx context.Context, c *Client, _ string) ([]interface{}, error) {
	envResp, _, err := c.apiClient.EnvironmentsApi.GetEnvironmentById(ctx, c.environmentID).Execute()
	if err != nil {
		return nil, fmt.Errorf("list environments: %w", err)
	}

	bomResp, _, err := c.apiClient.EnvironmentsApi.GetBillOfMaterialsByEnvironmentId(ctx, c.environmentID).Execute()
	if err != nil {
		return nil, fmt.Errorf("list environments: get bill of materials: %w", err)
	}

	data := toEnvironmentData(envResp, bomResp)
	return []interface{}{data}, nil
}

// getEnvironment fetches the target environment.
func getEnvironment(ctx context.Context, c *Client, _ string, _ string) (interface{}, error) {
	envResp, _, err := c.apiClient.EnvironmentsApi.GetEnvironmentById(ctx, c.environmentID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get environment: %w", err)
	}

	bomResp, _, err := c.apiClient.EnvironmentsApi.GetBillOfMaterialsByEnvironmentId(ctx, c.environmentID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get environment: get bill of materials: %w", err)
	}

	data := toEnvironmentData(envResp, bomResp)
	return data, nil
}
