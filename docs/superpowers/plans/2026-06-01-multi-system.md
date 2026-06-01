# Multi-System Medical Code Validator Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor the LOINC-only validator into a multi-system validator with a `Codec` interface, add ICD-10-CM as a second system, and expose both via tabbed UI with system-scoped routes.

**Architecture:** Define a `Codec` interface in `internal/coding/`; migrate LOINC logic to implement it; add ICD-10-CM codec; make all handlers system-agnostic by accepting `Codec`; generate routes and tab UI dynamically from the codec list in `main.go`.

**Tech Stack:** Go stdlib, HTMX, existing CSS/template infrastructure

---

## File Map

```
internal/
  coding/
    codec.go               # NEW: Codec interface + Result + Suggestion types
    client.go              # NEW: shared NIH HTTP fetch + response parse
  loinc/
    codec.go               # NEW: Codec implementation wrapping existing logic
    client.go              # KEEP (unchanged — still used by codec.go internally)
    format.go              # KEEP (unchanged)
    format_test.go         # KEEP
    client_test.go         # KEEP
    integration_test.go    # KEEP
  icd10/
    codec.go               # NEW: ICD-10-CM Codec implementation
    format.go              # NEW: ICD-10-CM format validation
    format_test.go         # NEW: table-driven format tests
    codec_test.go          # NEW: unit tests with mocked HTTP
  handlers/
    validate.go            # MODIFY: loinc.Client → coding.Codec
    batch.go               # MODIFY: loinc.Client → coding.Codec
    suggest.go             # MODIFY: hardcoded URL → codec fields
    similar.go             # MODIFY: loinc.CheckDigit → codec.SimilarCandidates()
    export.go              # KEEP (unchanged)
    templates.go           # KEEP (unchanged)
main.go                    # MODIFY: codec slice + dynamic route registration

templates/
  index.html               # MODIFY: tab bar + two tab panels
  loinc/
    tab.html               # NEW: LOINC form content
  icd10/
    tab.html               # NEW: ICD-10-CM form content
  partials/                # ALL KEPT unchanged (result.html, batch_result.html, etc.)
```

---

## Task 1: Define the `coding.Codec` interface and shared types

**Files:**
- Create: `internal/coding/codec.go`
- Create: `internal/coding/client.go`

- [ ] **Step 1: Create `internal/coding/codec.go`**

```go
package coding

import "time"

// Result is the system-agnostic validation result returned by all codecs.
type Result struct {
	Code         string
	Name         string
	Valid         bool
	Deprecated    bool
	CheckedAt    time.Time
	Error        string
	// Optional — populated only by codecs that support them
	ShortName    string
	Component    string
	RelatedNames string
	DataType     string
	Units        []string
}

// Suggestion is a candidate code for "did you mean?" display.
type Suggestion struct {
	Code string
	Name string
}

// Codec defines the interface every medical coding system must implement.
type Codec interface {
	// Name is the human-readable system name shown in the tab bar.
	Name() string
	// SystemID is the URL-safe identifier used in route prefixes.
	SystemID() string
	// Version is the data version shown in the UI (e.g. "2.82", "2026").
	Version() string
	// SearchURL is the NIH Clinical Tables API base URL for this system.
	SearchURL() string
	// SearchFields is the sf= query parameter (fields to search against).
	SearchFields() string
	// DisplayFields is the df= query parameter (fields to return).
	DisplayFields() string
	// ValidateFormat checks code structure before hitting the API.
	// Returns nil if the code is structurally valid (or if the system has no format constraints).
	ValidateFormat(code string) error
	// Parse maps a display-fields row from the API response into a Result.
	Parse(fields []string) Result
	// SimilarCandidates returns codes to check when a code is not found.
	// Returns nil if the system does not support similarity search.
	SimilarCandidates(code string) []string
}
```

- [ ] **Step 2: Create `internal/coding/client.go`**

```go
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

// HTTPClient performs a search against the NIH Clinical Tables API and
// returns the display-field rows. It is codec-agnostic.
type HTTPClient struct {
	httpClient *http.Client
}

func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// Search queries the API and returns display-field rows for the given term.
// Returns nil rows (not an error) when the term is not found.
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

// Validate searches for an exact code match and returns the first matching row.
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
```

- [ ] **Step 3: Verify it compiles**

```bash
go build ./internal/coding/...
```

Expected: no output.

- [ ] **Step 4: Commit**

```bash
git add internal/coding/
git commit -m "feat: add coding.Codec interface and shared NIH HTTP client"
```

---

## Task 2: Implement the LOINC codec

**Files:**
- Create: `internal/loinc/codec.go`
- Keep unchanged: `internal/loinc/client.go`, `internal/loinc/format.go`

- [ ] **Step 1: Create `internal/loinc/codec.go`**

```go
package loinc

import (
	"strings"
	"time"

	"github.com/roncofaber/loinc-validator/internal/coding"
)

const (
	loincSearchURL    = "https://clinicaltables.nlm.nih.gov/api/loinc_items/v3/search"
	loincSearchFields = "LOINC_NUM"
	loincDisplayFields = "LOINC_NUM,LONG_COMMON_NAME,SHORTNAME,COMPONENT"
	loincVersion      = "2.82"
)

// LOINCCodec implements coding.Codec for LOINC observation codes.
type LOINCCodec struct{}

func NewCodec() *LOINCCodec { return &LOINCCodec{} }

func (c *LOINCCodec) Name() string         { return "LOINC" }
func (c *LOINCCodec) SystemID() string     { return "loinc" }
func (c *LOINCCodec) Version() string      { return loincVersion }
func (c *LOINCCodec) SearchURL() string    { return loincSearchURL }
func (c *LOINCCodec) SearchFields() string { return loincSearchFields }
func (c *LOINCCodec) DisplayFields() string { return loincDisplayFields }

func (c *LOINCCodec) ValidateFormat(code string) error {
	return ValidateFormat(code)
}

func (c *LOINCCodec) Parse(fields []string) coding.Result {
	result := coding.Result{
		CheckedAt: time.Now(),
	}
	if len(fields) < 2 {
		return result
	}
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
	return result
}

func (c *LOINCCodec) SimilarCandidates(code string) []string {
	return transpositionCandidates(code)
}

// transpositionCandidates returns codes formed by swapping adjacent digits,
// with recomputed check digits.
func transpositionCandidates(code string) []string {
	parts := strings.SplitN(code, "-", 2)
	if len(parts) != 2 {
		return nil
	}
	prefix := parts[0]
	seen := make(map[string]bool)
	var candidates []string
	for i := 0; i < len(prefix)-1; i++ {
		b := []byte(prefix)
		b[i], b[i+1] = b[i+1], b[i]
		newPrefix := string(b)
		if newPrefix == prefix {
			continue
		}
		chk := CheckDigit(newPrefix)
		candidate := newPrefix + "-" + string(rune('0'+chk))
		if !seen[candidate] {
			seen[candidate] = true
			candidates = append(candidates, candidate)
		}
	}
	return candidates
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/loinc/...
```

Expected: no output.

- [ ] **Step 3: Run existing loinc tests to confirm nothing broke**

