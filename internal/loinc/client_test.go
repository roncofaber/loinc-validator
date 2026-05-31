package loinc_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/roncofaber/loinc-validator/internal/loinc"
)

func mockServer(response string, statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		w.Write([]byte(response))
	}))
}

func TestValidCode(t *testing.T) {
	body := `[1, ["2345-7"], null, [["2345-7", "Glucose [Mass/volume] in Serum or Plasma"]]]`
	srv := mockServer(body, 200)
	defer srv.Close()

	client := loinc.NewClient(srv.URL)
	result, err := client.Validate("2345-7")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Error("expected valid=true for known code")
	}
	if result.Code != "2345-7" {
		t.Errorf("expected code 2345-7, got %s", result.Code)
	}
	if result.Name != "Glucose [Mass/volume] in Serum or Plasma" {
		t.Errorf("unexpected name: %s", result.Name)
	}
	if result.CheckedAt.IsZero() {
		t.Error("CheckedAt should not be zero")
	}
}

func TestInvalidCode(t *testing.T) {
	body := `[0, [], null, []]`
	srv := mockServer(body, 200)
	defer srv.Close()

	client := loinc.NewClient(srv.URL)
	result, err := client.Validate("99999-9")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Error("expected valid=false for unknown code")
	}
}

func TestAPIError(t *testing.T) {
	srv := mockServer("internal server error", 500)
	defer srv.Close()

	client := loinc.NewClient(srv.URL)
	_, err := client.Validate("2345-7")

	if err == nil {
		t.Error("expected error for non-200 response")
	}
}

func TestMalformedResponse(t *testing.T) {
	srv := mockServer("not json", 200)
	defer srv.Close()

	client := loinc.NewClient(srv.URL)
	_, err := client.Validate("2345-7")

	if err == nil {
		t.Error("expected error for malformed JSON")
	}
}

func TestCodeMismatch(t *testing.T) {
	body := `[1, ["23456-7"], null, [["23456-7", "Some other test"]]]`
	srv := mockServer(body, 200)
	defer srv.Close()

	client := loinc.NewClient(srv.URL)
	result, err := client.Validate("2345-7")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Error("expected valid=false when returned code doesn't match exactly")
	}
}

func TestDeprecatedCode(t *testing.T) {
	body := `[1, ["100653-5"], null, [["100653-5", "Deprecated Pure tone air conduction threshold audiometry panel"]]]`
	srv := mockServer(body, 200)
	defer srv.Close()

	client := loinc.NewClient(srv.URL)
	result, err := client.Validate("100653-5")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Error("expected valid=true for existing (but deprecated) code")
	}
	if !result.Deprecated {
		t.Error("expected deprecated=true for code with 'Deprecated ' prefix in name")
	}
}

func TestActiveCodeNotDeprecated(t *testing.T) {
	body := `[1, ["2345-7"], null, [["2345-7", "Glucose [Mass/volume] in Serum or Plasma"]]]`
	srv := mockServer(body, 200)
	defer srv.Close()

	client := loinc.NewClient(srv.URL)
	result, err := client.Validate("2345-7")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Deprecated {
		t.Error("expected deprecated=false for active code")
	}
}

func TestAdvancedFields(t *testing.T) {
	body := `[1, ["2345-7"], {"RELATEDNAMES2":["Glu; Plasma; Serum"],"datatype":["REAL"],"units":[[{"unit":"mg/dL"},{"unit":"mmol/L"}]]}, [["2345-7", "Glucose [Mass/volume] in Serum or Plasma", "Glucose SerPl-mCnc", "Glucose"]]]`
	srv := mockServer(body, 200)
	defer srv.Close()

	client := loinc.NewClient(srv.URL)
	result, err := client.Validate("2345-7")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ShortName != "Glucose SerPl-mCnc" {
		t.Errorf("unexpected ShortName: %q", result.ShortName)
	}
	if result.Component != "Glucose" {
		t.Errorf("unexpected Component: %q", result.Component)
	}
	if result.DataType != "REAL" {
		t.Errorf("unexpected DataType: %q", result.DataType)
	}
	if result.RelatedNames != "Glu; Plasma; Serum" {
		t.Errorf("unexpected RelatedNames: %q", result.RelatedNames)
	}
	if len(result.Units) != 2 || result.Units[0] != "mg/dL" || result.Units[1] != "mmol/L" {
		t.Errorf("unexpected Units: %v", result.Units)
	}
}

func TestCheckedAtAlwaysSet(t *testing.T) {
	body := `[0, [], null, []]`
	srv := mockServer(body, 200)
	defer srv.Close()

	before := time.Now()
	client := loinc.NewClient(srv.URL)
	result, _ := client.Validate("99999-9")
	after := time.Now()

	if result.CheckedAt.Before(before) || result.CheckedAt.After(after) {
		t.Error("CheckedAt should be set to approximately now")
	}
}
