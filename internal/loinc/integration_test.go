package loinc_test

import (
	"os"
	"testing"

	"github.com/roncofaber/loinc-validator/internal/loinc"
)

func TestIntegrationValidCode(t *testing.T) {
	if os.Getenv("LOINC_INTEGRATION") == "" {
		t.Skip("skipping integration test; set LOINC_INTEGRATION=1 to run")
	}

	client := loinc.NewDefaultClient()
	result, err := client.Validate("2345-7")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Errorf("expected 2345-7 to be valid, got invalid")
	}
	if result.Name == "" {
		t.Error("expected non-empty name for valid code")
	}
}

func TestIntegrationInvalidCode(t *testing.T) {
	if os.Getenv("LOINC_INTEGRATION") == "" {
		t.Skip("skipping integration test; set LOINC_INTEGRATION=1 to run")
	}

	client := loinc.NewDefaultClient()
	result, err := client.Validate("99999-9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Error("expected 99999-9 to be invalid")
	}
}
