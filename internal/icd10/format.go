package icd10

import (
	"fmt"
	"regexp"
	"strings"
)

// icd10Pattern follows the official ICD-10-CM spec:
//   - Position 1: any alpha letter (A-Z, including U — U07.x codes are valid billable codes
//     e.g. U07.1 COVID-19, U07.0 vaping disorder, U09.9 post-COVID condition)
//   - Position 2: numeric
//   - Position 3: alpha or numeric
//   - Positions 4-7: optional decimal followed by 1-4 alphanumeric characters
//
// The U range was historically reserved in WHO ICD-10 for provisional new diseases,
// but ICD-10-CM actively uses it — excluding U would reject valid COVID-era codes.
// The API is the authoritative source for whether a code actually exists.
var icd10Pattern = regexp.MustCompile(`^[A-Z]\d[A-Z0-9](\.[A-Z0-9]{1,4})?$`)

// ValidateFormat checks ICD-10-CM code structure. It is case-insensitive.
// It validates shape only — existence is confirmed via the NIH API.
func ValidateFormat(code string) error {
	code = strings.TrimSpace(strings.ToUpper(code))
	if code == "" {
		return fmt.Errorf("ICD-10-CM code cannot be empty")
	}
	if !icd10Pattern.MatchString(code) {
		return fmt.Errorf("invalid ICD-10-CM format %q: expected a letter, a digit, an alphanumeric character, then optionally a decimal and up to 4 alphanumeric characters (e.g. E11.9, I10, S00.00XA, U07.1)", code)
	}
	return nil
}
