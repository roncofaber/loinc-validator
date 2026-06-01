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
		{"1234567-4", false}, // 7-digit codes: regex doesn't cap length, check digit is 4

		// LP Parts and LG Groups — clear rejection
		{"LP14635-4", true},
		{"LG100-4", true},
		{"lp14635-4", true}, // case-insensitive

		// Empty / whitespace
		{"", true},
		{"   ", true},

		// Wrong structure
		{"abc", true},
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

func TestValidateFormatLPMessage(t *testing.T) {
	err := loinc.ValidateFormat("LP14635-4")
	if err == nil {
		t.Fatal("expected error for LP Part identifier")
	}
	if !strings.Contains(err.Error(), "Part identifier") {
		t.Errorf("expected message to mention Part identifier, got: %s", err.Error())
	}
}

func TestValidateFormatLGMessage(t *testing.T) {
	err := loinc.ValidateFormat("LG100-4")
	if err == nil {
		t.Fatal("expected error for LG Group identifier")
	}
	if !strings.Contains(err.Error(), "Group identifier") {
		t.Errorf("expected message to mention Group identifier, got: %s", err.Error())
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
	if !strings.Contains(msg, "expected 7") {
		t.Errorf("expected error to mention 'expected 7', got: %s", msg)
	}
}
