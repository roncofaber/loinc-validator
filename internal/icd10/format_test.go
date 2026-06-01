package icd10_test

import (
	"strings"
	"testing"

	"github.com/roncofaber/loinc-validator/internal/icd10"
)

func TestValidateFormat(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		// Valid codes
		{"E11.9", false},
		{"I10", false},       // 3-char code, no decimal
		{"Z99.89", false},
		{"S00.00XA", false},  // X placeholder + 7th char
		{"e11.9", false},     // case-insensitive
		{"A00.0", false},
		{"M96.830", false},   // 7 chars
		{"A01", false},       // 3-char header (valid format, non-billable via API)

		// Invalid: empty
		{"", true},
		{"   ", true},

		// Invalid: U prefix (reserved for ICD-11)
		{"U07.1", true},
		{"U00.0", true},

		// Invalid: wrong structure
		{"123", true},         // digit in position 1
		{"A1", true},          // too short
		{"AB1.0", true},       // position 2 must be digit
		{"A00.00000", true},   // too many chars after decimal (max 4)
		{"A00.", true},        // decimal with no following chars
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := icd10.ValidateFormat(tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for input %q, got nil", tt.input)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for input %q: %v", tt.input, err)
			}
		})
	}
}

func TestValidateFormatUMessage(t *testing.T) {
	err := icd10.ValidateFormat("U07.1")
	if err == nil {
		t.Fatal("expected error for U prefix")
	}
	if !strings.Contains(err.Error(), "U") {
		t.Errorf("expected error to mention U prefix, got: %s", err.Error())
	}
}
