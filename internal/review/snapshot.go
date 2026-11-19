package review

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Snapshotter captures raw API snapshots for the review set.
type Snapshotter struct {
	// Client is the constructed API client.
	Client interface {
		ListResources(ctx context.Context, resourceType string, envID string) ([]interface{}, error)
	}

	// EnvironmentID is the target environment.
	EnvironmentID string
}

// WriteSnapshots marshals each closure type's raw list response to
// snapshot/<type>.json under outDir. A pathological type (e.g. a marshal
// panic) degrades to snapshot/<type>.json.error instead of failing the
// bundle. Returns the list of types snapshotted.
func (s *Snapshotter) WriteSnapshots(ctx context.Context, outDir string, types []string) ([]string, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("create snapshot dir: %w", err)
	}

	var done []string
	for _, rt := range types {
		name := filepath.Join(outDir, rt+".json")
		if err := s.writeOne(ctx, name, rt); err != nil {
			errPath := filepath.Join(outDir, rt+".json.error")
			_ = os.WriteFile(errPath, []byte(err.Error()), 0o644)
			continue
		}
		done = append(done, rt)
	}
	return done, nil
}

// writeOne captures and marshals one type's list response. A recover() guard
// converts panics (pathological SDK marshal behavior) into errors.
func (s *Snapshotter) writeOne(ctx context.Context, path, resourceType string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic marshaling %s: %v", resourceType, r)
		}
	}()

	raw, err := s.Client.ListResources(ctx, resourceType, s.EnvironmentID)
	if err != nil {
		return fmt.Errorf("list %s: %w", resourceType, err)
	}

	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", resourceType, err)
	}

	return os.WriteFile(path, data, 0o644)
}
