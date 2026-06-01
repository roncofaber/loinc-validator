package icd10_test

import (
	"os"
	"testing"

	"github.com/roncofaber/loinc-validator/internal/coding"
	"github.com/roncofaber/loinc-validator/internal/icd10"
)

func TestIntegrationValidCode(t *testing.T) {
	if os.Getenv("ICD10_INTEGRATION") == "" {
		t.Skip("skipping integration test; set ICD10_INTEGRATION=1 to run")
	}

	codec := icd10.NewCodec()
	client := coding.NewHTTPClient()

	rows, err := client.Validate(codec, "E11.9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	row, _ := coding.ExactMatch(rows, "E11.9")
	if row == nil {
		t.Fatal("expected E11.9 to be found")
	}
	result := codec.Parse(row)
	if !result.Valid {
		t.Error("expected valid=true")
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
	client := coding.NewHTTPClient()

	rows, err := client.Validate(codec, "Z99.99")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	row, _ := coding.ExactMatch(rows, "Z99.99")
	if row != nil {
		t.Error("expected Z99.99 to be not found")
	}
}

func TestIntegrationNonBillableHeader(t *testing.T) {
	if os.Getenv("ICD10_INTEGRATION") == "" {
		t.Skip("skipping integration test; set ICD10_INTEGRATION=1 to run")
	}

	// A01 is a valid ICD-10-CM category but non-billable — API returns not found on exact match
	codec := icd10.NewCodec()
	client := coding.NewHTTPClient()

	rows, err := client.Validate(codec, "A01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	row, _ := coding.ExactMatch(rows, "A01")
	if row != nil {
		t.Error("expected A01 (non-billable header) to return not found on exact match")
	}
}
