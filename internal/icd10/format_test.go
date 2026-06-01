package icd10_test

import (
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

		// U codes are valid — U07.1 (COVID-19), U07.0 (vaping), U09.9 (long COVID)
		// are real billable ICD-10-CM codes; the API is the authoritative existence check
		{"U07.1", false},
		{"U09.9", false},

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

func TestValidateFormatUCodesValid(t *testing.T) {
	// U codes are valid in ICD-10-CM — COVID-era codes use the U range
	for _, code := range []string{"U07.1", "U07.0", "U09.9"} {
		if err := icd10.ValidateFormat(code); err != nil {
			t.Errorf("expected U code %q to pass format validation, got: %v", code, err)
		}
	}
}
