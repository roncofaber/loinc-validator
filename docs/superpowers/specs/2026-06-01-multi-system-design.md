# Multi-System Medical Code Validator — Design Spec

**Date:** 2026-06-01
**Builds on:** `docs/superpowers/specs/2026-05-30-loinc-validator-design.md`

---

## Goal

Extend the LOINC validator into a multi-system medical coding validator, starting with ICD-10-CM as the second system. The architecture must allow future systems (HCPCS, RxNorm, HPO, etc.) to be added by implementing a single interface and registering one entry in `main.go` — with zero changes to handlers, templates, or routing logic.

---

## Architecture

### Codec Interface

All coding systems implement a single `Codec` interface defined in `internal/coding/codec.go`:

```go
type Result struct {
    Code      string
    Name      string
    Valid      bool
    Deprecated bool
    CheckedAt time.Time
    Error     string
    // Optional fields — populated by codecs that support them
    ShortName    string
    Component    string
    RelatedNames string
    DataType     string
    Units        []string
}

type Suggestion struct {
    Code string
    Name string
}

type Codec interface {
    Name()                          string   // Display name: "LOINC", "ICD-10-CM"
    SystemID()                      string   // Route prefix: "loinc", "icd10"
    Version()                       string   // e.g. "2.82", "2026"
    SearchURL()                     string   // NIH API base URL
    SearchFields()                  string   // sf= param
    DisplayFields()                 string   // df= param
    ValidateFormat(code string)     error    // nil if system has no format constraints
    Parse(fields []string)          Result   // maps API display fields → Result
    SimilarCandidates(code string)  []string // transpositions or nil if not applicable
}
```

### Shared HTTP Client

`internal/coding/client.go` handles the NIH API HTTP call and response parsing. It is codec-agnostic — it takes the search URL, fields, and query term, returns `[][]string` display rows. All systems use identical HTTP logic.

### Package Structure

```
internal/
  coding/
    codec.go        # Codec interface + Result + Suggestion types
    client.go       # shared NIH HTTP client
  loinc/
    codec.go        # Codec implementation (wraps existing logic)
    format.go       # format validation + check digit (unchanged)
    client.go       # LOINC-specific API fields + Parse (moved from current client.go)
  icd10/
    codec.go        # Codec implementation
    format.go       # ICD-10-CM format validation
  handlers/
    validate.go     # system-agnostic, takes Codec
    batch.go        # system-agnostic, takes Codec
    suggest.go      # system-agnostic, takes Codec
    similar.go      # system-agnostic, delegates to Codec.SimilarCandidates()
    export.go       # unchanged
    templates.go    # unchanged
```

### Route Registration

`main.go` ranges over a slice of codecs — adding a new system requires zero handler changes:

```go
codecs := []coding.Codec{
    loinc.NewCodec(),
    icd10.NewCodec(),
}

for _, codec := range codecs {
    id := codec.SystemID()
    mux.Handle("/"+id+"/validate",        handlers.NewValidateHandler(tmpl, codec))
    mux.Handle("/"+id+"/suggest",         handlers.NewSuggestHandler(tmpl, codec))
    mux.Handle("/"+id+"/suggest-similar", handlers.NewSimilarHandler(tmpl, codec))
    mux.Handle("/"+id+"/batch",           handlers.NewBatchHandler(tmpl, codec))
}
mux.Handle("/export", handlers.NewExportHandler(tmpl))
mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    tmpl.ExecuteTemplate(w, "index.html", map[string]any{
        "Codecs": codecs,
    })
})
```

---

## UI: Tab Layout

The page has a tab bar generated from the codec list — no hardcoded system names in HTML. Both tab panels are rendered server-side on page load and toggled via CSS class (not HTMX swaps), preserving results when switching tabs.

```html
<!-- Tab bar — generated from Codecs -->
<div class="tab-bar">
  {{range .Codecs}}
  <button class="tab-btn" data-tab="{{.SystemID}}" onclick="switchTab('{{.SystemID}}')">
    {{.Name}}
  </button>
  {{end}}
</div>

<!-- Tab panels — both in DOM, toggled by CSS -->
{{range .Codecs}}
<div id="tab-{{.SystemID}}" class="tab-panel">
  {{template (printf "%s/tab.html" .SystemID) .}}
</div>
{{end}}
```

Tab switching is a 5-line JS function — no HTMX, no network request, instant.

### Template Structure

