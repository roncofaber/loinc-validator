package icd10_test

import (
	"testing"

	"github.com/roncofaber/loinc-validator/internal/icd10"
)

func TestICD10CodecMeta(t *testing.T) {
	c := icd10.NewCodec()
	if c.Name() != "ICD-10-CM" {
		t.Errorf("unexpected Name: %s", c.Name())
	}
	if c.SystemID() != "icd10" {
		t.Errorf("unexpected SystemID: %s", c.SystemID())
	}
	if c.Version() == "" {
		t.Error("expected non-empty Version")
	}
}

func TestICD10CodecValidateFormat(t *testing.T) {
	c := icd10.NewCodec()
	if err := c.ValidateFormat("E11.9"); err != nil {
		t.Errorf("expected nil for valid code, got: %v", err)
	}
	if err := c.ValidateFormat("U07.1"); err == nil {
		t.Error("expected error for U prefix")
	}
}

func TestICD10CodecParse(t *testing.T) {
	c := icd10.NewCodec()
	result := c.Parse([]string{"E11.9", "Type 2 diabetes mellitus without complications"})
	if !result.Valid {
		t.Error("expected valid=true")
	}
	if result.Code != "E11.9" {
		t.Errorf("unexpected Code: %s", result.Code)
	}
	if result.Name != "Type 2 diabetes mellitus without complications" {
		t.Errorf("unexpected Name: %s", result.Name)
	}
	if result.Deprecated {
		t.Error("expected deprecated=false for ICD-10-CM")
	}
	if result.CheckedAt.IsZero() {
		t.Error("CheckedAt should not be zero")
	}
}

func TestICD10CodecSimilarCandidatesNil(t *testing.T) {
	c := icd10.NewCodec()
	candidates := c.SimilarCandidates("E11.9")
	if candidates != nil {
		t.Errorf("expected nil candidates for ICD-10-CM, got: %v", candidates)
	}
}
