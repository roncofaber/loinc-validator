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
	// U07.1 (COVID-19) is a valid billable ICD-10-CM code
	if err := c.ValidateFormat("U07.1"); err != nil {
		t.Errorf("expected nil for U07.1 (COVID-19), got: %v", err)
	}
	if err := c.ValidateFormat("123"); err == nil {
		t.Error("expected error for malformed code")
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

func TestICD10CodecSimilarCandidates(t *testing.T) {
	c := icd10.NewCodec()

	// 7-char code → should produce progressively shorter candidates
	candidates := c.SimilarCandidates("S00.00XA")
	if len(candidates) == 0 {
		t.Error("expected candidates for 7-char code")
	}
	// Should include S00.00X, S00.00, S00.0, S00
	for _, want := range []string{"S00.00X", "S00.00", "S00.0", "S00"} {
		found := false
		for _, got := range candidates {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected candidate %q in %v", want, candidates)
		}
	}

	// 3-char code → already at minimum, no candidates
	candidates3 := c.SimilarCandidates("I10")
	if len(candidates3) != 0 {
		t.Errorf("expected no candidates for 3-char code, got: %v", candidates3)
	}

	// 5-char code with decimal → "E11.9" → ["E11"]
	candidates5 := c.SimilarCandidates("E11.9")
	if len(candidates5) == 0 {
		t.Error("expected candidates for E11.9")
	}
	found := false
	for _, got := range candidates5 {
		if got == "E11" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'E11' in candidates for E11.9, got: %v", candidates5)
	}
}