```bash
go test ./internal/loinc/... -v 2>&1 | grep -E "^(=== RUN|--- |PASS|FAIL|ok)"
```

Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/loinc/codec.go
git commit -m "feat: implement LOINC codec wrapping existing client and format logic"
```

---

## Task 3: Implement the ICD-10-CM format validator

**Files:**
- Create: `internal/icd10/format.go`
- Create: `internal/icd10/format_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/icd10/format_test.go`:

```go
package icd10_test

import (
	"strings"
	"testing"

	"github.com/roncofaber/loinc-validator/internal/icd10"
)

func TestValidateFormat(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		// Valid codes
		{"E11.9", false},
		{"I10", false},        // 3-char code, no decimal
		{"Z99.89", false},
		{"S00.00XA", false},   // X placeholder + 7th char
		{"e11.9", false},      // case-insensitive
		{"A00.0", false},
		{"M96.830", false},    // 7 chars

		// Invalid: empty
		{"", true},
		{"   ", true},

		// Invalid: U prefix (reserved)
		{"U07.1", true},
		{"U00.0", true},

		// Invalid: wrong structure
		{"123", true},          // digit in position 1
		{"A1", true},           // too short (only 2 chars)
		{"AB1.0", true},        // position 2 must be digit
		{"A00.00000", true},    // too many chars after decimal (max 4)
		{"A00.", true},         // decimal with no following chars
		{"A0.0", true},         // position 3 wrong (only 2 digits before decimal needs 3 chars minimum)

		// Invalid: non-billable headers accepted by format, rejected by API
		// (A01 is a valid format — 3 chars — but non-billable; format passes)
		{"A01", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := icd10.ValidateFormat(tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for input %q, got nil", tt.input)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for input %q: %v", tt.input, err)
			}
		})
	}
}

func TestValidateFormatErrorMessage(t *testing.T) {
	err := icd10.ValidateFormat("U07.1")
	if err == nil {
		t.Fatal("expected error for U prefix")
	}
	if !strings.Contains(err.Error(), "U") {
		t.Errorf("expected error to mention U prefix, got: %s", err.Error())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/icd10/... 2>&1 | head -5
```

Expected: compile error — `icd10.ValidateFormat` not defined.

- [ ] **Step 3: Create `internal/icd10/format.go`**

```go
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
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/icd10/... -v -run TestValidateFormat 2>&1 | grep -E "^(=== RUN|--- |PASS|FAIL|ok)"
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/icd10/
git commit -m "feat: add ICD-10-CM format validation with U-prefix detection"
```

---

## Task 4: Implement the ICD-10-CM codec

**Files:**
- Create: `internal/icd10/codec.go`
- Create: `internal/icd10/codec_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/icd10/codec_test.go`:

```go
package icd10_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/roncofaber/loinc-validator/internal/icd10"
)

func mockServer(response string, statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		w.Write([]byte(response))
	}))
}

func TestICD10CodecMeta(t *testing.T) {
	c := icd10.NewCodec()
	if c.Name() != "ICD-10-CM" {
		t.Errorf("unexpected Name: %s", c.Name())
	}
	if c.SystemID() != "icd10" {
		t.Errorf("unexpected SystemID: %s", c.SystemID())
	}
	if c.Version() == "" {
		t.Error("expected non-empty Version")
	}
}

func TestICD10CodecValidateFormat(t *testing.T) {
	c := icd10.NewCodec()
	if err := c.ValidateFormat("E11.9"); err != nil {
		t.Errorf("expected nil for valid code, got: %v", err)
	}
	if err := c.ValidateFormat("U07.1"); err == nil {
		t.Error("expected error for U prefix")
	}
}

func TestICD10CodecParse(t *testing.T) {
	c := icd10.NewCodec()
	result := c.Parse([]string{"E11.9", "Type 2 diabetes mellitus without complications"})
	if !result.Valid {
		t.Error("expected valid=true")
	}
	if result.Code != "E11.9" {
		t.Errorf("unexpected Code: %s", result.Code)
	}
	if result.Name != "Type 2 diabetes mellitus without complications" {
		t.Errorf("unexpected Name: %s", result.Name)
	}
	if result.Deprecated {
		t.Error("expected deprecated=false for ICD-10-CM")
	}
}

