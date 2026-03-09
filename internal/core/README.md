# Core Package

Core processing engine for resource conversion.

## Purpose

This package provides the generic resource processing engine that:

- Interprets resource definitions from the schema package
- Extracts attributes from API responses
- Applies transformations
- Manages dependencies
- Coordinates the entire conversion pipeline

## Components

- **Processor** (`processor.go`) - Main resource processor
- **Orchestrator** (`orchestrator.go`) - Multi-resource export coordination
- **Transforms** (`transforms.go`) - Attribute transformation functions
- **Custom Handlers** (`custom_handlers.go`) - Complex resource handlers

## Usage

```go
processor := core.NewResourceProcessor(registry)
result, err := processor.ProcessResource(ctx, "pingone_davinci_variable", data, opts)
```
