# Example Codes

This folder contains example codes and batch files for testing the validator.

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
| `2345-4` | Wrong check digit (correct is 7) | Format error with suggestion |
| `abc` | Malformed input | Format error |
| `` | Empty input | Format error |

## LOINC Batch files

| File | Contents |
|------|----------|
| `loinc_batch_common_labs.txt` | 20 most common laboratory test codes — all active |
| `loinc_batch_vital_signs.txt` | Vital sign codes — all active |
| `loinc_batch_mixed_status.txt` | Mix of active, deprecated, discouraged, invalid, and malformed codes |
| `loinc_batch_large.txt` | 500 active codes sampled across clinical domains (chemistry, microbiology, hematology, drug/tox, serology, allergy, radiology, H&P, panels, coagulation, urinalysis, cardiology, vital signs, pathology) — good for testing performance and batch export |

## ICD-10-CM Single codes to try

| Code | Description | Expected result |
|------|-------------|-----------------|
| `E11.9` | Type 2 diabetes mellitus without complications | Valid |
| `I10` | Essential (primary) hypertension | Valid |
| `S00.00XA` | Scalp injury, initial encounter (X placeholder) | Valid |
| `A01` | Typhoid and paratyphoid fevers (category header) | Not found — non-billable |
| `U07.1` | COVID-19 (U reserved — invalid in ICD-10-CM) | Format error |
| `Z99.99` | Non-existent code | Not found |
| `123` | Malformed input | Format error |

## ICD-10-CM Batch files

| File | Contents |
|------|----------|
| `icd10_batch_common.txt` | 8 common diagnosis codes — all billable |
| `icd10_batch_large.txt` | ~83 billable codes across all chapters (A–Z excl. U), including X-placeholder and 7-character codes |
| `icd10_batch_mixed.txt` | Mix of valid billable, non-billable headers (correctly "not found"), non-existent, and malformed codes |