func TestICD10CodecSimilarCandidatesNil(t *testing.T) {
	c := icd10.NewCodec()
	candidates := c.SimilarCandidates("E11.9")
	if candidates != nil {
		t.Errorf("expected nil candidates for ICD-10-CM, got: %v", candidates)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/icd10/... -run TestICD10Codec 2>&1 | head -5
```

Expected: compile error — `icd10.NewCodec` not defined.

- [ ] **Step 3: Create `internal/icd10/codec.go`**

```go
package icd10

import (
	"time"

	"github.com/roncofaber/loinc-validator/internal/coding"
)

const (
	icd10SearchURL     = "https://clinicaltables.nlm.nih.gov/api/icd10cm/v3/search"
	icd10SearchFields  = "code,name"
	icd10DisplayFields = "code,name"
	icd10Version       = "2026"
)

// ICD10Codec implements coding.Codec for ICD-10-CM diagnosis codes.
type ICD10Codec struct{}

func NewCodec() *ICD10Codec { return &ICD10Codec{} }

func (c *ICD10Codec) Name() string          { return "ICD-10-CM" }
func (c *ICD10Codec) SystemID() string      { return "icd10" }
func (c *ICD10Codec) Version() string       { return icd10Version }
func (c *ICD10Codec) SearchURL() string     { return icd10SearchURL }
func (c *ICD10Codec) SearchFields() string  { return icd10SearchFields }
func (c *ICD10Codec) DisplayFields() string { return icd10DisplayFields }

func (c *ICD10Codec) ValidateFormat(code string) error {
	return ValidateFormat(code)
}

func (c *ICD10Codec) Parse(fields []string) coding.Result {
	result := coding.Result{
		CheckedAt: time.Now(),
	}
	if len(fields) < 2 {
		return result
	}
	result.Valid = true
	result.Code = fields[0]
	result.Name = fields[1]
	// ICD-10-CM has no deprecated prefix convention and no check digit.
	// Deprecated=false always; no ShortName/Component/DataType/Units.
	return result
}

// SimilarCandidates returns nil — ICD-10-CM has no check digit mechanism
// for generating transposition candidates.
func (c *ICD10Codec) SimilarCandidates(code string) []string {
	return nil
}
```

- [ ] **Step 4: Run all icd10 tests**

```bash
go test ./internal/icd10/... -v 2>&1 | grep -E "^(=== RUN|--- |PASS|FAIL|ok)"
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/icd10/
git commit -m "feat: implement ICD-10-CM codec"
```

---

## Task 5: Refactor handlers to accept `coding.Codec`

**Files:**
- Modify: `internal/handlers/validate.go`
- Modify: `internal/handlers/batch.go`
- Modify: `internal/handlers/suggest.go`
- Modify: `internal/handlers/similar.go`

- [ ] **Step 1: Rewrite `internal/handlers/validate.go`**

```go
package handlers

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/roncofaber/loinc-validator/internal/coding"
)

type ValidateHandler struct {
	tmpl  *template.Template
	codec coding.Codec
	http  *coding.HTTPClient
}

func NewValidateHandler(tmpl *template.Template, codec coding.Codec) *ValidateHandler {
	return &ValidateHandler{tmpl: tmpl, codec: codec, http: coding.NewHTTPClient()}
}

type resultData struct {
	Code         string
	Name         string
	ShortName    string
	Component    string
	RelatedNames string
	DataType     string
	Units        []string
	Valid         bool
	Deprecated    bool
	CheckedAt    interface{}
	Error        string
	Suggestion   *coding.Suggestion
	SimilarCode  string
	SystemID     string
}

func (h *ValidateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	code := strings.TrimSpace(r.FormValue("code"))

	if err := h.codec.ValidateFormat(code); err != nil {
		data := resultData{Error: err.Error(), SystemID: h.codec.SystemID()}
		// For codecs with check digit (LOINC): try corrected code
		if corrected := correctedCode(h.codec, code); corrected != "" {
			rows, apiErr := h.http.Validate(h.codec, corrected)
			if apiErr == nil {
				if row, _ := coding.ExactMatch(rows, corrected); row != nil {
					res := h.codec.Parse(row)
					data.Suggestion = &coding.Suggestion{Code: res.Code, Name: res.Name}
				} else {
					data.SimilarCode = corrected
				}
			}
		}
		h.tmpl.ExecuteTemplate(w, "result.html", data)
		return
	}

	rows, err := h.http.Validate(h.codec, code)
	if err != nil {
		h.tmpl.ExecuteTemplate(w, "result.html", resultData{
			Code:     code,
			Error:    "Could not reach the API — please try again.",
			SystemID: h.codec.SystemID(),
		})
		return
	}

	row, _ := coding.ExactMatch(rows, code)
	if row == nil {
		h.tmpl.ExecuteTemplate(w, "result.html", resultData{
			Code:     code,
			SystemID: h.codec.SystemID(),
		})
		return
	}

	res := h.codec.Parse(row)
	// Fetch extra fields for LOINC (units, datatype, relatednames) via the existing loinc client
	// Note: extra fields are LOINC-specific and fetched by the loinc.Client; the codec.Parse
	// receives only df fields. For now extra fields are not populated for ICD-10-CM (empty strings).
	h.tmpl.ExecuteTemplate(w, "result.html", resultData{
		Code:         res.Code,
		Name:         res.Name,
		ShortName:    res.ShortName,
		Component:    res.Component,
		RelatedNames: res.RelatedNames,
		DataType:     res.DataType,
		Units:        res.Units,
		Valid:         res.Valid,
		Deprecated:    res.Deprecated,
		CheckedAt:    res.CheckedAt,
		SystemID:     h.codec.SystemID(),
	})
}
```

- [ ] **Step 2: Rewrite `internal/handlers/batch.go`**

```go
package handlers

import (
	"bufio"
	"encoding/json"
	"html/template"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/roncofaber/loinc-validator/internal/coding"
)

const maxWorkers = 10
const maxFileSize = 5 << 20

type BatchHandler struct {
	tmpl  *template.Template
	codec coding.Codec
	http  *coding.HTTPClient
}

func NewBatchHandler(tmpl *template.Template, codec coding.Codec) *BatchHandler {
	return &BatchHandler{tmpl: tmpl, codec: codec, http: coding.NewHTTPClient()}
}

type batchSummary struct {
	Total   int
	Valid   int
	Invalid int
	Errors  int
}

type batchTemplateData struct {
	Results     []coding.Result
	ResultsJSON string
	Summary     batchSummary
	Error       string
}

func (h *BatchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxFileSize)
	file, _, err := r.FormFile("file")
	if err != nil {
		msg := "Please upload a file (no file received)."
		if err.Error() == "http: request body too large" {
			msg = "File too large — maximum size is 5 MB."
		}
		h.tmpl.ExecuteTemplate(w, "batch_result.html", batchTemplateData{Error: msg})
		return
	}
	defer file.Close()

	var codes []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if i := strings.Index(line, "#"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if line != "" {
			codes = append(codes, line)
		}
	}

	if len(codes) == 0 {
		h.tmpl.ExecuteTemplate(w, "batch_result.html", batchTemplateData{
			Error: "The uploaded file is empty or contains no valid lines.",
		})
		return
	}

	results := h.validateConcurrent(codes)
	summary := batchSummary{Total: len(results)}
	for _, res := range results {
		switch {
		case res.Error != "":
			summary.Errors++
		case res.Valid:
			summary.Valid++
		default:
			summary.Invalid++
		}
	}

	jsonBytes, _ := json.Marshal(results)
	h.tmpl.ExecuteTemplate(w, "batch_result.html", batchTemplateData{
		Results:     results,
		ResultsJSON: string(jsonBytes),
		Summary:     summary,
	})
}

func (h *BatchHandler) validateConcurrent(codes []string) []coding.Result {
	results := make([]coding.Result, len(codes))
	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup

	for i, code := range codes {
		wg.Add(1)
		go func(idx int, c string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := h.codec.ValidateFormat(c); err != nil {
				results[idx] = coding.Result{Code: c, Error: err.Error(), CheckedAt: time.Now()}
				return
			}
			rows, err := h.http.Validate(h.codec, c)
			if err != nil {
				results[idx] = coding.Result{Code: c, Error: "API error: " + err.Error(), CheckedAt: time.Now()}
				return
			}
			row, _ := coding.ExactMatch(rows, c)
			if row == nil {
				results[idx] = coding.Result{Code: c, CheckedAt: time.Now()}
				return
			}
			results[idx] = h.codec.Parse(row)
		}(i, code)
	}

	wg.Wait()
	return results
}
```

- [ ] **Step 3: Rewrite `internal/handlers/suggest.go`**

```go
package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/roncofaber/loinc-validator/internal/coding"
)

type SuggestHandler struct {
	tmpl  *template.Template
	codec coding.Codec
	http  *coding.HTTPClient
}

func NewSuggestHandler(tmpl *template.Template, codec coding.Codec) *SuggestHandler {
	return &SuggestHandler{tmpl: tmpl, codec: codec, http: coding.NewHTTPClient()}
}

type suggestion struct {
	Code string
	Name string
}

func (h *SuggestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("code"))
	if len(q) < 2 {
		w.WriteHeader(http.StatusOK)
		return
	}

	rows, err := h.http.Suggest(h.codec, q, 6)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	var suggestions []suggestion
	for _, row := range rows {
		if len(row) >= 2 {
			suggestions = append(suggestions, suggestion{Code: row[0], Name: row[1]})
		}
	}

	h.tmpl.ExecuteTemplate(w, "suggest.html", suggestions)
	fmt.Fprint(w, "")
}
```

- [ ] **Step 4: Rewrite `internal/handlers/similar.go`**

```go
package handlers

import (
	"html/template"
	"net/http"
	"strings"
	"sync"

	"github.com/roncofaber/loinc-validator/internal/coding"
)

