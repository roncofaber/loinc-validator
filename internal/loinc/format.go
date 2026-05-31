package loinc

import (
	"fmt"
	"regexp"
	"strings"
)

var loincPattern = regexp.MustCompile(`^\d{1,6}-\d$`)

// ValidateFormat checks that code has the correct LOINC format and a valid
// Mod-10 check digit. Returns a specific error message when the check digit
// is wrong so the user knows what the correct code should be.
func ValidateFormat(code string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return fmt.Errorf("LOINC code cannot be empty")
	}
	if !loincPattern.MatchString(code) {
		return fmt.Errorf("invalid LOINC format %q: expected 1–6 digits, a dash, then 1 digit (e.g. 2345-7)", code)
	}

	parts := strings.SplitN(code, "-", 2)
	submitted := int(parts[1][0] - '0')
	expected := checkDigit(parts[0])

	if submitted != expected {
		return fmt.Errorf("invalid check digit for %q: expected %d, got %d — did you mean %s-%d?",
			code, expected, submitted, parts[0], expected)
	}
	return nil
}

// checkDigit computes the LOINC Mod-10 check digit for the numeric prefix.
// Each digit is processed right-to-left; even-indexed positions (0-based from
// the right) are doubled, with values > 9 reduced by 9.
func checkDigit(prefix string) int {
	total := 0
	for i, ch := range reverse(prefix) {
		n := int(ch - '0')
		if i%2 == 0 {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		total += n
	}
	return (10 - (total % 10)) % 10
}

func reverse(s string) string {
	r := []byte(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}
