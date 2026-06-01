package coding

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// ExactMatch finds the first row whose first field matches code (case-insensitive).
func ExactMatch(rows [][]string, code string) ([]string, int) {
	for i, row := range rows {
		if len(row) >= 1 && strings.EqualFold(row[0], code) {
			return row, i
		}
	}
	return nil, -1
}

// Search queries any NIH Clinical Tables API and returns display-field rows.
// All NIH Clinical Tables endpoints share the same response format:
// [total, codes, extra, [[field1, field2, ...], ...]].
func Search(client *http.Client, baseURL, sf, df, term string, maxList int) ([][]string, error) {
	params := url.Values{}
	params.Set("terms", term)
	params.Set("sf", sf)
	params.Set("df", df)
	params.Set("maxList", fmt.Sprintf("%d", maxList))

	resp, err := client.Get(baseURL + "?" + params.Encode())
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
