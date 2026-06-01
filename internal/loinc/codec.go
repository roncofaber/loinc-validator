package loinc

import (
	"strings"
	"time"

	"github.com/roncofaber/loinc-validator/internal/coding"
)

const (
	loincSearchURL     = "https://clinicaltables.nlm.nih.gov/api/loinc_items/v3/search"
	loincSearchFields  = "LOINC_NUM"
	loincDisplayFields = "LOINC_NUM,LONG_COMMON_NAME,SHORTNAME,COMPONENT"
	loincVersion       = "2.82"
)

// LOINCCodec implements coding.Codec for LOINC observation codes.
type LOINCCodec struct{}

func NewCodec() *LOINCCodec { return &LOINCCodec{} }

func (c *LOINCCodec) Name() string          { return "LOINC" }
func (c *LOINCCodec) SystemID() string      { return "loinc" }
func (c *LOINCCodec) Version() string       { return loincVersion }
func (c *LOINCCodec) SearchURL() string     { return loincSearchURL }
func (c *LOINCCodec) SearchFields() string  { return loincSearchFields }
func (c *LOINCCodec) DisplayFields() string { return loincDisplayFields }

func (c *LOINCCodec) ValidateFormat(code string) error {
	return ValidateFormat(code)
}

func (c *LOINCCodec) Parse(fields []string) coding.Result {
	result := coding.Result{
		CheckedAt: time.Now(),
	}
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

// ValidateWithExtras calls the existing LOINC client (which fetches ef fields)
// and returns a full result including units, datatype, and related names.
// Used by the validate handler for single-code lookups.
func (c *LOINCCodec) ValidateWithExtras(code string) (coding.Result, error) {
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
