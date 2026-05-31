# Example LOINC Codes

This folder contains example codes and batch files for testing the LOINC Code Validator.

## Single codes to try

| Code | Description | Expected result |
|------|-------------|-----------------|
| `2345-7` | Glucose in Serum or Plasma | Valid |
| `8867-4` | Heart rate | Valid |
| `8480-6` | Systolic blood pressure | Valid |
| `100000-9` | 6-digit code (LOINC v2.74+) | Valid |
| `1009-0` | Deprecated antiglobulin test | Valid + deprecation warning |
| `100653-5` | Deprecated audiometry panel | Valid + deprecation warning |
| `102006-4` | Discouraged von Willebrand test | Valid (no warning — see Limitations in README) |
| `99999-9` | Non-existent code | Invalid |
| `abc` | Malformed input | Format error |
| `` | Empty input | Format error |

## Batch files

| File | Contents |
|------|----------|
| `batch_common_labs.txt` | 20 most common laboratory test codes — all active |
| `batch_vital_signs.txt` | Vital sign codes — all active |
| `batch_mixed_status.txt` | Mix of active, deprecated, discouraged, invalid, and malformed codes |
