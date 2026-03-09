// Package formatters defines the OutputFormatter interface and a factory
// function for constructing concrete formatter implementations.
package formatters

import (
	"fmt"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/core"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/formatters/hcl"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/formatters/tfjson"
	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
)

// Format name constants.
const (
	FormatHCL    = "hcl"
	FormatTFJSON = "tfjson"
)

// FormatOptions controls rendering behavior for all output formatters.
type FormatOptions struct {
	// SkipDependencies outputs raw UUIDs instead of Terraform references.
	SkipDependencies bool
	// EnvironmentID is the raw environment UUID used when SkipDependencies is true.
	EnvironmentID string
}

// OutputFormatter defines the contract for converting processed resource data
// into a specific output format (HCL, JSON, etc.).
type OutputFormatter interface {
	// Format generates a single resource block from resource data.
	Format(data *core.ResourceData, def *schema.ResourceDefinition, opts FormatOptions) (string, error)
	// FormatList generates multiple resource blocks from a slice of resource data.
	FormatList(dataList []*core.ResourceData, def *schema.ResourceDefinition, opts FormatOptions) (string, error)
	// FormatImportBlock generates a Terraform import block for a resource.
	FormatImportBlock(data *core.ResourceData, def *schema.ResourceDefinition, environmentID string) (string, error)
	// FileExtension returns the file extension for the output format (e.g. ".tf").
	FileExtension() string
}

// NewFormatter returns an OutputFormatter for the given format name.
// Supported formats: FormatHCL, FormatTFJSON.
func NewFormatter(format string) (OutputFormatter, error) {
	switch format {
	case FormatHCL:
		return &hclAdapter{inner: hcl.NewFormatter()}, nil
	case FormatTFJSON:
		return &tfjsonAdapter{inner: tfjson.NewFormatter()}, nil
	default:
		return nil, fmt.Errorf("unknown output format: %s", format)
	}
}

// hclAdapter wraps hcl.Formatter to satisfy the OutputFormatter interface,
// bridging the hcl-package FormatOptions to the shared FormatOptions type.
type hclAdapter struct {
	inner *hcl.Formatter
}

func (a *hclAdapter) Format(data *core.ResourceData, def *schema.ResourceDefinition, opts FormatOptions) (string, error) {
	return a.inner.Format(data, def, hcl.FormatOptions{
		SkipDependencies: opts.SkipDependencies,
		EnvironmentID:    opts.EnvironmentID,
	})
}

func (a *hclAdapter) FormatList(dataList []*core.ResourceData, def *schema.ResourceDefinition, opts FormatOptions) (string, error) {
	return a.inner.FormatList(dataList, def, hcl.FormatOptions{
		SkipDependencies: opts.SkipDependencies,
		EnvironmentID:    opts.EnvironmentID,
	})
}

func (a *hclAdapter) FormatImportBlock(data *core.ResourceData, def *schema.ResourceDefinition, environmentID string) (string, error) {
	return a.inner.FormatImportBlock(data, def, environmentID)
}

func (a *hclAdapter) FileExtension() string {
	return ".tf"
}

// tfjsonAdapter wraps tfjson.Formatter to satisfy the OutputFormatter interface,
// bridging the tfjson-package FormatOptions to the shared FormatOptions type.
type tfjsonAdapter struct {
	inner *tfjson.Formatter
}

func (a *tfjsonAdapter) Format(data *core.ResourceData, def *schema.ResourceDefinition, opts FormatOptions) (string, error) {
	return a.inner.Format(data, def, tfjson.FormatOptions{
		SkipDependencies: opts.SkipDependencies,
		EnvironmentID:    opts.EnvironmentID,
	})
}

func (a *tfjsonAdapter) FormatList(dataList []*core.ResourceData, def *schema.ResourceDefinition, opts FormatOptions) (string, error) {
	return a.inner.FormatList(dataList, def, tfjson.FormatOptions{
		SkipDependencies: opts.SkipDependencies,
		EnvironmentID:    opts.EnvironmentID,
	})
}

func (a *tfjsonAdapter) FormatImportBlock(data *core.ResourceData, def *schema.ResourceDefinition, environmentID string) (string, error) {
	return a.inner.FormatImportBlock(data, def, environmentID)
}

func (a *tfjsonAdapter) FileExtension() string {
	return ".tf.json"
}
