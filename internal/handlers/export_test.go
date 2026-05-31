package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/roncofaber/loinc-validator/internal/handlers"
	"github.com/roncofaber/loinc-validator/internal/loinc"
)

func TestExportHandler(t *testing.T) {
	tmpl := handlers.MustLoadTemplates("../../templates")
	h := handlers.NewExportHandler(tmpl)

	results := []loinc.LOINCResult{
		{Code: "2345-7", Name: "Glucose", Version: "2.73", Valid: true, CheckedAt: time.Now()},
		{Code: "99999-9", Valid: false, CheckedAt: time.Now()},
	}
	jsonBytes, _ := json.Marshal(results)

	form := url.Values{"results": {string(jsonBytes)}}
	req := httptest.NewRequest(http.MethodPost, "/export", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/csv") {
		t.Errorf("expected text/csv, got %s", contentType)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "2345-7") {
		t.Errorf("expected code 2345-7 in CSV, got: %s", body)
	}
	if !strings.Contains(body, "Code,Valid,Name,CheckedAt") {
		t.Errorf("expected CSV header, got: %s", body)
	}
}

func TestExportHandlerInvalidJSON(t *testing.T) {
	tmpl := handlers.MustLoadTemplates("../../templates")
	h := handlers.NewExportHandler(tmpl)

	form := url.Values{"results": {"not json"}}
	req := httptest.NewRequest(http.MethodPost, "/export", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}
