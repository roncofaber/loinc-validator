package handlers

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/roncofaber/loinc-validator/internal/coding"
	loincpkg "github.com/roncofaber/loinc-validator/internal/loinc"
)

type ValidateHandler struct {
	tmpl  *template.Template
	codec coding.Codec
}

func NewValidateHandler(tmpl *template.Template, codec coding.Codec) *ValidateHandler {
	return &ValidateHandler{tmpl: tmpl, codec: codec}
}

type resultData struct {
	Code         string
	Name         string
	ShortName    string
	Component    string
	RelatedNames string
	DataType     string
	Units        []string
	Valid         bool
	Deprecated    bool
	CheckedAt    interface{}
	Error        string
	Suggestion   *coding.Suggestion
	SimilarCode  string
	SystemID     string
}

func (h *ValidateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	code := strings.TrimSpace(r.FormValue("code"))

	if err := h.codec.ValidateFormat(code); err != nil {
		data := resultData{Error: err.Error(), SystemID: h.codec.SystemID()}
		if corrected := loincpkg.CorrectedCode(code); corrected != "" {
			res, apiErr := h.codec.Validate(corrected)
			if apiErr == nil && res.Valid {
				data.Suggestion = &coding.Suggestion{Code: res.Code, Name: res.Name}
			} else {
				data.SimilarCode = corrected
			}
		}
		h.tmpl.ExecuteTemplate(w, "result.html", data)
		return
	}

	res, err := h.codec.Validate(code)
	if err != nil {
		h.tmpl.ExecuteTemplate(w, "result.html", resultData{
			Code: code, Error: "Could not reach the API — please try again.", SystemID: h.codec.SystemID(),
		})
		return
	}

	if !res.Valid {
		h.tmpl.ExecuteTemplate(w, "result.html", resultData{
			Code: code, SystemID: h.codec.SystemID(),
		})
		return
	}

	h.tmpl.ExecuteTemplate(w, "result.html", resultData{
		Code:         res.Code,
		Name:         res.Name,
		ShortName:    res.ShortName,
		Component:    res.Component,
		RelatedNames: res.RelatedNames,
		DataType:     res.DataType,
		Units:        res.Units,
		Valid:         res.Valid,
		Deprecated:    res.Deprecated,
		CheckedAt:    res.CheckedAt,
		SystemID:     h.codec.SystemID(),
	})
}
