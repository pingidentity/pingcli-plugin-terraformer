package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsTruthy(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"1", true},
		{"true", true},
		{"TRUE", true},
		{"yes", true},
		{"YES", true},
		{" true ", true},
		{"0", false},
		{"false", false},
		{"", false},
		{"no", false},
		{"random", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, isTruthy(tt.input))
		})
	}
}
