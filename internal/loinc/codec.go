package loinc

import (
	"net/http"
	"strings"
	"time"

	"github.com/roncofaber/loinc-validator/internal/coding"
)

const (
	loincSearchURL        = "https://clinicaltables.nlm.nih.gov/api/loinc_items/v3/search"
	loincSearchFields     = "LOINC_NUM"
	loincSuggestFields    = "LOINC_NUM,LONG_COMMON_NAME"
	loincDisplayFields    = "LOINC_NUM,LONG_COMMON_NAME,SHORTNAME,COMPONENT"
	loincVersion          = "2.82"
)

// LOINCCodec implements coding.Codec for LOINC observation codes.
// It uses the existing loinc.Client for single-code validation (which fetches
// extra fields via ef=) and a lightweight shared search for autocomplete.
type LOINCCodec struct {
	httpClient *http.Client
}

func NewCodec() *LOINCCodec {
	return &LOINCCodec{httpClient: coding.NewHTTPClient()}
}

func (c *LOINCCodec) Name() string     { return "LOINC" }
func (c *LOINCCodec) SystemID() string { return "loinc" }
func (c *LOINCCodec) Version() string  { return loincVersion }

func (c *LOINCCodec) ValidateFormat(code string) error {
	return ValidateFormat(code)
}

// Validate calls the LOINC-specific client which fetches extra fields
// (units, datatype, related names) via ef= in a single API call.
func (c *LOINCCodec) Validate(code string) (coding.Result, error) {
	lc := NewDefaultClient()
	res, err := lc.Validate(code)
	if err != nil {
		return coding.Result{Code: code, CheckedAt: time.Now()}, err
	}
	return coding.Result{
		Code:         res.Code,
		Name:         res.Name,
		ShortName:    res.ShortName,
		Component:    res.Component,
		RelatedNames: res.RelatedNames,
		DataType:     res.DataType,
		Units:        res.Units,
		Valid:         res.Valid,
		Deprecated:    res.Deprecated,
		CheckedAt:    res.CheckedAt,
		Error:        res.Error,
	}, nil
}

// Suggest queries the LOINC API for autocomplete candidates by code or name.
func (c *LOINCCodec) Suggest(query string, maxList int) ([][]string, error) {
	return coding.Search(c.httpClient, loincSearchURL, loincSuggestFields, loincDisplayFields, query, maxList)
}

func (c *LOINCCodec) Parse(fields []string) coding.Result {
	result := coding.Result{CheckedAt: time.Now()}
	if len(fields) < 2 {
		return result
	}
	result.Valid = true
	result.Code = fields[0]
	result.Name = fields[1]
	result.Deprecated = strings.HasPrefix(fields[1], "Deprecated ")
	if len(fields) >= 3 {
		result.ShortName = fields[2]
	}
	if len(fields) >= 4 {
		result.Component = fields[3]
	}
	return result
}

func (c *LOINCCodec) SimilarCandidates(code string) []string {
	return transpositionCandidates(code)
}

// transpositionCandidates returns codes formed by swapping adjacent digits,
// with recomputed check digits.
func transpositionCandidates(code string) []string {
	parts := strings.SplitN(code, "-", 2)
	if len(parts) != 2 {
		return nil
	}
	prefix := parts[0]
	seen := make(map[string]bool)
	var candidates []string
	for i := 0; i < len(prefix)-1; i++ {
		b := []byte(prefix)
		b[i], b[i+1] = b[i+1], b[i]
		newPrefix := string(b)
		if newPrefix == prefix {
			continue
		}
		chk := CheckDigit(newPrefix)
		candidate := newPrefix + "-" + string(rune('0'+chk))
		if !seen[candidate] {
			seen[candidate] = true
			candidates = append(candidates, candidate)
		}
	}
	return candidates
}

