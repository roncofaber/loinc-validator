# Submission notes

Built as a submission for the **Swiss Data Science Center (SDSC)** Software R&D Engineer task on a LOINC code validator. The core ask was a web app to validate a single LOINC code against the [NIH Clinical Tables API](https://clinicaltables.nlm.nih.gov/) with clear feedback and edge-case handling; everything beyond that was an optional extension.

App overview, setup, and architecture live in the [README](README.md). This file covers the task-specific reflection.

## Scope

**Core requirement:** enter a LOINC code, validate it against the NIH API, and show valid/invalid, with handling for empty input, malformed codes, API errors, and no-result responses.

**Optional extensions implemented**: batch file upload with per-row results, additional metadata for valid codes, timestamped CSV export, app deployment (Fly.io), similar-code suggestions, and support for multiple coding systems (ICD-10-CM, alongside LOINC).

## Technical choices

Go stdlib + HTMX: the app is a thin layer between a form and an external API. HTMX eliminates a frontend build pipeline and JSON API contract. No web framework keeps the dependency surface minimal.

The `coding.Codec` interface keeps handlers independent of the underlying system. Under the hood, each system manages its own API calls. LOINC needs a dedicated client because it fetches extra fields like units and synonyms via the `ef=` parameter, while ICD-10-CM only returns code and name so it can share the generic `coding.Search()`. The result is that adding HCPCS, RxNorm, or any other NIH Clinical Tables endpoint is straightforward: implement the interface, add a tab template, and register it in `main.go`.

## Assumptions & limitations

The main assumptions and known gaps (deprecated-status detection via name prefix, no MapTo mapping, hardcoded LOINC version, batch results carried in a hidden field) are documented in [README → Limitations](README.md#limitations).

## What I would pursue with more time

In rough priority order:

- **MapTo replacement suggestions** for deprecated LOINC codes, using the `MapTo.csv` from the LOINC release (the API does not expose this).
- **Server-side store for batch results** instead of the hidden JSON field, so very large batch files export reliably.
- **Automated LOINC version detection** to remove the hardcoded version constant.
- **More coding systems** (RxNorm, HCPCS) — each is roughly one new `Codec` implementation plus a tab template, thanks to the interface.
- **Per-system result templates** to replace the conditional guards in the shared `partials/result.html`.
- **CI** (GitHub Actions running `go vet` and `go test`) to guard regressions on every push.

## Development & AI assistance

I was new to Go, so this was a fun project to learn how the language and the HTMX stack work in practice. I worked in a spec-driven, AI-assisted workflow: I made the design and architecture decisions (the `Codec` interface, stateless server, dynamic route/tab generation) and wrote the specs and step-by-step plans first; an AI coding assistant (Claude Code) then helped generate the implementation under that direction, and I reviewed every change and verified behaviour against the live NIH API and the test suite. The planning trail is preserved under [`docs/`](docs/).

This reflects how I find AI most useful: it lets me stay in the data-architect role — using my software development skills to focus on code structure, architectural decisions, and trade-offs — while developing quickly and fluently in languages I am not necessarily proficient in.
