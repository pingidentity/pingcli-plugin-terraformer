package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pingidentity/pingcli-plugin-terraformer/internal/schema"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: validate-definitions <directory>\n")
		os.Exit(1)
	}

	dir := os.Args[1]
	info, err := os.Stat(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "error: %s is not a directory\n", dir)
		os.Exit(1)
	}

	loader := schema.NewLoader()
	validator := schema.NewValidator()

	var files []string
	if walkErr := filepath.Walk(dir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".yaml" || ext == ".yml" {
			if strings.Contains(filepath.Base(path), "_test") {
				return nil
			}
			files = append(files, path)
		}
		return nil
	}); walkErr != nil {
		fmt.Fprintf(os.Stderr, "error walking directory: %v\n", walkErr)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "no YAML files found in %s\n", dir)
		os.Exit(1)
	}

	totalErrors := 0
	totalPass := 0

	fmt.Printf("Validating %d definition(s) in %s\n\n", len(files), dir)

	for _, file := range files {
		relPath, _ := filepath.Rel(dir, file)
		if relPath == "" {
			relPath = file
		}

		def, loadErr := loader.LoadDefinition(file)
		if loadErr != nil {
			fmt.Printf("  FAIL  %s\n", relPath)
			fmt.Printf("        load error: %v\n", loadErr)
			totalErrors++
			continue
		}

		if valErr := validator.Validate(def); valErr != nil {
			fmt.Printf("  FAIL  %s (%s)\n", relPath, def.Metadata.ResourceType)
			fmt.Printf("        %v\n", valErr)
			totalErrors++
			continue
		}

		attrCount := countAttributes(def)
		fmt.Printf("  PASS  %s (%s) - %d attributes\n", relPath, def.Metadata.ResourceType, attrCount)
		totalPass++
	}

	fmt.Printf("\n%d passed, %d failed, %d total\n", totalPass, totalErrors, totalPass+totalErrors)

	if totalErrors > 0 {
		os.Exit(1)
	}
}

func countAttributes(def *schema.ResourceDefinition) int {
	count := 0
	for _, attr := range def.Attributes {
		count += countAttrRecursive(attr)
	}
	return count
}

func countAttrRecursive(attr schema.AttributeDefinition) int {
	count := 1
	for _, nested := range attr.NestedAttributes {
		count += countAttrRecursive(nested)
	}
	return count
}
