package icd10

import (
	"time"

	"github.com/roncofaber/loinc-validator/internal/coding"
)

const (
	icd10SearchURL     = "https://clinicaltables.nlm.nih.gov/api/icd10cm/v3/search"
	icd10SearchFields  = "code,name"
	icd10DisplayFields = "code,name"
	icd10Version       = "2026"
)

// ICD10Codec implements coding.Codec for ICD-10-CM diagnosis codes.
type ICD10Codec struct{}

func NewCodec() *ICD10Codec { return &ICD10Codec{} }

func (c *ICD10Codec) Name() string          { return "ICD-10-CM" }
func (c *ICD10Codec) SystemID() string      { return "icd10" }
func (c *ICD10Codec) Version() string       { return icd10Version }
func (c *ICD10Codec) SearchURL() string     { return icd10SearchURL }
func (c *ICD10Codec) SearchFields() string  { return icd10SearchFields }
func (c *ICD10Codec) DisplayFields() string { return icd10DisplayFields }

func (c *ICD10Codec) ValidateFormat(code string) error {
	return ValidateFormat(code)
}

func (c *ICD10Codec) Parse(fields []string) coding.Result {
	result := coding.Result{
		CheckedAt: time.Now(),
	}
	if len(fields) < 2 {
		return result
	}
	result.Valid = true
	result.Code = fields[0]
	result.Name = fields[1]
	// ICD-10-CM has no deprecated prefix convention and no check digit.
	return result
}

// SimilarCandidates returns prefix-truncated variants of an ICD-10-CM code
// to surface near-matches when an exact code is not found.
// For example, "E11.99" → ["E11.9", "E11"] (drop last char, drop to category).
// This catches the most common ICD-10-CM mistake: a code that is too specific
// or has a wrong final character.
func (c *ICD10Codec) SimilarCandidates(code string) []string {
	var candidates []string
	seen := make(map[string]bool)

	add := func(s string) {
		if s != code && !seen[s] && len(s) >= 3 {
			seen[s] = true
			candidates = append(candidates, s)
		}
	}

	// Drop one character at a time from the end, down to the 3-char category minimum.
	current := code
	for len(current) > 3 {
		current = current[:len(current)-1]
		// Remove trailing dot if we just exposed it (e.g. "E11." → "E11")
		if len(current) > 0 && current[len(current)-1] == '.' {
			current = current[:len(current)-1]
		}
		add(current)
	}

	return candidates
}
