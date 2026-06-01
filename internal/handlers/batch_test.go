package handlers_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/roncofaber/loinc-validator/internal/handlers"
	"github.com/roncofaber/loinc-validator/internal/loinc"
)

func makeFileUpload(t *testing.T, content string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("file", "codes.txt")
	if err != nil {
		t.Fatal(err)
	}
	fw.Write([]byte(content))
	w.Close()
	req := httptest.NewRequest(http.MethodPost, "/loinc/batch", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func TestBatchHandlerNoFile(t *testing.T) {
	tmpl := handlers.MustLoadTemplates("../../templates")
	h := handlers.NewBatchHandler(tmpl, loinc.NewCodec())

	req := httptest.NewRequest(http.MethodPost, "/loinc/batch", nil)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=xxx")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), "no file") {
		t.Errorf("expected 'no file' error, got: %s", rec.Body.String())
	}
}

func TestBatchHandlerEmptyFile(t *testing.T) {
	tmpl := handlers.MustLoadTemplates("../../templates")
	h := handlers.NewBatchHandler(tmpl, loinc.NewCodec())

	req := makeFileUpload(t, "")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), "empty") {
		t.Errorf("expected 'empty' error, got: %s", rec.Body.String())
	}
}
