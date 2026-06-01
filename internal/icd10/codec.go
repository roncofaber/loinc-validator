package icd10

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	return &ICD10Codec{httpClient: &http.Client{Timeout: 10 * time.Second}}
}

func (c *ICD10Codec) Name() string     { return "ICD-10-CM" }
func (c *ICD10Codec) SystemID() string { return "icd10" }
func (c *ICD10Codec) Version() string  { return icd10Version }

func (c *ICD10Codec) ValidateFormat(code string) error {
	return ValidateFormat(code)
}

// Validate queries the ICD-10-CM API for the given code and returns the result.
func (c *ICD10Codec) Validate(code string) (coding.Result, error) {
	rows, err := c.search(icd10SearchURL, icd10SearchFields, icd10DisplayFields, code, 5)
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
	return c.search(icd10SearchURL, icd10SearchFields, icd10DisplayFields, query, maxList)
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

// SimilarCandidates returns prefix-truncated variants of an ICD-10-CM code
// to surface near-matches when an exact code is not found.
func (c *ICD10Codec) SimilarCandidates(code string) []string {
	var candidates []string
	seen := make(map[string]bool)

	add := func(s string) {
		if s != code && !seen[s] && len(s) >= 3 {
			seen[s] = true
			candidates = append(candidates, s)
		}
	}

	current := code
	for len(current) > 3 {
		current = current[:len(current)-1]
		if len(current) > 0 && current[len(current)-1] == '.' {
			current = current[:len(current)-1]
		}
		add(current)
	}
	return candidates
}

func (c *ICD10Codec) search(baseURL, sf, df, term string, maxList int) ([][]string, error) {
	params := url.Values{}
	params.Set("terms", term)
	params.Set("sf", sf)
	params.Set("df", df)
	params.Set("maxList", fmt.Sprintf("%d", maxList))

	resp, err := c.httpClient.Get(baseURL + "?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var raw []json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	if len(raw) < 4 {
		return nil, fmt.Errorf("unexpected response structure")
	}

	var rows [][]string
	if err := json.Unmarshal(raw[3], &rows); err != nil {
		return nil, fmt.Errorf("parsing display fields: %w", err)
	}
	return rows, nil
}

// ValidateForSimilar validates a candidate code against the API — used by
// the similar handler when checking transposition/prefix candidates.
func (c *ICD10Codec) ValidateForSimilar(code string) (coding.Result, error) {
	return c.Validate(code)
}
