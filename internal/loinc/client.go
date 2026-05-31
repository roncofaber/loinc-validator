package loinc

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultBaseURL = "https://clinicaltables.nlm.nih.gov/api/loinc_items/v3/search"

type LOINCResult struct {
	Code      string
	Name      string
	Version   string
	Valid      bool
	CheckedAt time.Time
	Error     string
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func NewDefaultClient() *Client {
	return NewClient(defaultBaseURL)
}

func (c *Client) Validate(code string) (LOINCResult, error) {
	result := LOINCResult{
		Code:      code,
		CheckedAt: time.Now(),
	}

	params := url.Values{}
	params.Set("terms", code)
	params.Set("sf", "LOINC_NUM")
	params.Set("df", "LOINC_NUM,LONG_COMMON_NAME,VersionLastChanged")
	params.Set("maxList", "5")

	reqURL := c.baseURL + "?" + params.Encode()

	resp, err := c.httpClient.Get(reqURL)
	if err != nil {
		return result, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, fmt.Errorf("reading response: %w", err)
	}

	// NIH response format: [total, code_list, extra, [[field1, field2, field3], ...]]
	var raw []json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return result, fmt.Errorf("parsing response: %w", err)
	}
	if len(raw) < 4 {
		return result, fmt.Errorf("unexpected response structure")
	}

	var displayFields [][]string
	if err := json.Unmarshal(raw[3], &displayFields); err != nil {
		return result, fmt.Errorf("parsing display fields: %w", err)
	}

	for _, fields := range displayFields {
		if len(fields) >= 3 && strings.EqualFold(fields[0], code) {
			result.Valid = true
			result.Code = fields[0]
			result.Name = fields[1]
			result.Version = fields[2]
			return result, nil
		}
	}

	return result, nil
}
