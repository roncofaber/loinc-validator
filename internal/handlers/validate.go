package handlers

import (
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/roncofaber/loinc-validator/internal/loinc"
)

type ValidateHandler struct {
	tmpl   *template.Template
	client *loinc.Client
}

func NewValidateHandler(tmpl *template.Template, client *loinc.Client) *ValidateHandler {
	return &ValidateHandler{tmpl: tmpl, client: client}
}

type resultData struct {
	Code          string
	Name          string
	ShortName     string
	Component     string
	RelatedNames  string
	DataType      string
	Units         []string
	Valid          bool
	Deprecated     bool
	CheckedAt     time.Time
	Error         string
	Suggestion    *loinc.Suggestion // set when corrected code exists directly
	SimilarCode   string            // set when corrected code doesn't exist — triggers transposition search
}

func (h *ValidateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	code := strings.TrimSpace(r.FormValue("code"))

	if err := loinc.ValidateFormat(code); err != nil {
		data := resultData{Error: err.Error()}

		// If the only problem is the check digit, look up the corrected code.
		// If it exists, show it as a direct clickable suggestion.
		// If it doesn't exist either, pass it to the template so transposition
		// search runs on the corrected form.
		if corrected := loinc.CorrectedCode(code); corrected != "" {
			if res, apiErr := h.client.Validate(corrected); apiErr == nil && res.Valid {
				data.Suggestion = &loinc.Suggestion{
					Code: res.Code,
					Name: res.Name,
				}
			} else {
				data.SimilarCode = corrected
			}
		}

		h.tmpl.ExecuteTemplate(w, "result.html", data)
		return
	}

	result, err := h.client.Validate(code)
	if err != nil {
		h.tmpl.ExecuteTemplate(w, "result.html", resultData{
			Code:  code,
			Error: "Could not reach the LOINC API — please try again.",
		})
		return
	}

	h.tmpl.ExecuteTemplate(w, "result.html", resultData{
		Code:         result.Code,
		Name:         result.Name,
		ShortName:    result.ShortName,
		Component:    result.Component,
		RelatedNames: result.RelatedNames,
		DataType:     result.DataType,
		Units:        result.Units,
		Valid:         result.Valid,
		Deprecated:    result.Deprecated,
		CheckedAt:    result.CheckedAt,
	})
}
