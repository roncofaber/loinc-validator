package icd10

import (
	"fmt"
	"regexp"
	"strings"
)

// icd10Pattern follows the official ICD-10-CM spec:
//   - Position 1: alpha, U excluded (reserved for ICD-11 WHO extensions)
//   - Position 2: numeric
//   - Position 3: alpha or numeric
//   - Positions 4-7: optional decimal followed by 1-4 alphanumeric characters
var icd10Pattern = regexp.MustCompile(`^[A-TV-Z]\d[A-Z0-9](\.[A-Z0-9]{1,4})?$`)

// ValidateFormat checks ICD-10-CM code structure. It is case-insensitive.
// It validates shape only — existence is confirmed via the NIH API.
func ValidateFormat(code string) error {
	code = strings.TrimSpace(strings.ToUpper(code))
	if code == "" {
		return fmt.Errorf("ICD-10-CM code cannot be empty")
	}
	if strings.HasPrefix(code, "U") {
		return fmt.Errorf("invalid ICD-10-CM code %q: the letter U is reserved for ICD-11 WHO extensions and is not used in ICD-10-CM", code)
	}
	if !icd10Pattern.MatchString(code) {
		return fmt.Errorf("invalid ICD-10-CM format %q: expected a letter (not U), a digit, an alphanumeric character, then optionally a decimal and up to 4 alphanumeric characters (e.g. E11.9, I10, S00.00XA)", code)
	}
	return nil
}
