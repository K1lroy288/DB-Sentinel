package topology

import (
	"testing"
)

func TestCleanVersion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "standard verbose postgres version string",
			input:    "PostgreSQL 16.1 on x86_64-pc-linux-gnu, compiled by gcc (Debian 12.2.0-14) 12.2.0, 64-bit",
			expected: "PostgreSQL 16.1",
		},
		{
			name:     "short version string without on",
			input:    "PostgreSQL 17.0",
			expected: "PostgreSQL 17.0",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanVersion(tt.input)
			if got != tt.expected {
				t.Errorf("cleanVersion(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
