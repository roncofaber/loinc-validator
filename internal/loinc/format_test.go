package loinc_test

import (
	"testing"

	"github.com/roncofaber/loinc-validator/internal/loinc"
)

func TestValidateFormat(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"2345-7", false},
		{"1-1", false},
		{"12345-6", false},
		{"", true},
		{"   ", true},
		{"abc", true},
		{"123456-7", true}, // too many leading digits
		{"2345", true},     // missing check digit
		{"2345-", true},    // missing check digit value
		{"2345-77", true},  // check digit > 1 digit
		{"-7", true},       // no leading digits
		{"23 45-7", true},  // space inside
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := loinc.ValidateFormat(tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for input %q, got nil", tt.input)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for input %q: %v", tt.input, err)
			}
		})
	}
}
