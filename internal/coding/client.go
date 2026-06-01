package coding

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HTTPClient performs searches against the NIH Clinical Tables API.
// It is codec-agnostic — the codec provides the URL and field config.
type HTTPClient struct {
	httpClient *http.Client
}

func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// Search queries the API and returns display-field rows for the given term.
func (c *HTTPClient) Search(baseURL, searchFields, displayFields, term string, maxList int) ([][]string, error) {
	params := url.Values{}
	params.Set("terms", term)
	params.Set("sf", searchFields)
	params.Set("df", displayFields)
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

	// NIH response: [total, codes, extra, [[field1, field2, ...], ...]]
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

// Validate searches for an exact code match using the codec's configuration.
func (c *HTTPClient) Validate(codec Codec, code string) ([][]string, error) {
	return c.Search(codec.SearchURL(), codec.SearchFields(), codec.DisplayFields(), code, 5)
}

// Suggest returns up to maxList rows for the given query (for autocomplete).
func (c *HTTPClient) Suggest(codec Codec, query string, maxList int) ([][]string, error) {
	return c.Search(codec.SearchURL(), codec.SearchFields(), codec.DisplayFields(), query, maxList)
}

// ExactMatch finds the first row whose first field matches code (case-insensitive).
func ExactMatch(rows [][]string, code string) ([]string, int) {
	for i, row := range rows {
		if len(row) >= 1 && strings.EqualFold(row[0], code) {
			return row, i
		}
	}
	return nil, -1
}
