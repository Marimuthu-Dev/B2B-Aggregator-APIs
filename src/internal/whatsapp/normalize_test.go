package whatsapp

import (
	"testing"
)

func TestNormalizePhoneNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"10-digit mobile", "9004535221", "919004535221"},
		{"Already country code", "919004535221", "919004535221"},
		{"Plus 91", "+919004535221", "919004535221"},
		{"Plus 91 with spaces", "+91 90045 35221", "919004535221"},
		{"Leading zero", "09004535221", "919004535221"},
		{"Invalid empty", "", ""},
		{"Invalid letters", "abc9004535221", "919004535221"},
		{"International number (non-Indian)", "12025550123", "12025550123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizePhoneNumber(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizePhoneNumber(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
