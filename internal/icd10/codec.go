package icd10

import (
	"fmt"
	"net/http"
	"strings"
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
// The ICD-10-CM API exposes only two fields (code, name) — there are no
// additional ef= fields available beyond these.
type ICD10Codec struct {
	httpClient *http.Client
}

func NewCodec() *ICD10Codec {
	return &ICD10Codec{httpClient: coding.NewHTTPClient()}
}

func (c *ICD10Codec) Name() string     { return "ICD-10-CM" }
func (c *ICD10Codec) SystemID() string { return "icd10" }
func (c *ICD10Codec) Version() string  { return icd10Version }

func (c *ICD10Codec) ValidateFormat(code string) error {
	return ValidateFormat(code)
}

// MissingDecimalBase detects codes like "E800" where the user likely forgot
// the decimal point (should be "E80.0"). Returns the 3-char base (e.g. "E80")
// if the pattern matches, or empty string otherwise.
func MissingDecimalBase(code string) string {
	upper := strings.ToUpper(strings.TrimSpace(code))
	// Pattern: letter + 2 digits + 1 or more chars, no decimal
	if len(upper) >= 4 && !strings.Contains(upper, ".") {
		if upper[0] >= 'A' && upper[0] <= 'Z' && upper[1] >= '0' && upper[1] <= '9' && upper[2] >= '0' && upper[2] <= '9' {
			return upper[:3]
		}
	}
	return ""
}

// Validate queries the ICD-10-CM API for the given code and returns the result.
func (c *ICD10Codec) Validate(code string) (coding.Result, error) {
	rows, err := coding.Search(c.httpClient, icd10SearchURL, icd10SearchFields, icd10DisplayFields, code, 5)
	if err != nil {
		return coding.Result{Code: code, CheckedAt: time.Now()}, err
	}
	row, _ := coding.ExactMatch(rows, code)
	if row == nil {
		return coding.Result{Code: code, CheckedAt: time.Now()}, nil
	}
	return c.Parse(row), nil
}

// Suggest queries the ICD-10-CM API for autocomplete candidates.
func (c *ICD10Codec) Suggest(query string, maxList int) ([][]string, error) {
	return coding.Search(c.httpClient, icd10SearchURL, icd10SearchFields, icd10DisplayFields, query, maxList)
}

func (c *ICD10Codec) Parse(fields []string) coding.Result {
	result := coding.Result{CheckedAt: time.Now()}
	if len(fields) < 2 {
		return result
	}
	result.Valid = true
	result.Code = fields[0]
	result.Name = fields[1]
	return result
}

// SimilarCandidates returns candidates to check when an ICD-10-CM code is not found.
// It uses two strategies:
//  1. Truncation — progressively shorter prefixes (e.g. E11.99 → E11.9 → E11)
//  2. Expansion — decimal children for 3-char category codes (e.g. E80 → E80.0…E80.9)
func (c *ICD10Codec) SimilarCandidates(code string) []string {
	var candidates []string
	seen := make(map[string]bool)

	add := func(s string) {
		if s != code && !seen[s] && len(s) >= 3 {
			seen[s] = true
			candidates = append(candidates, s)
		}
	}

	// Strategy 1: truncation
	current := code
	for len(current) > 3 {
		current = current[:len(current)-1]
		if len(current) > 0 && current[len(current)-1] == '.' {
			current = current[:len(current)-1]
		}
		add(current)
	}

	// Strategy 2: expansion — if code is a 3-char category (e.g. E80),
	// try E80.0 through E80.9 as likely billable children.
	base := code
	if idx := strings.Index(code, "."); idx != -1 {
		base = code[:idx]
	}
	if len(base) == 3 {
		for d := 0; d <= 9; d++ {
			add(fmt.Sprintf("%s.%d", base, d))
		}
	}

	return candidates
}
