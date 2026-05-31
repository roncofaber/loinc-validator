# LOINC Code Validator Web App — Design Spec

**Date:** 2026-05-30
**Stack:** Go (stdlib) + HTMX
**Task:** RSDE 2026 Task 4

---

## Overview

A single-binary Go web application that allows users to validate LOINC codes against the NIH Clinical Tables API. Supports single-code validation, batch file upload, and CSV export of results with metadata. No frontend build step — Go's `html/template` renders pages server-side, HTMX handles dynamic UI updates via HTML fragment swapping.

---

## Architecture

```
Browser (HTMX)
    │  POST /validate   → returns HTML fragment
    │  POST /batch      → returns HTML table fragment
    │  POST /export     → returns CSV file download
    ▼
Go HTTP Server (net/http)
    ├── handlers/validate.go   single-code validation
    ├── handlers/batch.go      file upload + multi-code validation
    ├── handlers/export.go     CSV generation with metadata
    └── loinc/client.go        NIH API wrapper
         │
         ▼
    NIH LOINC API (clinicaltables.nlm.nih.gov)
```

The server is **stateless** — no database, no sessions. Batch results are passed back to the browser as embedded JSON in the HTML response, so the export endpoint receives and re-formats them without re-querying the API.

### Project Layout

```
loinc-validator/
├── main.go
├── go.mod
├── Dockerfile
├── fly.toml
├── README.md
├── internal/
│   ├── loinc/client.go
│   └── handlers/
│       ├── validate.go
│       ├── batch.go
│       └── export.go
└── templates/
    ├── index.html
    └── partials/
        ├── result.html
        └── batch_result.html
```

---

## Components & Data Flow

### `internal/loinc/client.go` — NIH API wrapper

Calls the NIH LOINC search endpoint with exact `LOINC_NUM` field search:

```
GET https://clinicaltables.nlm.nih.gov/api/loinc/v3/search
    ?terms=<code>&sf=LOINC_NUM&df=LOINC_NUM,LONG_COMMON_NAME,VersionLastChanged
```

A code is **valid** if the API returns at least one result whose `LOINC_NUM` matches the input exactly (case-insensitive). Returns a `LOINCResult` struct:

```go
type LOINCResult struct {
    Code      string
    Name      string
    Version   string
    Valid      bool
    CheckedAt time.Time
    Error     string
}
```

### `internal/handlers/validate.go` — single code handler

- Accepts `POST /validate` with form field `code`
- Validates format with regex `^\d{1,5}-\d$` before hitting the API
- Returns `partials/result.html` fragment (HTMX swaps into `#result`)

### `internal/handlers/batch.go` — file upload handler

- Accepts `POST /batch` with multipart file upload (CSV or TXT, one code per line)
- Validates each code concurrently with a bounded goroutine pool (max 10 in-flight)
- Returns `partials/batch_result.html` — a table with summary + per-code rows
- Results also embedded as hidden JSON so `/export` can consume them

### `internal/handlers/export.go` — CSV download

- Accepts `POST /export` with JSON result payload from the batch result page
- Returns `text/csv` with header `Content-Disposition: attachment; filename=loinc_validation_<timestamp>.csv`
- CSV columns: `Code, Valid, Name, Version, CheckedAt, LoincVersion`

### `templates/index.html`

Single page with two sections:
- **Single validation**: text input + submit button; result swapped into `#result` div via HTMX
- **Batch validation**: file input + submit button; table swapped into `#batch-result` div; export button appears after results load

---

## Error Handling

| Case | Detection point | User feedback |
|------|----------------|---------------|
| Empty input | Handler, pre-API | "Please enter a LOINC code" |
| Malformed format | Regex, pre-API | "Invalid format — expected e.g. 2345-7" |
| API error (network/timeout) | HTTP client error | "Could not reach the LOINC API — please try again" |
| Code not found | 0 results or no exact match | "Code XXXXX-X is not a valid LOINC code" |
| Valid code | Exact match | Green card with name, version, checked-at time |

For batch: each code gets its own row status. A summary line shows e.g. "12 valid, 3 invalid, 1 error" at the top. No silent failures — every code in the input file appears in the output table.

---

## Testing

- **Unit tests** for `loinc/client.go`: mock HTTP transport via `net/http/httptest`, covering all four error cases plus malformed JSON response
- **Unit tests** for format validation: table-driven tests covering `""`, `"abc"`, `"12345-6"`, `"123456-7"`, `"1-1"`, `"2345-7"`
- **Integration smoke test**: one real API call to known-valid (`2345-7` — heart rate) and one known-invalid code; skipped unless `LOINC_INTEGRATION=1` env var is set
- **No browser/E2E tests**: out of scope for time budget — noted in README as a future improvement

---

## Deployment

Multi-stage Dockerfile produces a minimal (~10MB) static binary image. Deployed to Fly.io free tier via `flyctl deploy`. `fly.toml` configures the app name, region, and port.

---

## Extensions Planned (time permitting)

| Priority | Feature |
|----------|---------|
| A (Small) | Display additional info for valid codes: name and LOINC version shown in core response; related/synonymous terms fetched via additional `df` fields in the same API call |
| B (Small) | Upload a file with a list of LOINC codes, output a validation report |
| C (Small) | Export validation results with metadata (timestamp, LOINC version) |
| D (Medium) | Deploy to Fly.io |

---

## Limitations & Assumptions

- **Rate limiting**: the NIH API has no documented rate limit, but the batch handler caps concurrent requests at 10 to be respectful. Very large files (>1000 codes) may be slow.
- **Format validation**: LOINC codes follow a `NNNNN-N` pattern but the spec allows 1–5 leading digits. The regex `^\d{1,5}-\d$` covers the documented range; edge cases outside this may exist.
- **No persistence**: results exist only in the browser session. The export CSV is generated on demand from the batch result payload.
- **HTTPS**: Fly.io provides TLS termination automatically; local dev runs on plain HTTP.
- **No authentication**: the NIH API is public and requires no API key.

---

## Reflection (for README)

Go's standard library + HTMX is a strong fit for this use case: the app is primarily a thin layer between a user form and an external API. The HTMX model (server returns HTML fragments) eliminates the need for a JSON API contract, a frontend build pipeline, and client-side state management. The result is a single deployable binary with straightforward, readable code — appropriate for a healthcare-adjacent tool where auditability and maintainability matter more than UI sophistication.

The main limitation of this stack is interactivity ceiling: if the app were to grow into a rich dashboard with client-side filtering, sorting, or charting, HTMX would need to be supplemented or replaced. For this scope, it is the right choice.
