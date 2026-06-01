package icd10

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// search queries the ICD-10-CM NIH Clinical Tables API and returns display-field rows.
// The ICD-10-CM API exposes only code and name — no ef= extra fields are available.
func search(client *http.Client, baseURL, sf, df, term string, maxList int) ([][]string, error) {
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
