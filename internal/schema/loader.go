package schema

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Loader handles loading resource definitions from YAML files
type Loader struct {
	validator *Validator
}

// NewLoader creates a new definition loader
func NewLoader() *Loader {
	return &Loader{
		validator: NewValidator(),
	}
}

// LoadDefinition loads a single resource definition from a YAML file
func (l *Loader) LoadDefinition(path string) (*ResourceDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read definition file %s: %w", path, err)
	}

	var def ResourceDefinition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("failed to parse YAML in %s: %w", path, err)
	}

	if err := l.validator.Validate(&def); err != nil {
		return nil, fmt.Errorf("validation failed for %s: %w", path, err)
	}

	return &def, nil
}

// LoadFromDirectory loads all resource definitions from a directory
func (l *Loader) LoadFromDirectory(dir string) ([]*ResourceDefinition, error) {
	var definitions []*ResourceDefinition

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, "_test.yaml") {
			return nil
		}

		def, err := l.LoadDefinition(path)
		if err != nil {
			return fmt.Errorf("failed to load %s: %w", path, err)
		}

		// Skip disabled definitions
		if def.Metadata.Enabled != nil && !*def.Metadata.Enabled {
			return nil
		}

		definitions = append(definitions, def)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return definitions, nil
}

// LoadPlatformDefinitions loads all definitions for a specific platform
func (l *Loader) LoadPlatformDefinitions(baseDir, platform string) ([]*ResourceDefinition, error) {
	platformDir := filepath.Join(baseDir, platform)

	if _, err := os.Stat(platformDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("platform directory does not exist: %s", platformDir)
	}

	return l.LoadFromDirectory(platformDir)
}

// LoadFromFS loads all resource definitions from an fs.FS (e.g., embed.FS).
// The dir parameter is the root path within the FS to walk.
func (l *Loader) LoadFromFS(fsys fs.FS, dir string) ([]*ResourceDefinition, error) {
	var definitions []*ResourceDefinition

	err := fs.WalkDir(fsys, dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, "_test.yaml") {
			return nil
		}

		data, readErr := fs.ReadFile(fsys, path)
		if readErr != nil {
			return fmt.Errorf("failed to read %s: %w", path, readErr)
		}

		var def ResourceDefinition
		if unmarshalErr := yaml.Unmarshal(data, &def); unmarshalErr != nil {
			return fmt.Errorf("failed to parse YAML in %s: %w", path, unmarshalErr)
		}

		if validateErr := l.validator.Validate(&def); validateErr != nil {
			return fmt.Errorf("validation failed for %s: %w", path, validateErr)
		}

		// Skip disabled definitions
		if def.Metadata.Enabled != nil && !*def.Metadata.Enabled {
			return nil
		}

		definitions = append(definitions, &def)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return definitions, nil
}
