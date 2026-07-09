// Copyright © 2025 Ping Identity Corporation

package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListOutputsCommand_Configuration(t *testing.T) {
	cmd := &ListOutputsCommand{}
	config, err := cmd.Configuration()
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, ListOutputsUse, config.Use)
	assert.Equal(t, ListOutputsShort, config.Short)
	assert.NotEmpty(t, config.Long)
	assert.NotEmpty(t, config.Example)
}

func TestListOutputsCommand_MissingCredentials(t *testing.T) {
	// Clear credentials so we hit the validation error, not a real API call.
	for _, env := range []string{
		"PINGCLI_PINGONE_ENVIRONMENT_ID",
		"PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID",
		"PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET",
		"PINGCLI_PINGONE_REGION_CODE",
		"PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID",
	} {
		old := os.Getenv(env)
		_ = os.Unsetenv(env)
		defer func(k, v string) {
			if v != "" {
				_ = os.Setenv(k, v)
			}
		}(env, old)
	}

	cmd := &ListOutputsCommand{}
	logger := &mockLogger{}
	err := cmd.Run([]string{}, logger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "worker environment ID is required")
}

func TestListOutputsCommand_MissingClientID(t *testing.T) {
	for _, env := range []string{
		"PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID",
		"PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET",
	} {
		old := os.Getenv(env)
		_ = os.Unsetenv(env)
		defer func(k, v string) {
			if v != "" {
				_ = os.Setenv(k, v)
			}
		}(env, old)
	}

	cmd := &ListOutputsCommand{}
	logger := &mockLogger{}
	err := cmd.Run([]string{"--pingone-worker-environment-id", "env-123"}, logger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "client ID is required")
}

func TestListOutputsCommand_MissingClientSecret(t *testing.T) {
	old := os.Getenv("PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET")
	_ = os.Unsetenv("PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET")
	defer func() {
		if old != "" {
			_ = os.Setenv("PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET", old)
		}
	}()

	cmd := &ListOutputsCommand{}
	logger := &mockLogger{}
	err := cmd.Run([]string{
		"--pingone-worker-environment-id", "env-123",
		"--pingone-worker-client-id", "client-id",
	}, logger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "client secret is required")
}

func TestListOutputsCommand_UnknownFlag(t *testing.T) {
	cmd := &ListOutputsCommand{}
	logger := &mockLogger{}
	err := cmd.Run([]string{"--unknown-flag"}, logger)
	require.Error(t, err)
}

// TestTfCommand_ListOutputsRouting verifies that "list-outputs" is dispatched.
func TestTfCommand_ListOutputsRouting(t *testing.T) {
	// Clear credentials so the command reaches the credential-validation error,
	// confirming the subcommand was routed correctly.
	for _, env := range []string{
		"PINGCLI_PINGONE_ENVIRONMENT_ID",
		"PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID",
		"PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET",
	} {
		old := os.Getenv(env)
		_ = os.Unsetenv(env)
		defer func(k, v string) {
			if v != "" {
				_ = os.Setenv(k, v)
			}
		}(env, old)
	}

	tf := &TfCommand{}
	logger := &mockLogger{}
	err := tf.Run([]string{"list-outputs"}, logger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "worker environment ID is required")
}
