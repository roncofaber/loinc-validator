# LOINC Code Validator

**Live:** https://loinc-validator.fly.dev &nbsp;|&nbsp; **Source:** https://github.com/roncofaber/loinc-validator

A web application for validating LOINC codes against the [NIH Clinical Tables API](https://clinicaltables.nlm.nih.gov/apidoc/loinc/v3/doc.html), built with Go and HTMX.

## Features

- **Single code validation** — enter a LOINC code and instantly see if it's valid, along with its name, short name, component, data type, units, and related terms
- **Check digit validation** — the Mod-10 check digit is verified locally before hitting the API; wrong check digits produce a specific correction suggestion (e.g. *"did you mean 2345-7?"*)
- **Deprecated code detection** — codes whose name begins with "Deprecated" are flagged with a warning (see Limitations for coverage details)
- **LP Part / LG Group detection** — LOINC Part (`LP…`) and Group (`LG…`) identifiers are explicitly rejected with a clear explanation
- **Batch validation** — upload a plain text or CSV file (one code per line) and get a full validation report; lines starting with `#` are treated as comments
- **CSV export** — download batch results as a timestamped CSV file with metadata (code, valid, deprecated, name, checked-at time)
- **Clear error handling** — empty input, malformed codes, wrong check digits, API errors, and no-match cases are all handled with specific, actionable messages

## Local Setup

**Prerequisites:** Go 1.23+

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

# Include integration tests against the real NIH API (requires internet)
LOINC_INTEGRATION=1 go test ./...
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
| [`batch_common_labs.txt`](examples/batch_common_labs.txt) | 20 most common lab codes — all active |
| [`batch_vital_signs.txt`](examples/batch_vital_signs.txt) | 8 vital sign codes — all active |
| [`batch_mixed_status.txt`](examples/batch_mixed_status.txt) | Mix of active, deprecated, discouraged, invalid, and malformed codes — exercises every result type |
| [`batch_large.txt`](examples/batch_large.txt) | 500 active codes across 19 clinical domains — good for testing batch performance and CSV export |

See [`examples/README.md`](examples/README.md) for individual codes to try in the single-code validator.

## Project Structure

```
├── main.go                        # HTTP server, route registration
├── internal/
│   ├── loinc/
│   │   ├── client.go              # NIH API wrapper
│   │   ├── format.go              # LOINC code format validation
│   │   ├── client_test.go         # Unit tests (mocked HTTP)
│   │   ├── format_test.go         # Table-driven format tests
│   │   └── integration_test.go    # Real API smoke tests
│   └── handlers/
│       ├── templates.go           # Template loader
│       ├── validate.go            # POST /validate
│       ├── batch.go               # POST /batch
│       └── export.go              # POST /export
└── templates/
    ├── index.html                 # Main page
    └── partials/
        ├── result.html            # Single-code result fragment
        └── batch_result.html      # Batch result table fragment
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
- **No replacement code suggestions for deprecated codes** — the LOINC release ships a `MapTo.csv` file mapping each deprecated code to its active replacement (with a comment indicating context when multiple replacements exist, e.g. Ordinal vs Quantitative). The NIH API does not expose this mapping. A future improvement would load `MapTo.csv` at startup and surface the replacement code and name alongside the deprecation warning, saving users the manual lookup.

## Technical Choices

Go's standard library + HTMX is a strong fit for this use case: the app is primarily a thin layer between a user form and an external API. The HTMX model (server returns HTML fragments, not JSON) eliminates the need for a separate frontend API contract, a build pipeline, and client-side state management.

Using only the Go standard library (no web framework) keeps the dependency surface minimal and the code easy to audit — relevant for a healthcare-adjacent tool where understanding every layer of the stack matters. The single-binary deployment model also simplifies operations significantly.

The main trade-off of this stack is an interactivity ceiling: if the app were to grow into a rich dashboard with client-side filtering, sorting, or real-time updates, HTMX would need to be supplemented or replaced. For this scope, it is the right choice.