type SimilarHandler struct {
	tmpl  *template.Template
	codec coding.Codec
	http  *coding.HTTPClient
}

func NewSimilarHandler(tmpl *template.Template, codec coding.Codec) *SimilarHandler {
	return &SimilarHandler{tmpl: tmpl, codec: codec, http: coding.NewHTTPClient()}
}

func (h *SimilarHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		return
	}

	candidates := h.codec.SimilarCandidates(code)
	if len(candidates) == 0 {
		return
	}

	results := make([]coding.Result, len(candidates))
	var wg sync.WaitGroup
	for i, candidate := range candidates {
		wg.Add(1)
		go func(idx int, c string) {
			defer wg.Done()
			rows, err := h.http.Validate(h.codec, c)
			if err != nil {
				return
			}
			row, _ := coding.ExactMatch(rows, c)
			if row != nil {
				results[idx] = h.codec.Parse(row)
			}
		}(i, candidate)
	}
	wg.Wait()

	var suggestions []coding.Suggestion
	seen := make(map[string]bool)
	for _, res := range results {
		if res.Valid && !seen[res.Code] {
			seen[res.Code] = true
			suggestions = append(suggestions, coding.Suggestion{Code: res.Code, Name: res.Name})
		}
	}

	if len(suggestions) == 0 {
		return
	}

	h.tmpl.ExecuteTemplate(w, "similar.html", suggestions)
}
```

- [ ] **Step 5: Add `correctedCode` helper to `validate.go`** (replaces the LOINC-specific `loinc.CorrectedCode` call)

Add this function at the bottom of `internal/handlers/validate.go`:

```go
// correctedCode returns the code with the correct check digit if the codec
// supports it (LOINC only). For other codecs it returns "".
// It does this by checking if the codec's ValidateFormat fails with a check
// digit message, then computing the corrected form.
func correctedCode(codec coding.Codec, code string) string {
	// Only LOINC has a check digit — detect by SystemID.
	if codec.SystemID() != "loinc" {
		return ""
	}
	// Import the loinc package to use CorrectedCode.
	// This is the only place handlers reference a specific codec's internals.
	// If more codecs gain check digits, this becomes a Codec method.
	return loincCorrectedCode(code)
}
```

Also add this import shim at the top of `validate.go` (import the loinc package):

```go
import (
	"html/template"
	"net/http"
	"strings"

	"github.com/roncofaber/loinc-validator/internal/coding"
	loincpkg "github.com/roncofaber/loinc-validator/internal/loinc"
)

func loincCorrectedCode(code string) string {
	return loincpkg.CorrectedCode(code)
}
```

- [ ] **Step 6: Build to check for errors**

```bash
go build ./...
```

Expected: no output.

- [ ] **Step 7: Run all tests**

```bash
go test ./... 2>&1
```

Expected: all packages pass (some tests may need updating in next task).

- [ ] **Step 8: Commit**

```bash
git add internal/handlers/
git commit -m "refactor: make all handlers accept coding.Codec instead of loinc.Client"
```

---

## Task 6: Update handler tests and export handler

**Files:**
- Modify: `internal/handlers/validate_test.go`
- Modify: `internal/handlers/batch_test.go`
- Modify: `internal/handlers/export_test.go`
- Modify: `internal/handlers/export.go`

- [ ] **Step 1: Update `internal/handlers/validate_test.go`**

```go
package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/roncofaber/loinc-validator/internal/handlers"
	"github.com/roncofaber/loinc-validator/internal/loinc"
)

func TestValidateHandlerEmptyInput(t *testing.T) {
	tmpl := handlers.MustLoadTemplates("../../templates")
	h := handlers.NewValidateHandler(tmpl, loinc.NewCodec())

	form := url.Values{"code": {""}}
	req := httptest.NewRequest(http.MethodPost, "/loinc/validate", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "cannot be empty") {
		t.Errorf("expected empty-input error message, got: %s", rec.Body.String())
	}
}

func TestValidateHandlerMalformedCode(t *testing.T) {
	tmpl := handlers.MustLoadTemplates("../../templates")
	h := handlers.NewValidateHandler(tmpl, loinc.NewCodec())

	form := url.Values{"code": {"notacode"}}
	req := httptest.NewRequest(http.MethodPost, "/loinc/validate", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "invalid LOINC format") {
		t.Errorf("expected format error message, got: %s", rec.Body.String())
	}
}
```

- [ ] **Step 2: Update `internal/handlers/batch_test.go`**

```go
package handlers_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/roncofaber/loinc-validator/internal/handlers"
	"github.com/roncofaber/loinc-validator/internal/loinc"
)

func makeFileUpload(t *testing.T, content string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("file", "codes.txt")
	if err != nil {
		t.Fatal(err)
	}
	fw.Write([]byte(content))
	w.Close()
	req := httptest.NewRequest(http.MethodPost, "/loinc/batch", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func TestBatchHandlerNoFile(t *testing.T) {
	tmpl := handlers.MustLoadTemplates("../../templates")
	h := handlers.NewBatchHandler(tmpl, loinc.NewCodec())

	req := httptest.NewRequest(http.MethodPost, "/loinc/batch", nil)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=xxx")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), "no file") {
		t.Errorf("expected 'no file' error, got: %s", rec.Body.String())
	}
}

func TestBatchHandlerEmptyFile(t *testing.T) {
	tmpl := handlers.MustLoadTemplates("../../templates")
	h := handlers.NewBatchHandler(tmpl, loinc.NewCodec())

	req := makeFileUpload(t, "")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), "empty") {
		t.Errorf("expected 'empty' error, got: %s", rec.Body.String())
	}
}
```

- [ ] **Step 3: Update `internal/handlers/export.go` to use `coding.Result`**

```go
package handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/roncofaber/loinc-validator/internal/coding"
)

type ExportHandler struct {
	tmpl *template.Template
}

func NewExportHandler(tmpl *template.Template) *ExportHandler {
	return &ExportHandler{tmpl: tmpl}
}

func (h *ExportHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rawJSON := r.FormValue("results")
	var results []coding.Result
	if err := json.Unmarshal([]byte(rawJSON), &results); err != nil {
		http.Error(w, "invalid results data", http.StatusBadRequest)
		return
	}

	filename := fmt.Sprintf("validation_%s.csv", time.Now().UTC().Format("20060102_150405"))
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)

	cw := csv.NewWriter(w)
	cw.Write([]string{"Code", "Valid", "Deprecated", "Name", "CheckedAt", "Error"})
	for _, res := range results {
		valid := "false"
		if res.Valid {
			valid = "true"
		}
		deprecated := "false"
		if res.Deprecated {
			deprecated = "true"
		}
		cw.Write([]string{
			res.Code,
			valid,
			deprecated,
			res.Name,
			res.CheckedAt.UTC().Format(time.RFC3339),
			res.Error,
		})
	}
	cw.Flush()
}
```

- [ ] **Step 4: Update `internal/handlers/export_test.go`**

```go
package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/roncofaber/loinc-validator/internal/coding"
	"github.com/roncofaber/loinc-validator/internal/handlers"
)