```
templates/
  index.html                  # tab bar + panels
  partials/
    result.html               # shared result card (all systems)
    batch_result.html         # shared batch table (all systems)
    similar.html              # shared "Did you mean?" (all systems)
    suggest.html              # shared autocomplete dropdown (all systems)
  loinc/
    tab.html                  # LOINC form: placeholder "e.g. 2345-7", routes /loinc/*
  icd10/
    tab.html                  # ICD-10-CM form: placeholder "e.g. E11.9", routes /icd10/*
```

HTMX targets (`#result`, `#batch-result`, `#suggestions`) are scoped inside each tab panel with system-prefixed IDs (`#loinc-result`, `#icd10-result`) to avoid collisions.

---

## ICD-10-CM Codec

### Format Validation

Follows the official ICD-10-CM specification:
- Character 1: alpha, `U` excluded (reserved for ICD-11 WHO extensions)
- Character 2: numeric
- Character 3: alpha or numeric
- Characters 4-7: alpha or numeric, preceded by a decimal point after position 3
- Total length: 3-7 characters
- Case-insensitive (uppercased before matching)
- `X` is a valid placeholder character (dummy placeholder per spec)

```go
var icd10Pattern = regexp.MustCompile(`^[A-TV-Z]\d[A-Z0-9](\.[A-Z0-9]{1,4})?$`)

func ValidateFormat(code string) error {
    code = strings.TrimSpace(strings.ToUpper(code))
    if code == "" {
        return fmt.Errorf("ICD-10-CM code cannot be empty")
    }
    if !icd10Pattern.MatchString(code) {
        return fmt.Errorf("invalid ICD-10-CM format %q: expected a letter (not U), a digit, an alphanumeric character, then optionally a decimal and up to 4 alphanumeric characters (e.g. E11.9, S00.00XA)", code)
    }
    return nil
}
```

### No Check Digit, No Similar Candidates

ICD-10-CM has no check digit. `SimilarCandidates` returns `nil`. The `similar.go` handler already handles this gracefully (early return when no candidates). No "Did you mean?" is shown for ICD-10-CM not-found results.

### Parse

ICD-10-CM API returns `[code, name]` only. The `Parse` method populates `Code` and `Name`. `ShortName`, `Component`, `DataType`, `Units`, and `RelatedNames` are empty — the advanced details section renders nothing.

### Deprecation / Retired Codes

ICD-10-CM codes are revised annually by CMS. The API does not expose a status field, so retired codes cannot be detected at runtime — identical limitation to LOINC's discouraged codes. This is documented in the README. The NIH API data version is "2026", so codes in the API are current for 2026; codes entered by users that predate 2026 may be retired without warning.

### API Fields

- `SearchURL`: `https://clinicaltables.nlm.nih.gov/api/icd10cm/v3/search`
- `SearchFields` (sf): `code,name`
- `DisplayFields` (df): `code,name`

---

## Migration from Current LOINC-Only Structure

The existing `internal/loinc/` package is refactored minimally:
- `client.go` → split: HTTP logic moves to `internal/coding/client.go`, LOINC-specific fields/parse logic stays in `internal/loinc/codec.go`
- `format.go` → stays in `internal/loinc/` unchanged
- All existing tests pass without modification
- Existing routes (`/validate`, `/batch`, etc.) are replaced by `/loinc/validate`, `/loinc/batch`, etc.

**Breaking change:** existing URLs change. The live Fly.io deployment will need redeployment. Any bookmarks to `/validate` will 404 — acceptable since this is a demo app with no external consumers.

---

## Limitations

- **Retired ICD-10-CM codes** — the API does not expose a status/retired field. Codes retired in prior years may still return as valid if the underlying API data includes them, or may return as not found if they have been removed. No workaround without a local copy of the ICD-10-CM table.
- **No similar-code suggestions for ICD-10-CM** — the transposition logic is LOINC-specific (check digit recomputation). ICD-10-CM has no equivalent mechanism. Future improvement: off-by-one character suggestions via the API's name search.
- **Advanced details panel empty for ICD-10-CM** — the NIH API for ICD-10-CM returns only code and name. No additional fields are available.

---

## Adding Future Systems

To add HCPCS as a third system:
1. Create `internal/hcpcs/codec.go` implementing `coding.Codec`
2. Add `hcpcs.NewCodec()` to the codecs slice in `main.go`
3. Create `templates/hcpcs/tab.html`
4. No changes to handlers, routing logic, shared templates, or CSS
