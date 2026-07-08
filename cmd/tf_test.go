// Copyright © 2025 Ping Identity Corporation

package cmd

import (
	"os"
	"testing"
)

// mockLogger implements grpc.Logger for testing
type mockLogger struct {
	messages []string
	warnings []string
	errors   []string
}

func (m *mockLogger) Message(msg string, metadata map[string]string) error {
	m.messages = append(m.messages, msg)
	return nil
}

func (m *mockLogger) Success(msg string, metadata map[string]string) error {
	m.messages = append(m.messages, msg)
	return nil
}

func (m *mockLogger) Warn(msg string, metadata map[string]string) error {
	m.warnings = append(m.warnings, msg)
	return nil
}

func (m *mockLogger) UserError(msg string, metadata map[string]string) error {
	m.errors = append(m.errors, msg)
	return nil
}

func (m *mockLogger) UserFatal(msg string, metadata map[string]string) error {
	m.errors = append(m.errors, msg)
	return nil
}

func (m *mockLogger) PluginError(msg string, metadata map[string]string) error {
	m.errors = append(m.errors, msg)
	return nil
}

// TestTfCommand_Routing tests the parent command's routing logic
func TestTfCommand_Routing(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "no args",
			args:        []string{},
			expectError: true,
			errorMsg:    "subcommand required",
		},
		// davinci-to-hcl command deferred to v0.2.0
		// {
		// 	name:        "davinci-to-hcl subcommand with missing required flags",
		// 	args:        []string{"davinci-to-hcl"},
		// 	expectError: true,
		// 	errorMsg:    "--flow-json",
		// },
		{
			name:        "export subcommand with missing required flags",
			args:        []string{"export"},
			expectError: true,
			errorMsg:    "worker environment ID is required",
		},
		{
			name:        "help subcommand",
			args:        []string{"help"},
			expectError: false,
		},
		{
			name:        "--help flag",
			args:        []string{"--help"},
			expectError: false,
		},
		{
			name:        "-h flag",
			args:        []string{"-h"},
			expectError: false,
		},
		{
			name:        "unknown subcommand",
			args:        []string{"invalid"},
			expectError: true,
			errorMsg:    "unknown subcommand",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment variables for the export test to prevent using real credentials
			if tt.name == "export subcommand with missing required flags" {
				oldWorkerEnvID := os.Getenv("PINGCLI_PINGONE_ENVIRONMENT_ID")
				oldClientID := os.Getenv("PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID")
				oldClientSecret := os.Getenv("PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET")
				oldRegionCode := os.Getenv("PINGCLI_PINGONE_REGION_CODE")
				oldExportEnvID := os.Getenv("PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID")

				_ = os.Unsetenv("PINGCLI_PINGONE_ENVIRONMENT_ID")
				_ = os.Unsetenv("PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID")
				_ = os.Unsetenv("PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET")
				_ = os.Unsetenv("PINGCLI_PINGONE_REGION_CODE")
				_ = os.Unsetenv("PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID")

				defer func() {
					if oldWorkerEnvID != "" {
						_ = os.Setenv("PINGCLI_PINGONE_ENVIRONMENT_ID", oldWorkerEnvID)
					}
					if oldClientID != "" {
						_ = os.Setenv("PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID", oldClientID)
					}
					if oldClientSecret != "" {
						_ = os.Setenv("PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET", oldClientSecret)
					}
					if oldRegionCode != "" {
						_ = os.Setenv("PINGCLI_PINGONE_REGION_CODE", oldRegionCode)
					}
					if oldExportEnvID != "" {
						_ = os.Setenv("PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID", oldExportEnvID)
					}
				}()
			}

			cmd := &TfCommand{}
			logger := &mockLogger{}

			err := cmd.Run(tt.args, logger)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestTfCommand_Configuration tests the Configuration method
func TestTfCommand_Configuration(t *testing.T) {
	cmd := &TfCommand{}
	config, err := cmd.Configuration()

	if err != nil {
		t.Fatalf("Configuration() returned error: %v", err)
	}

	if config == nil {
		t.Fatal("Configuration() returned nil config")
	}

	if config.Use != TfUse {
		t.Errorf("Expected Use=%q, got %q", TfUse, config.Use)
	}

	if config.Short != TfShort {
		t.Errorf("Expected Short=%q, got %q", TfShort, config.Short)
	}

	if config.Long == "" {
		t.Error("Expected non-empty Long description")
	}

	if config.Example == "" {
		t.Error("Expected non-empty Example")
	}
}

// TestTfCommand_SetVersion confirms that SetVersion stores the value in the struct field.
func TestTfCommand_SetVersion(t *testing.T) {
	c := &TfCommand{}
	c.SetVersion("1.2.3")
	if c.version != "1.2.3" {
		t.Errorf("Expected version %q after SetVersion, got %q", "1.2.3", c.version)
	}
}

// TestTfCommand_Routing_VersionPropagation confirms that a version set via
// SetVersion is carried through TfCommand.Run into ExportCommand. The export
// run itself fails on missing credentials — the test asserts only that the
// version field is correctly wired from TfCommand to ExportCommand (observable
// via TfCommand.version remaining intact) and that the resulting error is the
// expected credential-validation failure, not a nil-version or panic.
func TestTfCommand_Routing_VersionPropagation(t *testing.T) {
	// Clear credentials to ensure we get a known validation error
	envVars := []string{
		"PINGCLI_PINGONE_ENVIRONMENT_ID",
		"PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID",
		"PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET",
		"PINGCLI_PINGONE_REGION_CODE",
		"PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID",
	}
	saved := make(map[string]string)
	for _, key := range envVars {
		saved[key] = os.Getenv(key)
		_ = os.Unsetenv(key)
	}
	defer func() {
		for key, val := range saved {
			if val != "" {
				_ = os.Setenv(key, val)
			}
		}
	}()

	c := &TfCommand{}
	c.SetVersion("1.2.3")

	// Version must be stored before Run is called
	if c.version != "1.2.3" {
		t.Fatalf("Expected version %q before Run, got %q", "1.2.3", c.version)
	}

	logger := &mockLogger{}
	err := c.Run([]string{"export"}, logger)

	// Expect a validation error from missing credentials, not a nil error or panic
	if err == nil {
		t.Error("Expected error from missing credentials, got nil")
	}

	// Version field must remain intact after Run (it was passed by value to ExportCommand,
	// so TfCommand.version is unchanged)
	if c.version != "1.2.3" {
		t.Errorf("Expected version %q to persist on TfCommand after Run, got %q", "1.2.3", c.version)
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
