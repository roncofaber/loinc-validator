package coding

import "strings"

// ExactMatch finds the first row whose first field matches code (case-insensitive).
func ExactMatch(rows [][]string, code string) ([]string, int) {
	for i, row := range rows {
		if len(row) >= 1 && strings.EqualFold(row[0], code) {
			return row, i
		}
	}
	return nil, -1
}
