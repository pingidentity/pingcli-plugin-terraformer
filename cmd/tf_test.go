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

				os.Unsetenv("PINGCLI_PINGONE_ENVIRONMENT_ID")
				os.Unsetenv("PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID")
				os.Unsetenv("PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET")
				os.Unsetenv("PINGCLI_PINGONE_REGION_CODE")
				os.Unsetenv("PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID")

				defer func() {
					if oldWorkerEnvID != "" {
						os.Setenv("PINGCLI_PINGONE_ENVIRONMENT_ID", oldWorkerEnvID)
					}
					if oldClientID != "" {
						os.Setenv("PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID", oldClientID)
					}
					if oldClientSecret != "" {
						os.Setenv("PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET", oldClientSecret)
					}
					if oldRegionCode != "" {
						os.Setenv("PINGCLI_PINGONE_REGION_CODE", oldRegionCode)
					}
					if oldExportEnvID != "" {
						os.Setenv("PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID", oldExportEnvID)
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
