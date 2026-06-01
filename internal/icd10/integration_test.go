package icd10_test

import (
	"os"
	"testing"

	"github.com/roncofaber/loinc-validator/internal/icd10"
)

func TestIntegrationValidCode(t *testing.T) {
	if os.Getenv("ICD10_INTEGRATION") == "" {
		t.Skip("skipping integration test; set ICD10_INTEGRATION=1 to run")
	}

	codec := icd10.NewCodec()
	result, err := codec.Validate("E11.9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Fatal("expected E11.9 to be valid")
	}
	if result.Name == "" {
		t.Error("expected non-empty name")
	}
}

func TestIntegrationInvalidCode(t *testing.T) {
	if os.Getenv("ICD10_INTEGRATION") == "" {
		t.Skip("skipping integration test; set ICD10_INTEGRATION=1 to run")
	}

	codec := icd10.NewCodec()
	result, err := codec.Validate("Z99.99")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Error("expected Z99.99 to be not found")
	}
}

func TestIntegrationNonBillableHeader(t *testing.T) {
	if os.Getenv("ICD10_INTEGRATION") == "" {
		t.Skip("skipping integration test; set ICD10_INTEGRATION=1 to run")
	}

	// A01 is a valid ICD-10-CM category but non-billable — returns not found on exact match
	codec := icd10.NewCodec()
	result, err := codec.Validate("A01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Error("expected A01 (non-billable header) to return not found")
	}
}
