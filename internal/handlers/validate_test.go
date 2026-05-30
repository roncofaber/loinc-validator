package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/roncofaber/loinc-validator/internal/handlers"
	"github.com/roncofaber/loinc-validator/internal/loinc"
)

func TestValidateHandlerEmptyInput(t *testing.T) {
	tmpl := handlers.MustLoadTemplates("../../templates")
	client := loinc.NewDefaultClient()
	h := handlers.NewValidateHandler(tmpl, client)

	form := url.Values{"code": {""}}
	req := httptest.NewRequest(http.MethodPost, "/validate", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "cannot be empty") {
		t.Errorf("expected empty-input error message, got: %s", rec.Body.String())
	}
}

func TestValidateHandlerMalformedCode(t *testing.T) {
	tmpl := handlers.MustLoadTemplates("../../templates")
	client := loinc.NewDefaultClient()
	h := handlers.NewValidateHandler(tmpl, client)

	form := url.Values{"code": {"notacode"}}
	req := httptest.NewRequest(http.MethodPost, "/validate", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "invalid LOINC format") {
		t.Errorf("expected format error message, got: %s", rec.Body.String())
	}
}
