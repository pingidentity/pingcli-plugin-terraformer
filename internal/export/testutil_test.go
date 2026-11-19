// Copyright © 2025 Ping Identity Corporation

package export

import ()

// mockLogger implements the package's Logger interface for testing.
type mockLogger struct {
	messages []string
	warnings []string
}

func (m *mockLogger) Message(msg string, metadata map[string]string) error {
	m.messages = append(m.messages, msg)
	return nil
}

func (m *mockLogger) Warn(msg string, metadata map[string]string) error {
	m.warnings = append(m.warnings, msg)
	return nil
}
