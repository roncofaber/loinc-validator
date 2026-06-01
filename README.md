# Medical Code Validator

**Live:** https://loinc-validator.fly.dev &nbsp;|&nbsp; **Source:** https://github.com/roncofaber/loinc-validator

Validates medical codes against the [NIH Clinical Tables API](https://clinicaltables.nlm.nih.gov/), built with Go and HTMX. Currently supports **LOINC v2.82** and **ICD-10-CM v2026**.

> Built as a submission for the **Swiss Data Science Center (SDSC)** Software R&D Engineer task on a LOINC code validator. The core ask was a web app to validate a single LOINC code against the NIH API. For the task-specific write-up — scope, what was built in the timeframe, development process and AI assistance, and future work — see [`SUBMISSION.md`](SUBMISSION.md).

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

## Deployment (Fly.io)

```bash
flyctl auth login
flyctl deploy
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

- The app is a **single static binary** with no runtime dependencies and no frontend build step.
- The **modular structure** of each coding system is self-contained with its own client, format validator, and codec. This makes extending the app to support other NIH APIs trivial. The ICD-10-CM integration on top of LOINC is an example.
- For **batch processing**, up to 10 concurrent API calls are made (the limit is set to avoid overloading the NIH API). This makes batch file analysis faster.
- **Stateless server**: no database required; batch results travel directly through the browser for export (which is also a scaling limitation — see [Limitations](#limitations)).

## Limitations

- **Deprecated/discouraged status**: for LOINC, the NIH API does not expose `STATUS`. Deprecated codes are detected via name prefix ("Deprecated "). This covers most but not all cases. Discouraged and TRIAL codes are not properly flagged.
- **No MapTo suggestions**: `MapTo.csv` in the LOINC release maps deprecated codes to active replacements; the API does not expose this mapping.
- **LOINC version hardcoded**: the API exposes no machine-readable version; `loincVersion` in `internal/loinc/codec.go` must be updated manually with each LOINC release.
- **Batch results in hidden field**: the JSON payload grows with batch size; a session store would be the production solution for very large files.

## Technical Choices

See [`SUBMISSION.md` → Technical choices](SUBMISSION.md#technical-choices) for the rationale behind the stack and the `Codec` architecture.
