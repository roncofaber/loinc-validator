# LOINC Code Validator

A web application for validating LOINC codes against the [NIH Clinical Tables API](https://clinicaltables.nlm.nih.gov/apidoc/loinc/v3/doc.html), built with Go and HTMX.

## Features

- **Single code validation** — enter a LOINC code and instantly see if it's valid, along with its name and version
- **Batch validation** — upload a plain text or CSV file (one code per line) and get a full validation report
- **CSV export** — download batch results as a timestamped CSV file with metadata (code, validity, name, version, checked-at time)
- **Clear error handling** — empty input, malformed codes, API errors, and no-match cases are all handled with user-friendly messages

## Local Setup

**Prerequisites:** Go 1.23+

```bash
git clone <repo-url>
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
- **LOINC format assumption** — the format regex `^\d{1,5}-\d$` covers documented LOINC patterns; edge cases outside this range may exist in practice.
- **No E2E tests** — browser-level testing (loading states, HTMX fragment swaps) is not covered. This is a known gap.
- **JSON in hidden form field** — batch results are passed to `/export` as a JSON-encoded hidden field. For very large batches this field grows proportionally; a session store would be the production solution.

## Technical Choices

Go's standard library + HTMX is a strong fit for this use case: the app is primarily a thin layer between a user form and an external API. The HTMX model (server returns HTML fragments, not JSON) eliminates the need for a separate frontend API contract, a build pipeline, and client-side state management.

Using only the Go standard library (no web framework) keeps the dependency surface minimal and the code easy to audit — relevant for a healthcare-adjacent tool where understanding every layer of the stack matters. The single-binary deployment model also simplifies operations significantly.

The main trade-off of this stack is an interactivity ceiling: if the app were to grow into a rich dashboard with client-side filtering, sorting, or real-time updates, HTMX would need to be supplemented or replaced. For this scope, it is the right choice.
