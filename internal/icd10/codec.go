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

// SimilarCandidates returns nil — ICD-10-CM has no check digit mechanism
// for generating transposition candidates.
func (c *ICD10Codec) SimilarCandidates(code string) []string {
	return nil
}
