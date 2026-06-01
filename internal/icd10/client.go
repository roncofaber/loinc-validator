// Package icd10 uses coding.Search (internal/coding/utils.go) for all NIH API
// calls — no system-specific client is needed because the ICD-10-CM API exposes
// only code and name with no ef= extra fields beyond what the shared search handles.
//
// If the ICD-10-CM API gains additional fields worth fetching, implement them here
// following the pattern in internal/loinc/client.go.
package icd10
