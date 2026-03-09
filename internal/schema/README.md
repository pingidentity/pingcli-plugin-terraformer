# Schema Package

Resource definition system for the terraformer.

## Purpose

This package provides the schema definition system that allows resources to be defined declaratively in YAML files. The schema system includes:

- **Type definitions** (`types.go`) - Go structs that map to YAML structure
- **Loader** (`loader.go`) - Loads and parses YAML definitions
- **Validator** (`validator.go`) - Validates definitions against rules
- **Registry** (`registry.go`) - Central registry of all resource definitions

## Usage

```go
// Load a single definition
def, err := schema.LoadDefinition("definitions/pingone-davinci/variable.yaml")

// Load all definitions from directory
registry := schema.NewRegistry()
err := registry.LoadFromDirectory("definitions/pingone-davinci")

// Get a definition
def, err := registry.Get("pingone_davinci_variable")
```

## Schema Structure

See [01_ARCHITECTURE_REDESIGN_V2.md](../../.github/prompts/redesign/01_ARCHITECTURE_REDESIGN_V2.md) for complete schema documentation.