func TestExportHandler(t *testing.T) {
	tmpl := handlers.MustLoadTemplates("../../templates")
	h := handlers.NewExportHandler(tmpl)

	results := []coding.Result{
		{Code: "2345-7", Name: "Glucose", Valid: true, CheckedAt: time.Now()},
		{Code: "99999-9", Valid: false, CheckedAt: time.Now()},
	}
	jsonBytes, _ := json.Marshal(results)

	form := url.Values{"results": {string(jsonBytes)}}
	req := httptest.NewRequest(http.MethodPost, "/export", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "text/csv") {
		t.Errorf("expected text/csv, got %s", rec.Header().Get("Content-Type"))
	}
	body := rec.Body.String()
	if !strings.Contains(body, "2345-7") {
		t.Errorf("expected code 2345-7 in CSV, got: %s", body)
	}
	if !strings.Contains(body, "Code,Valid,Deprecated,Name") {
		t.Errorf("expected CSV header, got: %s", body)
	}
}

func TestExportHandlerInvalidJSON(t *testing.T) {
	tmpl := handlers.MustLoadTemplates("../../templates")
	h := handlers.NewExportHandler(tmpl)

	form := url.Values{"results": {"not json"}}
	req := httptest.NewRequest(http.MethodPost, "/export", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}
```

- [ ] **Step 5: Build and run all tests**

```bash
go build ./... && go test ./... 2>&1
```

Expected: all packages pass.

- [ ] **Step 6: Commit**

```bash
git add internal/handlers/
git commit -m "refactor: update handlers and tests to use coding.Result; export uses coding.Result"
```

---

## Task 7: Update `main.go` with codec slice and dynamic routes

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Rewrite `main.go`**

```go
package main

import (
	"log"
	"net/http"

	"github.com/roncofaber/loinc-validator/internal/coding"
	"github.com/roncofaber/loinc-validator/internal/handlers"
	"github.com/roncofaber/loinc-validator/internal/icd10"
	"github.com/roncofaber/loinc-validator/internal/loinc"
)

func main() {
	tmpl := handlers.MustLoadTemplates("templates")

	codecs := []coding.Codec{
		loinc.NewCodec(),
		icd10.NewCodec(),
	}

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.Handle("/export", handlers.NewExportHandler(tmpl))

	for _, codec := range codecs {
		id := codec.SystemID()
		mux.Handle("/"+id+"/validate",        handlers.NewValidateHandler(tmpl, codec))
		mux.Handle("/"+id+"/suggest",         handlers.NewSuggestHandler(tmpl, codec))
		mux.Handle("/"+id+"/suggest-similar", handlers.NewSimilarHandler(tmpl, codec))
		mux.Handle("/"+id+"/batch",           handlers.NewBatchHandler(tmpl, codec))
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		tmpl.ExecuteTemplate(w, "index.html", map[string]any{
			"Codecs": codecs,
		})
	})

	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
```

- [ ] **Step 2: Build to confirm**

```bash
go build ./...
```

Expected: no output.

- [ ] **Step 3: Commit**

```bash
git add main.go
git commit -m "refactor: register codec-driven routes dynamically in main.go"
```

---

## Task 8: Update templates — tab bar and system-scoped forms

**Files:**
- Modify: `templates/index.html`
- Create: `templates/loinc/tab.html`
- Create: `templates/icd10/tab.html`
- Modify: `templates/partials/result.html` (scope IDs)
- Modify: `templates/partials/batch_result.html` (use coding.Result fields)
- Modify: `templates/partials/similar.html` (use coding.Suggestion)

- [ ] **Step 1: Create `templates/loinc/tab.html`** (extracted from current index.html single+batch sections)

```html
<section class="card">
  <p class="card-title">Single Code</p>
  <button class="card-reset" id="loinc-single-reset" aria-label="Clear"
          onclick="document.getElementById('loinc-single-form').reset();
                   document.getElementById('loinc-result').innerHTML='';
                   document.getElementById('loinc-suggestions').innerHTML='';
                   this.style.display='none';"
          style="display:none">
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/>
      <path d="M3 3v5h5"/>
    </svg>
    Clear
  </button>
  <form id="loinc-single-form"
        hx-post="/loinc/validate"
        hx-target="#loinc-result"
        hx-swap="innerHTML"
        hx-indicator="#loinc-single-indicator"
        hx-on:submit="document.getElementById('loinc-suggestions').innerHTML='';
                      document.getElementById('loinc-single-reset').style.display='flex';">
    <label for="loinc-code">LOINC Code or name</label>
    <div class="input-wrap">
      <input type="text" id="loinc-code" name="code" placeholder="e.g. 2345-7 or &quot;glucose&quot;"
             autocomplete="off" spellcheck="false"
             hx-get="/loinc/suggest"
             hx-trigger="input changed delay:300ms"
             hx-target="#loinc-suggestions"
             hx-swap="innerHTML"
             hx-include="this"
             hx-params="code"
             hx-indicator="#loinc-suggest-progress"
             onclick="document.getElementById('loinc-suggestions').innerHTML=''"
             onblur="setTimeout(function(){document.getElementById('loinc-suggestions').innerHTML=''},200)">
      <div id="loinc-suggestions" class="suggest-dropdown"></div>
      <div class="progress-bar" id="loinc-suggest-progress"></div>
    </div>
    <div style="margin-top:0.75rem;">
      <button class="btn" type="submit">Validate</button>
      <span id="loinc-single-indicator" class="indicator">Checking</span>
    </div>
  </form>
  <div id="loinc-result"></div>
</section>

<section class="card">
  <p class="card-title">Batch Validation</p>
  <button class="card-reset" id="loinc-batch-reset" aria-label="Clear"
          onclick="document.getElementById('loinc-batch-form').reset();
                   document.getElementById('loinc-batch-result').innerHTML='';
                   this.style.display='none';"
          style="display:none">
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/>
      <path d="M3 3v5h5"/>
    </svg>
    Clear
  </button>
  <form id="loinc-batch-form"
        hx-post="/loinc/batch"
        hx-target="#loinc-batch-result"
        hx-swap="innerHTML"
        hx-indicator="#loinc-batch-indicator, #loinc-batch-progress"
        hx-encoding="multipart/form-data"
        hx-on:submit="document.getElementById('loinc-batch-reset').style.display='flex';">
    <label for="loinc-file">File upload</label>
    <p class="hint">Plain text or CSV — one LOINC code per line. Lines starting with # are ignored.</p>
    <input type="file" id="loinc-file" name="file" accept=".txt,.csv">
    <div>
      <button class="btn" type="submit">Validate File</button>
      <span id="loinc-batch-indicator" class="indicator">Processing</span>
    </div>
  </form>
  <div class="progress-bar" id="loinc-batch-progress"></div>
  <div id="loinc-batch-result"></div>
</section>
```

- [ ] **Step 2: Create `templates/icd10/tab.html`**

```html
<section class="card">
  <p class="card-title">Single Code</p>
  <button class="card-reset" id="icd10-single-reset" aria-label="Clear"
          onclick="document.getElementById('icd10-single-form').reset();
                   document.getElementById('icd10-result').innerHTML='';
                   document.getElementById('icd10-suggestions').innerHTML='';
                   this.style.display='none';"
          style="display:none">
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/>
      <path d="M3 3v5h5"/>
    </svg>
    Clear
  </button>
  <form id="icd10-single-form"
        hx-post="/icd10/validate"
        hx-target="#icd10-result"
        hx-swap="innerHTML"
        hx-indicator="#icd10-single-indicator"
        hx-on:submit="document.getElementById('icd10-suggestions').innerHTML='';
                      document.getElementById('icd10-single-reset').style.display='flex';">
    <label for="icd10-code">ICD-10-CM Code or name</label>
    <div class="input-wrap">
      <input type="text" id="icd10-code" name="code" placeholder="e.g. E11.9 or &quot;diabetes&quot;"
             autocomplete="off" spellcheck="false"
             hx-get="/icd10/suggest"
             hx-trigger="input changed delay:300ms"
             hx-target="#icd10-suggestions"
             hx-swap="innerHTML"
             hx-include="this"
             hx-params="code"
             hx-indicator="#icd10-suggest-progress"
             onclick="document.getElementById('icd10-suggestions').innerHTML=''"
             onblur="setTimeout(function(){document.getElementById('icd10-suggestions').innerHTML=''},200)">
      <div id="icd10-suggestions" class="suggest-dropdown"></div>
      <div class="progress-bar" id="icd10-suggest-progress"></div>
    </div>
    <div style="margin-top:0.75rem;">
      <button class="btn" type="submit">Validate</button>
      <span id="icd10-single-indicator" class="indicator">Checking</span>
    </div>
  </form>
  <div id="icd10-result"></div>
</section>

<section class="card">
  <p class="card-title">Batch Validation</p>
  <button class="card-reset" id="icd10-batch-reset" aria-label="Clear"
          onclick="document.getElementById('icd10-batch-form').reset();
                   document.getElementById('icd10-batch-result').innerHTML='';
                   this.style.display='none';"
          style="display:none">
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/>
      <path d="M3 3v5h5"/>
    </svg>
    Clear
  </button>
  <form id="icd10-batch-form"
        hx-post="/icd10/batch"
        hx-target="#icd10-batch-result"
        hx-swap="innerHTML"
        hx-indicator="#icd10-batch-indicator, #icd10-batch-progress"
        hx-encoding="multipart/form-data"
        hx-on:submit="document.getElementById('icd10-batch-reset').style.display='flex';">
    <label for="icd10-file">File upload</label>
    <p class="hint">Plain text or CSV — one ICD-10-CM code per line. Lines starting with # are ignored.</p>
    <input type="file" id="icd10-file" name="file" accept=".txt,.csv">
    <div>
      <button class="btn" type="submit">Validate File</button>
      <span id="icd10-batch-indicator" class="indicator">Processing</span>
    </div>
  </form>
  <div class="progress-bar" id="icd10-batch-progress"></div>
  <div id="icd10-batch-result"></div>
</section>
```

- [ ] **Step 3: Rewrite `templates/index.html`**

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Medical Code Validator</title>
  <link rel="icon" href="data:image/svg+xml,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 100 100'><text y='.9em' font-size='90'>🐷</text></svg>">
  <link rel="stylesheet" href="/static/style.css">
  <script src="https://unpkg.com/htmx.org@2.0.4"></script>
</head>
<body>
  <div class="page-wrap">

    <header class="site-header">
      <h1><span class="h1-loinc">Medical Code</span> <span class="h1-validator">Validator</span></h1>
      <p>Validate medical coding system codes against the NIH Clinical Tables API</p>
    </header>

    <div class="tab-bar">
      {{range $i, $c := .Codecs}}
      <button class="tab-btn{{if eq $i 0}} active{{end}}"
              data-tab="{{$c.SystemID}}"
              onclick="switchTab('{{$c.SystemID}}')">
        {{$c.Name}} <span class="tab-version">v{{$c.Version}}</span>
      </button>
      {{end}}
    </div>

    {{range $i, $c := .Codecs}}
    <div id="tab-{{$c.SystemID}}" class="tab-panel{{if eq $i 0}} active{{end}}">
      {{template (printf "%s/tab.html" $c.SystemID) $c}}
    </div>
    {{end}}

  </div>

  <footer class="site-footer">
    Data from <a href="https://clinicaltables.nlm.nih.gov/" target="_blank" rel="noopener">NIH Clinical Tables API</a> &nbsp;·&nbsp;
    <a href="https://loinc.org/" target="_blank" rel="noopener">LOINC®</a> is a registered trademark of Regenstrief Institute &nbsp;·&nbsp;
    <a href="https://github.com/roncofaber/loinc-validator" target="_blank" rel="noopener">
      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" style="width:13px;height:13px;vertical-align:middle;margin-right:0.2rem;">
        <path d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0 1 12 6.844a9.59 9.59 0 0 1 2.504.337c1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.02 10.02 0 0 0 22 12.017C22 6.484 17.522 2 12 2z"/>
      </svg>GitHub</a>
  </footer>

  <script>
  function switchTab(id) {
    document.querySelectorAll('.tab-panel').forEach(p => p.classList.remove('active'));
    document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
    document.getElementById('tab-' + id).classList.add('active');
    document.querySelector('[data-tab="' + id + '"]').classList.add('active');
  }
  </script>
</body>
</html>
```

- [ ] **Step 4: Update `templates/partials/result.html`** — replace hardcoded suggest-similar URL with dynamic system ID

```html
{{if .Error}}
<div class="result-box error">
  <span class="status-badge error-tag">Error</span>
  <p>{{.Error}}</p>
  {{if .Suggestion}}
  <div class="similar-suggestions">
    <p class="similar-label">Did you mean?</p>
    <ul class="suggest-list">
      <li class="suggest-item"
          onmousedown="event.preventDefault();
                       document.getElementById('{{.SystemID}}-code').value='{{.Suggestion.Code}}';
                       document.getElementById('{{.SystemID}}-suggestions').innerHTML='';
                       document.getElementById('{{.SystemID}}-single-form').requestSubmit();">
        <span class="suggest-code">{{.Suggestion.Code}}</span>
        <span class="suggest-name">{{.Suggestion.Name}}</span>
      </li>
    </ul>
  </div>
  {{end}}
  {{if .SimilarCode}}
  <div id="{{.SystemID}}-similar-results"
       hx-get="/{{.SystemID}}/suggest-similar?code={{.SimilarCode}}"
       hx-trigger="load"
       hx-swap="innerHTML">
  </div>
  {{end}}
</div>
{{else if .Valid}}
<div class="result-box valid">
  <span class="status-badge valid">Valid</span>
  <table class="result-fields">
    <tr class="copyable-row">
      <td>Code</td>
      <td class="code-value">
        <span class="field-value">{{.Code}}</span>
        <button class="copy-btn" onclick="copyText(this, '{{.Code}}')" aria-label="Copy code">
          <svg class="icon-copy" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <rect x="5" y="4" width="8" height="10" rx="1.5"/>
            <path d="M3 12H2.5A1.5 1.5 0 0 1 1 10.5v-8A1.5 1.5 0 0 1 2.5 1h7A1.5 1.5 0 0 1 11 2.5V4"/>
          </svg>
          <svg class="icon-check" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="2.5,8.5 6,12 13.5,4.5"/>
          </svg>
        </button>
      </td>
    </tr>
    <tr class="copyable-row">
      <td>Name</td>
      <td>
        <span class="field-value">{{.Name}}</span>
        <button class="copy-btn" onclick="copyText(this, '{{.Name}}')" aria-label="Copy name">
          <svg class="icon-copy" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <rect x="5" y="4" width="8" height="10" rx="1.5"/>
            <path d="M3 12H2.5A1.5 1.5 0 0 1 1 10.5v-8A1.5 1.5 0 0 1 2.5 1h7A1.5 1.5 0 0 1 11 2.5V4"/>
          </svg>
          <svg class="icon-check" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="2.5,8.5 6,12 13.5,4.5"/>
          </svg>
        </button>
      </td>
    </tr>
    <tr>
      <td>Checked</td>
      <td>{{.CheckedAt.Format "2006-01-02 15:04:05 UTC"}}</td>
    </tr>
  </table>

  {{if or .ShortName .Component .DataType .Units .RelatedNames}}
  <details>
    <summary>Advanced details</summary>
    <table class="result-fields" style="margin-top:0.75rem;">
      {{if .ShortName}}<tr><td>Short name</td><td>{{.ShortName}}</td></tr>{{end}}
      {{if .Component}}<tr><td>Component</td><td>{{.Component}}</td></tr>{{end}}
      {{if .DataType}}<tr><td>Data type</td><td>{{.DataType}}</td></tr>{{end}}
      {{if .Units}}<tr><td>Units</td><td>{{range $i, $u := .Units}}{{if $i}}, {{end}}{{$u}}{{end}}</td></tr>{{end}}
      {{if .RelatedNames}}<tr><td>Related</td><td class="related">{{.RelatedNames}}</td></tr>{{end}}
    </table>
  </details>
  {{end}}
</div>
{{if .Deprecated}}
<div class="deprecated-warn">
  <strong>Deprecated:</strong> This code should not be used for new implementations. Consider finding an active replacement.
</div>
{{end}}
{{else}}
<div class="result-box invalid">
  <span class="status-badge invalid">Not found</span>
  <p><span class="code-value">{{.Code}}</span> was not found in the database.</p>
  <div id="{{.SystemID}}-similar-results"
       hx-get="/{{.SystemID}}/suggest-similar?code={{.Code}}"
       hx-trigger="load"
       hx-swap="innerHTML">
  </div>
</div>
{{end}}

<script>
function copyText(btn, text) {
  navigator.clipboard.writeText(text).then(function() {
    btn.classList.add('copy-btn--done');
    setTimeout(function() { btn.classList.remove('copy-btn--done'); }, 1500);
  });
}
</script>
```

- [ ] **Step 5: Update `templates/partials/similar.html`** to use `coding.Suggestion` (same fields, no change needed — already uses `.Code` and `.Name`)

Verify current similar.html already works with `coding.Suggestion` (it does — same field names).

- [ ] **Step 6: Add tab CSS to `static/style.css`**

```css
/* ── Tab bar ─────────────────────────────────────────────────── */
.tab-bar {
  display: flex;
  gap: 0.25rem;
  margin-bottom: 1.5rem;
  border-bottom: 2px solid var(--border);
  padding-bottom: 0;
}

.tab-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.5rem 1.1rem;
  font-family: var(--font-sans);
  font-size: 0.82rem;
  font-weight: 600;
  color: var(--text-muted);
  background: transparent;
  border: none;
  border-bottom: 2px solid transparent;
  margin-bottom: -2px;
  cursor: pointer;
  transition: color 0.15s, border-color 0.15s;
}

.tab-btn:hover { color: var(--text); }

.tab-btn.active {
  color: var(--accent);
  border-bottom-color: var(--accent);
}

.tab-version {
  font-size: 0.68rem;
  font-weight: 400;
  color: var(--text-faint);
  font-family: var(--font-mono);
}

.tab-panel { display: none; }
.tab-panel.active { display: block; }
```

- [ ] **Step 7: Update `handlers/templates.go`** to load system subdirectory templates

```go
package handlers

import (
	"html/template"
	"path/filepath"
)

func MustLoadTemplates(dir string) *template.Template {
	tmpl, err := template.ParseGlob(filepath.Join(dir, "*.html"))
	if err != nil {
		panic("failed to parse base templates: " + err.Error())
	}
	// Load partials
	partials, err := filepath.Glob(filepath.Join(dir, "partials", "*.html"))
	if err != nil {
		panic("failed to glob partials: " + err.Error())
	}
	if len(partials) > 0 {
		tmpl, err = tmpl.ParseFiles(partials...)
		if err != nil {
			panic("failed to parse partial templates: " + err.Error())
		}
	}
	// Load system tab templates (loinc/tab.html, icd10/tab.html, etc.)
	systemTabs, err := filepath.Glob(filepath.Join(dir, "*/tab.html"))
	if err != nil {
		panic("failed to glob system tabs: " + err.Error())
	}
	if len(systemTabs) > 0 {
		tmpl, err = tmpl.ParseFiles(systemTabs...)
		if err != nil {
			panic("failed to parse system tab templates: " + err.Error())
		}
	}
	return tmpl
}
```

- [ ] **Step 8: Build and verify**

```bash
go build ./...
go run . &
sleep 1
curl -s http://localhost:8080/ | grep "tab-btn"
kill %1 2>/dev/null; wait 2>/dev/null
```

Expected: HTML containing `tab-btn` for both systems.

- [ ] **Step 9: Run all tests**

```bash
go test ./... 2>&1
```

Expected: all pass.

- [ ] **Step 10: Commit**

```bash
git add templates/ static/style.css internal/handlers/templates.go
git commit -m "feat: add tabbed UI with LOINC and ICD-10-CM panels; system-scoped IDs"
```

---

## Task 9: Handle LOINC extra fields (units, datatype, relatednames) via codec

**Context:** The current LOINC `client.go` fetches extra fields (`ef=RELATEDNAMES2,datatype,units`) in the same API call. The new shared `coding.HTTPClient` doesn't handle `ef` params. We need the LOINC codec's validate path to still populate these fields.

**Files:**
- Modify: `internal/loinc/codec.go` — add `ValidateWithExtras` method used by the validate handler
- Modify: `internal/handlers/validate.go` — use LOINC-specific extras when available

- [ ] **Step 1: Add `ValidateWithExtras` to the LOINC codec**

Add to `internal/loinc/codec.go`:

```go
// ExtraResult extends coding.Result with LOINC-specific extra fields.
type ExtraResult struct {
	coding.Result
	RelatedNames string
	DataType     string
	Units        []string
}

// ValidateWithExtras calls the LOINC API with ef fields and returns full result.
// Used by the validate handler for single-code lookups.
func (c *LOINCCodec) ValidateWithExtras(code string) (ExtraResult, error) {
	res, err := defaultClient.Validate(code)
	if err != nil {
		return ExtraResult{Result: coding.Result{Code: code, CheckedAt: time.Now()}}, err
	}
	base := coding.Result{
		Code:       res.Code,
		Name:       res.Name,
		ShortName:  res.ShortName,
		Component:  res.Component,
		Valid:       res.Valid,
		Deprecated:  res.Deprecated,
		CheckedAt:  res.CheckedAt,
	}
	return ExtraResult{
		Result:       base,
		RelatedNames: res.RelatedNames,
		DataType:     res.DataType,
		Units:        res.Units,
	}, nil
}

// defaultClient is the existing LOINC client (with ef support).
var defaultClient = NewDefaultClient()
```

- [ ] **Step 2: Update `internal/handlers/validate.go` to use extras for LOINC**

In the `ServeHTTP` method, replace the generic validate path with:

```go
// Use LOINC extras if codec supports it.
type extrasProvider interface {
	ValidateWithExtras(code string) (interface{ GetResult() coding.Result; GetRelatedNames() string; GetDataType() string; GetUnits() []string }, error)
}

// Simpler: type-assert to *loinc.LOINCCodec
if loincCodec, ok := h.codec.(interface {
	ValidateWithExtras(string) (loinc.ExtraResult, error)
}); ok {
	extra, err := loincCodec.ValidateWithExtras(code)
	if err != nil {
		h.tmpl.ExecuteTemplate(w, "result.html", resultData{
			Code: code, Error: "Could not reach the API — please try again.", SystemID: h.codec.SystemID(),
		})
		return
	}
	h.tmpl.ExecuteTemplate(w, "result.html", resultData{
		Code:         extra.Code,
		Name:         extra.Name,
		ShortName:    extra.ShortName,
		Component:    extra.Component,
		RelatedNames: extra.RelatedNames,
		DataType:     extra.DataType,
		Units:        extra.Units,
		Valid:         extra.Valid,
		Deprecated:    extra.Deprecated,
		CheckedAt:    extra.CheckedAt,
		SystemID:     h.codec.SystemID(),
	})
	return
}
```

Then keep the generic path (for ICD-10-CM and future codecs) as the fallback.

- [ ] **Step 3: Build and run tests**

```bash
go build ./... && go test ./... 2>&1
```

Expected: all pass.

- [ ] **Step 4: Commit**

```bash
git add internal/loinc/codec.go internal/handlers/validate.go
git commit -m "feat: preserve LOINC extra fields (units, datatype, relatednames) via codec type assertion"
```

---

## Task 10: Integration smoke test and README update

**Files:**
- Create: `internal/icd10/integration_test.go`
- Modify: `README.md`

- [ ] **Step 1: Create `internal/icd10/integration_test.go`**

```go
package icd10_test

import (
	"os"
	"testing"

	"github.com/roncofaber/loinc-validator/internal/coding"
	"github.com/roncofaber/loinc-validator/internal/icd10"
)

func TestIntegrationValidCode(t *testing.T) {
	if os.Getenv("ICD10_INTEGRATION") == "" {
		t.Skip("skipping integration test; set ICD10_INTEGRATION=1 to run")
	}

	codec := icd10.NewCodec()
	client := coding.NewHTTPClient()

	rows, err := client.Validate(codec, "E11.9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	row, _ := coding.ExactMatch(rows, "E11.9")
	if row == nil {
		t.Fatal("expected E11.9 to be found")
	}
	result := codec.Parse(row)
	if !result.Valid {
		t.Error("expected valid=true")
	}
	if result.Name == "" {
		t.Error("expected non-empty name")
	}
}

func TestIntegrationInvalidCode(t *testing.T) {
	if os.Getenv("ICD10_INTEGRATION") == "" {
		t.Skip("skipping integration test; set ICD10_INTEGRATION=1 to run")
	}

	codec := icd10.NewCodec()
	client := coding.NewHTTPClient()

	rows, err := client.Validate(codec, "Z99.99")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	row, _ := coding.ExactMatch(rows, "Z99.99")
	if row != nil {
		t.Error("expected Z99.99 to be not found")
	}
}
```

- [ ] **Step 2: Run integration tests**

```bash
ICD10_INTEGRATION=1 go test ./internal/icd10/... -run TestIntegration -v 2>&1
```

Expected: both tests PASS.

- [ ] **Step 3: Update README.md**

Add ICD-10-CM to the Features section:

```markdown
- **ICD-10-CM validation** — validate diagnosis codes (format, existence via NIH API); autocomplete by code or name; batch validation and CSV export
```

Add to Limitations:

```markdown
- **ICD-10-CM non-billable headers** — category codes like `A01` (Typhoid and paratyphoid fevers) are valid ICD-10-CM concepts but cannot be used for billing; the NIH API correctly returns them as "not found" on exact lookup. A production system would load the CMS tabular list locally to distinguish billable vs non-billable.
- **ICD-10-CM no similarity search** — unlike LOINC, ICD-10-CM has no check digit, so transposition-based "did you mean?" suggestions are not available. The autocomplete covers the discovery use case.
```

- [ ] **Step 4: Run all tests one final time**

```bash
go test ./... 2>&1
```

Expected: all packages PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/icd10/integration_test.go README.md
git commit -m "test: add ICD-10-CM integration tests; update README for multi-system"
```

---

## Self-Review

**Spec coverage:**
- [x] `Codec` interface in `internal/coding/codec.go` — Task 1
- [x] Shared NIH HTTP client in `internal/coding/client.go` — Task 1
- [x] LOINC codec wrapping existing logic — Task 2
- [x] ICD-10-CM format validation with U-prefix detection — Task 3
- [x] ICD-10-CM codec (no check digit, no similar candidates) — Task 4
- [x] Handlers refactored to accept `coding.Codec` — Task 5
- [x] Tests updated for new signatures — Task 6
- [x] Export updated to use `coding.Result` — Task 6
- [x] `main.go` dynamic route registration — Task 7
- [x] Tab bar generated from codec list — Task 8
- [x] System-scoped IDs (no collisions between tabs) — Task 8
- [x] Tab state preserved (both panels in DOM) — Task 8
- [x] LOINC extra fields (units/datatype/relatednames) preserved — Task 9
- [x] ICD-10-CM integration tests — Task 10
- [x] README updated — Task 10

**Type consistency:**
- `coding.Codec` interface defined in Task 1, implemented in Tasks 2+4, consumed in Tasks 5+7 ✓
- `coding.Result` defined in Task 1, returned by `Parse()` in Tasks 2+4, used in Tasks 5+6 ✓
- `coding.Suggestion` defined in Task 1, used in Tasks 5+8 ✓
- `coding.HTTPClient` defined in Task 1, instantiated in handlers in Task 5 ✓
- `loinc.ExtraResult` defined in Task 9, type-asserted in Task 9's validate handler ✓
- System-scoped IDs follow pattern `{systemID}-{element}` consistently in Tasks 8 ✓

**Note on Task 9:** The `ValidateWithExtras` type assertion approach keeps the handler mostly codec-agnostic while allowing LOINC to provide richer data. Future codecs that support extra fields can implement the same interface without changing handler code.
