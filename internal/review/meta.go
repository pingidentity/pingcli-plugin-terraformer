package review

import (
	"encoding/json"
	"fmt"
	"time"
)

// Meta is the bundle's provenance record (meta.json).
type Meta struct {
	// GeneratedAt is the bundle generation timestamp.
	GeneratedAt string `json:"generated_at"`

	// BaseRef is the git ref the diff was taken against.
	BaseRef string `json:"base_ref"`

	// Mode is "live" or "dry-run".
	Mode string `json:"mode"`

	// ResourceTypes lists every reviewed type.
	ResourceTypes []string `json:"resource_types"`

	// Changes is the detected change set (empty when types were passed
	// explicitly without detection).
	Changes []TypeChange `json:"changes,omitempty"`

	// PlatformCoreChanged notes a platform-wide change (client/dispatch).
	PlatformCoreChanged bool `json:"platform_core_changed,omitempty"`

	// TargetEnvironmentID records the live environment the export/snapshot
	// came from. Deliberately NOT credentials.
	TargetEnvironmentID string `json:"target_environment_id,omitempty"`

	// Region of the live environment.
	Region string `json:"region,omitempty"`

	// SkippedLive lists types excluded from the live phase (e.g. DaVinci in
	// an SSO-only environment) with the reason.
	SkippedLive map[string]string `json:"skipped_live,omitempty"`

	// SnapshotNote carries the snapshot-format caveat for tag-less structs.
	SnapshotNote string `json:"snapshot_note,omitempty"`
}

// WriteMeta renders meta.json content.
func WriteMeta(m Meta) ([]byte, error) {
	if m.GeneratedAt == "" {
		m.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if m.Mode == "" {
		m.Mode = "dry-run"
	}
	if m.SnapshotNote == "" {
		m.SnapshotNote = "Snapshots are JSON marshals of handler list results (post union-unwrap, post bootstrap-filter). Structs without json tags marshal with Go field names."
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal meta: %w", err)
	}
	return data, nil
}
