package review

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SecretScanGuard verifies no secret material leaked into the bundle.
// bundleDir is scanned recursively; secrets are the caller-supplied
// credential values that must never appear in an artifact.
func SecretScanGuard(bundleDir string, secrets ...string) error {
	if len(secrets) == 0 {
		return nil
	}

	var hits []string
	err := filepath.Walk(bundleDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil // unreadable files are not hits
		}
		for _, s := range secrets {
			if s != "" && bytes.Contains(data, []byte(s)) {
				if rel, err := filepath.Rel(bundleDir, path); err == nil {
					hits = append(hits, rel)
				} else {
					hits = append(hits, path)
				}
				break
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("secret scan: %w", err)
	}

	if len(hits) > 0 {
		// Destroy the bundle rather than risk uploading it.
		_ = os.RemoveAll(bundleDir)
		return fmt.Errorf("SECRET LEAK: bundle deleted; client secret found in %s", joinStrings(hits))
	}
	return nil
}

// CredentialEnv holds the live-phase credentials, parsed from the
// PINGCLI_PINGONE_* environment variables (the same contract
// tests/regression-provisioned/run-local.sh exports after provisioning).
type CredentialEnv struct {
	WorkerEnvironmentID string
	ExportEnvironmentID string
	RegionCode          string
	ClientID            string
	ClientSecret        string
}

// CredentialsFromEnv reads the credential environment variables. Returns
// (creds, true) when all required vars are present, or (zero, false) for
// dry-run mode.
func CredentialsFromEnv(getenv func(string) string) (CredentialEnv, bool) {
	c := CredentialEnv{
		WorkerEnvironmentID: getenv("PINGCLI_PINGONE_ENVIRONMENT_ID"),
		ExportEnvironmentID: getenv("PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID"),
		RegionCode:          getenv("PINGCLI_PINGONE_REGION_CODE"),
		ClientID:            getenv("PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID"),
		ClientSecret:        getenv("PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET"),
	}
	if c.ExportEnvironmentID == "" {
		c.ExportEnvironmentID = c.WorkerEnvironmentID
	}
	if c.RegionCode == "" {
		c.RegionCode = "NA"
	}
	if c.WorkerEnvironmentID == "" || c.ClientID == "" || c.ClientSecret == "" {
		return CredentialEnv{}, false
	}
	return c, true
}

// EnvID returns the export-target environment ID (the live environment the
// bundle's contents came from).
func (c CredentialEnv) EnvID() string { return c.ExportEnvironmentID }

// DetectOnlyJSON renders the detect-only output: the types the bundle would
// review plus their closure description.
func DetectOnlyJSON(types []string, closure *ClosureResult) ([]byte, error) {
	payload := map[string]interface{}{
		"resource_types": types,
	}
	if closure != nil {
		payload["closure"] = closure.Types
		payload["skipped_live"] = closure.SkippedLive
		payload["fallback_targets"] = closure.FallbackTargets
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, err
	}
	return data, nil
}
