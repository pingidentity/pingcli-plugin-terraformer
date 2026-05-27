package pingone

import (
	"strings"
	"testing"

	"github.com/pingidentity/pingone-go-client/config"
	"github.com/pingidentity/pingone-go-client/pingone"
)

// TestUserAgentSuffix_AppendedCorrectly verifies that NewConfiguration followed
// by AppendUserAgent produces a User-Agent string that ends with
// "pingcli-plugin-terraformer/<version>" and retains the SDK default prefix.
func TestUserAgentSuffix_AppendedCorrectly(t *testing.T) {
	tests := []struct {
		name            string
		version         string
		expectedSuffix  string
		expectedContain string
	}{
		{
			name:            "dev version",
			version:         "dev",
			expectedSuffix:  "pingcli-plugin-terraformer/dev",
			expectedContain: "pingtools pingone-go-client/",
		},
		{
			name:            "semver release version",
			version:         "v1.2.3",
			expectedSuffix:  "pingcli-plugin-terraformer/v1.2.3",
			expectedContain: "pingtools pingone-go-client/",
		},
		{
			name:            "empty version produces trailing slash",
			version:         "",
			expectedSuffix:  "pingcli-plugin-terraformer/",
			expectedContain: "pingtools",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serviceCfg := config.NewConfiguration()
			cfg := pingone.NewConfiguration(serviceCfg)

			// Capture the base UserAgent before modification.
			baseUA := cfg.UserAgent
			if !strings.Contains(baseUA, "pingtools") {
				t.Errorf("SDK default UserAgent does not contain 'pingtools': %q", baseUA)
			}

			// Apply the same transformation as NewFromCredentials.
			cfg.AppendUserAgent("pingcli-plugin-terraformer/" + tt.version)

			ua := cfg.UserAgent

			// Must still contain the SDK prefix.
			if !strings.Contains(ua, tt.expectedContain) {
				t.Errorf("UserAgent %q does not contain expected prefix %q", ua, tt.expectedContain)
			}

			// Must end with the tool suffix.
			if !strings.HasSuffix(ua, tt.expectedSuffix) {
				t.Errorf("UserAgent %q does not end with expected suffix %q", ua, tt.expectedSuffix)
			}

			// The suffix must be space-appended, not concatenated directly.
			toolIdx := strings.Index(ua, "pingcli-plugin-terraformer/")
			if toolIdx > 0 && ua[toolIdx-1] != ' ' {
				t.Errorf("Expected space before 'pingcli-plugin-terraformer/' in UserAgent %q", ua)
			}
		})
	}
}

// TestUserAgentSuffix_DoesNotOverwriteBase verifies that AppendUserAgent never
// overwrites the SDK default string — only appends to it.
func TestUserAgentSuffix_DoesNotOverwriteBase(t *testing.T) {
	serviceCfg := config.NewConfiguration()
	cfg := pingone.NewConfiguration(serviceCfg)
	baseUA := cfg.UserAgent

	cfg.AppendUserAgent("pingcli-plugin-terraformer/dev")

	ua := cfg.UserAgent

	// The result must begin with the original base string.
	if !strings.HasPrefix(ua, baseUA) {
		t.Errorf("UserAgent after AppendUserAgent lost the original base.\nBase:   %q\nResult: %q", baseUA, ua)
	}

	// Length must strictly increase.
	if len(ua) <= len(baseUA) {
		t.Errorf("UserAgent length did not increase after AppendUserAgent.\nBefore: %d\nAfter:  %d", len(baseUA), len(ua))
	}
}

// TestNewFromCredentials_SignatureAcceptsVersionParam is a compile-time check
// encoded as a runtime test: it verifies that NewFromCredentials accepts a seventh
// string argument. If the signature ever regresses to 6 arguments this file will
// not compile.
func TestNewFromCredentials_SignatureAcceptsVersionParam(t *testing.T) {
	// Passing an empty workerEnvID ensures we hit the first validation guard
	// immediately, without making any network calls.
	_, err := NewFromCredentials(nil, "", "target", "NA", "cid", "csecret", "v1.0.0") //nolint:staticcheck
	if err == nil {
		t.Error("Expected error for empty workerEnvID, got nil")
	}
	if !strings.Contains(err.Error(), "auth environment ID is required") {
		t.Errorf("Expected 'auth environment ID is required' error, got: %v", err)
	}
}
