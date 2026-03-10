# Formatters Package

Output formatters for different target formats.

## Purpose

Convert processed resources into target output formats (HCL, Terraform JSON, Pulumi, etc.).

## Formatters

- **HCL** (`hcl/`) - Terraform HCL format (primary, `.tf` files)
- **tfjson** (`tfjson/`) - Terraform JSON configuration syntax (`.tf.json` files)

## Interface

All formatters implement the `OutputFormatter` interface defined in `formatter.go`:

```go
type OutputFormatter interface {
    Format(data *core.ResourceData, def *schema.ResourceDefinition, opts FormatOptions) (string, error)
    FormatList(dataList []*core.ResourceData, def *schema.ResourceDefinition, opts FormatOptions) (string, error)
    FormatImportBlock(data *core.ResourceData, def *schema.ResourceDefinition, environmentID string) (string, error)
    FileExtension() string
}
```

## Factory

Use `formatters.NewFormatter(format)` to instantiate a formatter:

```go
f, err := formatters.NewFormatter("hcl")     // returns HCL formatter
f, err := formatters.NewFormatter("tfjson")   // returns tfjson formatter
```
