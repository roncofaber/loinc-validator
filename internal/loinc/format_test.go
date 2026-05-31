package loinc_test

import (
	"strings"
	"testing"

	"github.com/roncofaber/loinc-validator/internal/loinc"
)

func TestValidateFormat(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		// Valid codes (correct check digit)
		{"2345-7", false},
		{"8867-4", false},
		{"12345-5", false},
		{"100000-9", false}, // 6-digit codes exist in LOINC 2.74+

		// Empty / whitespace
		{"", true},
		{"   ", true},

		// Wrong structure
		{"abc", true},
		{"1234567-7", true}, // 7 digits — no such codes in LOINC 2.82
		{"2345", true},      // missing check digit
		{"2345-", true},     // missing check digit value
		{"2345-77", true},   // check digit > 1 digit
		{"-7", true},        // no leading digits
		{"23 45-7", true},   // space inside

		// Wrong check digit (correct format, wrong final digit)
		{"2345-4", true}, // should be 7
		{"8867-9", true}, // should be 4
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

func TestValidateFormatCheckDigitMessage(t *testing.T) {
	err := loinc.ValidateFormat("2345-4")
	if err == nil {
		t.Fatal("expected error for wrong check digit")
	}
	msg := err.Error()
	if !strings.Contains(msg, "expected 7") {
		t.Errorf("expected error to mention 'expected 7', got: %s", msg)
	}
	if !strings.Contains(msg, "2345-7") {
		t.Errorf("expected error to suggest '2345-7', got: %s", msg)
	}
}
