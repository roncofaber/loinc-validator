// This file contains the LOINC-specific NIH API client. It exists separately
// from internal/coding/client.go because the LOINC API uniquely supports the
// ef= (extra fields) parameter, returning structured metadata such as units of
// measure, data type, and related synonym terms. All other NIH Clinical Tables
// APIs return only code and name, and are handled by the shared client.
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
	Code         string
	Name         string
	ShortName    string
	Component    string
	RelatedNames string
	DataType     string
	Units        []string
	Valid         bool
	Deprecated    bool
	CheckedAt    time.Time
	Error        string
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

// apiUnit matches the JSON structure returned by the "units" ef field.
type apiUnit struct {
	Unit string `json:"unit"`
}

// apiExtra matches the ef (extra fields) hash in the API response.
type apiExtra struct {
	RelatedNames []string    `json:"RELATEDNAMES2"`
	DataType     []string    `json:"datatype"`
	Units        [][]apiUnit `json:"units"`
}

func (c *Client) Validate(code string) (LOINCResult, error) {
	result := LOINCResult{
		Code:      code,
		CheckedAt: time.Now(),
	}

	params := url.Values{}
	params.Set("terms", code)
	params.Set("sf", "LOINC_NUM")
	params.Set("df", "LOINC_NUM,LONG_COMMON_NAME,SHORTNAME,COMPONENT")
	params.Set("ef", "RELATEDNAMES2,datatype,units")
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

	// NIH response format: [total, code_list, extra_hash, [[df_field1, df_field2, ...], ...]]
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

	// Find the entry whose LOINC_NUM matches exactly (case-insensitive).
	matchIdx := -1
	for i, fields := range displayFields {
		if len(fields) >= 2 && strings.EqualFold(fields[0], code) {
			matchIdx = i
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
			break
		}
	}

	if matchIdx == -1 || raw[2] == nil {
		return result, nil
	}

	// Parse extra fields (raw[2]) — only if we found a match.
	var extra apiExtra
	if err := json.Unmarshal(raw[2], &extra); err == nil {
		if matchIdx < len(extra.RelatedNames) {
			result.RelatedNames = extra.RelatedNames[matchIdx]
		}
		if matchIdx < len(extra.DataType) {
			result.DataType = extra.DataType[matchIdx]
		}
		if matchIdx < len(extra.Units) {
			for _, u := range extra.Units[matchIdx] {
				result.Units = append(result.Units, u.Unit)
			}
		}
	}

	return result, nil
}
