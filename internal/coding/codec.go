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
	// Returns nil if the code is structurally valid.
	ValidateFormat(code string) error
	// Parse maps a display-fields row from the API response into a Result.
	Parse(fields []string) Result
	// SimilarCandidates returns codes to check when a code is not found.
	// Returns nil if the system does not support similarity search.
	SimilarCandidates(code string) []string
}
