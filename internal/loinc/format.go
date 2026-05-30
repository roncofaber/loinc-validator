package loinc

import (
	"fmt"
	"regexp"
	"strings"
)

var loincPattern = regexp.MustCompile(`^\d{1,5}-\d$`)

func ValidateFormat(code string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return fmt.Errorf("LOINC code cannot be empty")
	}
	if !loincPattern.MatchString(code) {
		return fmt.Errorf("invalid LOINC format %q: expected 1–5 digits, a dash, then 1 digit (e.g. 2345-7)", code)
	}
	return nil
}
