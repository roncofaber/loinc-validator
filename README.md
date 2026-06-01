# Medical Code Validator

**Live:** https://loinc-validator.fly.dev &nbsp;|&nbsp; **Source:** https://github.com/roncofaber/loinc-validator

A multi-system medical code validator supporting LOINC and ICD-10-CM, built with Go and HTMX against the [NIH Clinical Tables API](https://clinicaltables.nlm.nih.gov/).

## Features

### LOINC (v2.82)
- **Single code validation** — enter a LOINC code and instantly see if it's valid, along with its name, short name, component, data type, units, and related terms
- **Check digit validation** — the Mod-10 check digit is verified locally before hitting the API; wrong check digits produce a specific correction suggestion (e.g. *"did you mean 2345-7?"*)
- **Deprecated code detection** — codes whose name begins with "Deprecated" are flagged with a warning (see Limitations for coverage details)
- **LP Part / LG Group detection** — LOINC Part (`LP…`) and Group (`LG…`) identifiers are explicitly rejected with a clear explanation
- **Batch validation** — upload a plain text or CSV file (one code per line) and get a full validation report; lines starting with `#` are treated as comments
- **CSV export** — download batch results as a timestamped CSV file with metadata (code, valid, deprecated, name, checked-at time)
- **Clear error handling** — empty input, malformed codes, wrong check digits, API errors, and no-match cases are all handled with specific, actionable messages

### ICD-10-CM (v2026)
- **Single code validation** — enter an ICD-10-CM diagnosis code and see if it exists in the NIH database
- **Format validation** — validates structure (letter + 2 digits + optional decimal + up to 4 alphanumeric characters); all letters A–Z are accepted, including U (U07.1 COVID-19, U09.9 long COVID are valid billable codes)
- **Autocomplete** — search by code or diagnosis name; works for both systems independently
- **Batch validation** — same file upload and CSV export as LOINC
- **Extensible architecture** — adding a third system (e.g. HCPCS) requires only one new codec file and one line in `main.go`

## Local Setup

**Prerequisites:** Go 1.22+ (developed with Go 1.26)

```bash
git clone https://github.com/roncofaber/loinc-validator
cd loinc-validator
go run .
# Open http://localhost:8080
```

## Running Tests

```bash
# Unit tests only
go test ./...

# Include LOINC integration tests against the real NIH API (requires internet)
LOINC_INTEGRATION=1 go test ./...

# Include ICD-10-CM integration tests
ICD10_INTEGRATION=1 go test ./...
```

## Docker

```bash
docker build -t loinc-validator .
docker run -p 8080:8080 loinc-validator
# Open http://localhost:8080
```

## Deployment (Fly.io)

```bash
# Install flyctl: https://fly.io/docs/hands-on/install-flyctl/
flyctl auth login
flyctl deploy
```

The app will be available at `https://loinc-validator.fly.dev` (or the name configured in `fly.toml`).

## Examples

The [`examples/`](examples/) directory contains ready-to-use test inputs:

| File | Description |
|------|-------------|
| [`loinc_batch_common_labs.txt`](examples/loinc_batch_common_labs.txt) | 20 most common lab codes — all active |
| [`loinc_batch_vital_signs.txt`](examples/loinc_batch_vital_signs.txt) | 8 vital sign codes — all active |
| [`loinc_batch_mixed_status.txt`](examples/loinc_batch_mixed_status.txt) | Mix of active, deprecated, discouraged, invalid, and malformed codes — exercises every result type |
| [`loinc_batch_large.txt`](examples/loinc_batch_large.txt) | 500 active codes across 19 clinical domains — good for testing batch performance and CSV export |

See [`examples/README.md`](examples/README.md) for individual codes to try in the single-code validator.

## Project Structure

```
├── main.go                        # HTTP server, dynamic route registration per codec
├── internal/
│   ├── coding/
│   │   ├── codec.go               # Codec interface + shared Result/Suggestion types
│   │   └── client.go              # ExactMatch helper (shared response utility)
│   ├── loinc/
│   │   ├── codec.go               # LOINC Codec implementation
│   │   ├── client.go              # LOINC API wrapper (extra fields: units, datatype, etc.)
│   │   ├── format.go              # LOINC format validation + Mod-10 check digit
│   │   ├── client_test.go         # Unit tests (mocked HTTP)
│   │   ├── format_test.go         # Table-driven format tests
│   │   └── integration_test.go    # Real API smoke tests
│   ├── icd10/
│   │   ├── codec.go               # ICD-10-CM Codec implementation
│   │   ├── format.go              # ICD-10-CM format validation
│   │   ├── format_test.go         # Table-driven format tests
│   │   ├── codec_test.go          # Unit tests
│   │   └── integration_test.go    # Real API smoke tests
│   └── handlers/
│       ├── templates.go           # Template loader (base + partials + system tabs)
│       ├── validate.go            # POST /{system}/validate
│       ├── batch.go               # POST /{system}/batch
│       ├── suggest.go             # GET /{system}/suggest (autocomplete)
│       ├── similar.go             # GET /{system}/suggest-similar
│       └── export.go              # POST /export (system-agnostic)
├── templates/
│   ├── index.html                 # Tab bar + panels
│   ├── loinc/tab.html             # LOINC form content
│   ├── icd10/tab.html             # ICD-10-CM form content
│   └── partials/
│       ├── result.html            # Single-code result fragment (shared)
│       ├── batch_result.html      # Batch result table (shared)
│       ├── suggest.html           # Autocomplete dropdown (shared)
│       └── similar.html          # "Did you mean?" suggestions (shared)
├── static/style.css               # All styles via CSS variables — no inline styles
└── examples/                      # Ready-to-use test files for both systems
```

## Strengths

- **Single binary, no runtime dependencies** — the entire app compiles to one ~8MB static binary. Deployment is trivial.
- **No frontend build step** — HTMX is loaded from CDN. Changing the UI means editing HTML templates only, with no npm, no bundler, no build step.
- **Concurrent batch processing** — up to 10 codes are validated in parallel, bounded to be respectful of the NIH API.
- **Stateless server** — no database required; batch results are embedded as JSON in the HTML response and passed to the export endpoint by the browser.
- **Clean separation of concerns** — the NIH API wrapper, format validator, and HTTP handlers are independent units with well-defined interfaces and their own tests.

## Limitations

- **Rate limiting** — the NIH API has no documented rate limit, but very large batches (1000+ codes) may be slow or throttled. The 10-worker cap is a courtesy measure, not a guarantee.
- **No persistence** — results exist only within the browser session. Closing the tab loses the batch results (though the CSV can be exported first).
- **Format regex is a shape filter only** — the regex `^\d+-\d$` validates only the structural shape of an observation code (one or more digits, a dash, exactly one check digit). It intentionally imposes no upper length bound on the numeric prefix, since LOINC codes are assigned sequentially and will eventually exceed any fixed cap. Real validation is done by the check digit (catches typos) and the API lookup (catches non-existent codes).
- **No E2E tests** — browser-level testing (loading states, HTMX fragment swaps) is not covered. This is a known gap.
- **JSON in hidden form field** — batch results are passed to `/export` as a JSON-encoded hidden field. For very large batches this field grows proportionally; a session store would be the production solution.
- **Deprecated code detection is heuristic** — the NIH API does not expose a `STATUS` field, so we cannot distinguish `ACTIVE`, `DEPRECATED`, `DISCOURAGED`, or `TRIAL` codes server-side. As a partial mitigation, codes whose name begins with `"Deprecated "` are flagged with a warning in the UI and CSV — this covers the majority of deprecated codes but not all (LOINC 2.82 has ~589 deprecated codes without this prefix, and ~1,378 discouraged codes which are not flagged at all). A production-grade solution would bundle the LOINC release table locally and cross-reference `STATUS` at validation time, at the cost of periodic table updates when new LOINC releases are published (approximately twice per year).
- **LOINC version is hardcoded** — the NIH Clinical Tables API does not expose the LOINC release version it serves in any machine-readable way (no response header, no dedicated endpoint). The version shown in the footer (`loincVersion` constant in `main.go`) must be updated manually when LOINC publishes a new release. The API docs page uses a JavaScript function `showVersion()` to display the version client-side, but the underlying endpoint is not publicly documented or stable.
- **ICD-10-CM non-billable category headers** — codes like `A01` (Typhoid and paratyphoid fevers) are valid ICD-10-CM concepts used to organise the tabular list but cannot be used for billing. The NIH API correctly returns them as "not found" on exact lookup. A production system would load the CMS tabular list locally to distinguish billable vs non-billable codes explicitly.
- **ICD-10-CM retired codes** — the API does not expose whether a code has been retired in a prior year's release. Codes entered by users that were valid in older releases may return as "not found" without explanation.
- **ICD-10-CM similarity search is prefix-based only** — unlike LOINC, ICD-10-CM has no check digit, so transposition-based suggestions are not possible. Instead, the validator tries progressively shorter prefix variants (e.g. `E11.99` → `E11.9`, `E11`) to surface near-matches. The autocomplete complements this for name-based discovery.
- **No replacement code suggestions for deprecated codes** — the LOINC release ships a `MapTo.csv` file mapping each deprecated code to its active replacement (with a comment indicating context when multiple replacements exist, e.g. Ordinal vs Quantitative). The NIH API does not expose this mapping. A future improvement would load `MapTo.csv` at startup and surface the replacement code and name alongside the deprecation warning, saving users the manual lookup.

## Technical Choices

Go's standard library + HTMX is a strong fit for this use case: the app is primarily a thin layer between a user form and an external API. The HTMX model (server returns HTML fragments, not JSON) eliminates the need for a separate frontend API contract, a build pipeline, and client-side state management.

Using only the Go standard library (no web framework) keeps the dependency surface minimal and the code easy to audit — relevant for a healthcare-adjacent tool where understanding every layer of the stack matters. The single-binary deployment model also simplifies operations significantly.

The main trade-off of this stack is an interactivity ceiling: if the app were to grow into a rich dashboard with client-side filtering, sorting, or real-time updates, HTMX would need to be supplemented or replaced. For this scope, it is the right choice.

### Extensible multi-system architecture

The app is built around a `coding.Codec` interface (`internal/coding/codec.go`) that every medical coding system implements. All HTTP handlers, routing, and UI are system-agnostic — they operate on the interface, not on any specific system. Adding a new coding system (e.g. HCPCS, RxNorm, HPO) requires exactly three changes:

1. `internal/<system>/codec.go` — implement the `Codec` interface
2. `templates/<system>/tab.html` — the form UI for that system
3. One line in `main.go` — add the codec to the slice

**ICD-10-CM is the proof of concept** for this design. It was added without modifying any handler, routing logic, shared template, or CSS.

### Per-system HTTP clients

Each coding system owns its own HTTP client within its package (`loinc/codec.go`, `icd10/codec.go`). This is intentional — the `ef=` (extra fields) parameter is a general NIH Clinical Tables API feature available on all endpoints, but each system exposes different fields. LOINC uses it to fetch units of measure, data type, and synonym terms. ICD-10-CM only exposes `code` and `name` with no additional fields available via `ef=`. Future systems may expose their own extra fields.

By keeping the HTTP logic in each codec, every system is fully self-contained and can evolve independently. The shared `internal/coding/client.go` contains only `ExactMatch` — a small utility for finding an exact-match row in an API response array.
