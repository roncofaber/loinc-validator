# Medical Code Validator

**Live:** https://loinc-validator.fly.dev &nbsp;|&nbsp; **Source:** https://github.com/roncofaber/loinc-validator

Validates medical codes against the [NIH Clinical Tables API](https://clinicaltables.nlm.nih.gov/), built with Go and HTMX. Currently supports **LOINC v2.82** and **ICD-10-CM v2026**.

## Features

- **Single code validation**: displays name and metadata; system-specific details (LOINC: component, data type, units, related terms; ICD-10-CM: diagnosis name)
- **Batch validation**: upload a plain text or CSV file (one code per line, `#` comments supported); per-row status in results table
- **Format validation**: structural checks before hitting the API (LOINC Mod-10 check digit with correction suggestion; ICD-10-CM spec compliance)
- **Similar code suggestions**: LOINC: adjacent-digit transpositions with recomputed check digit; ICD-10-CM: prefix truncation, decimal expansion, and missing decimal detection 
- **Live autocomplete**: search by code or name for both systems
- **Deprecated code detection**: LOINC codes flagged via name prefix; LP Part / LG Group identifiers explicitly rejected with clear explanation
- **CSV export**: timestamped file with code, valid, deprecated, name, checked-at, error columns
- **Extensible**: adding a third system (e.g. HCPCS, RxNorm) requires one codec file, one template, one line in `main.go`

## Setup

**Prerequisites:** Go 1.22+ (developed with Go 1.26)

```bash
git clone https://github.com/roncofaber/loinc-validator
cd loinc-validator
go run .
# Open http://localhost:8080
```

## Tests

```bash
go test ./...                     # unit tests
LOINC_INTEGRATION=1 go test ./... # real LOINC API
ICD10_INTEGRATION=1 go test ./... # real ICD-10-CM API
```

## Docker

```bash
docker build -t loinc-validator .
docker run -p 8080:8080 loinc-validator
```

## Examples

See [`examples/`](examples/) and [`examples/README.md`](examples/README.md) for ready-to-use batch files for both systems.

## Project Structure

```
├── main.go
├── internal/
│   ├── coding/
│   │   ├── codec.go      # Codec interface + Result/Suggestion types
│   │   └── utils.go      # Search(), ExactMatch(), NewHTTPClient()
│   ├── loinc/
│   │   ├── codec.go      # LOINC Codec
│   │   ├── client.go     # LOINC API client (ef= extra fields)
│   │   └── format.go     # Format validation + Mod-10 check digit
│   ├── icd10/
│   │   ├── codec.go      # ICD-10-CM Codec
│   │   ├── client.go     # Delegates to coding.Search()
│   │   └── format.go     # ICD-10-CM format validation
│   └── handlers/         # validate, batch, suggest, similar, export
├── templates/
│   ├── index.html        # Tab bar + panels
│   ├── icons.html        # Named SVG templates
│   ├── loinc/tab.html
│   ├── icd10/tab.html
│   └── partials/         # result, batch_result, suggest, similar
└── static/style.css      # CSS variables
```

## Strengths

- **Single static binary** — no runtime dependencies, no frontend build step; HTMX loaded from CDN.
- **Three-layer LOINC validation** — regex shape → Mod-10 check digit (local) → API existence. Each layer fails with a specific, actionable message.
- **Codec interface** — handlers are fully system-agnostic; each coding system is self-contained with its own client, format validator, and codec.
- **Concurrent batch processing** — up to 10 concurrent API calls, bounded to be respectful of the NIH API.
- **Stateless server** — no database required; batch results travel through the browser for export.
- **Go test conventions** — unit tests co-located with code, mocked HTTP transports, integration tests gated by environment variables.

## Limitations

- **Deprecated/discouraged status** — the NIH API does not expose `STATUS`. Deprecated codes are detected heuristically via name prefix ("Deprecated ") — covers most but not all cases. Discouraged and TRIAL codes are not flagged.
- **No MapTo suggestions** — `MapTo.csv` in the LOINC release maps deprecated codes to active replacements; the API does not expose this mapping.
- **LOINC version hardcoded** — the API exposes no machine-readable version; `loincVersion` in `main.go` must be updated manually with each LOINC release (~twice per year).
- **ICD-10-CM non-billable headers** — category codes (e.g. `A01`) return "not found" from the API; the similar suggestions machinery surfaces billable children as alternatives.
- **ICD-10-CM retired codes** — the API does not indicate whether a code has been retired in a prior release.
- **Batch results in hidden field** — the JSON payload grows with batch size; a session store would be the production solution for very large files.
- **No E2E tests** — HTMX fragment swaps and browser interactions are not covered.

## Technical Choices

Go stdlib + HTMX: the app is a thin layer between a form and an external API. HTMX eliminates a frontend build pipeline and JSON API contract. No web framework keeps the dependency surface minimal and the code auditable — relevant for a healthcare-adjacent tool.

The `coding.Codec` interface decouples systems from handlers. Each system owns its HTTP client — LOINC uses `ef=` for rich extra fields; ICD-10-CM delegates to the shared `coding.Search()` since its API exposes only code and name. Adding HCPCS or RxNorm follows the same pattern: one codec, one template, one registration.

**Known improvement:** per-system result templates (e.g. `loinc/result.html`) would replace the conditional guards in the shared `partials/result.html` as more systems with different optional fields are added.
