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
	Code       string
	Name       string
	Valid       bool
	Deprecated  bool
	CheckedAt  time.Time
	Error      string
}

func (h *ValidateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	code := strings.TrimSpace(r.FormValue("code"))

	if err := loinc.ValidateFormat(code); err != nil {
		h.tmpl.ExecuteTemplate(w, "result.html", resultData{Error: err.Error()})
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
		Code:       result.Code,
		Name:       result.Name,
		Valid:       result.Valid,
		Deprecated:  result.Deprecated,
		CheckedAt:  result.CheckedAt,
	})
}
