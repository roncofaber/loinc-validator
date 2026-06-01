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
	http  *coding.HTTPClient
}

func NewValidateHandler(tmpl *template.Template, codec coding.Codec) *ValidateHandler {
	return &ValidateHandler{tmpl: tmpl, codec: codec, http: coding.NewHTTPClient()}
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

// extrasProvider is implemented by codecs that support extra field fetching.
type extrasProvider interface {
	ValidateWithExtras(code string) (coding.Result, error)
}

func (h *ValidateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	code := strings.TrimSpace(r.FormValue("code"))

	if err := h.codec.ValidateFormat(code); err != nil {
		data := resultData{Error: err.Error(), SystemID: h.codec.SystemID()}
		// For LOINC: try corrected code via check digit
		if corrected := loincpkg.CorrectedCode(code); corrected != "" {
			rows, apiErr := h.http.Validate(h.codec, corrected)
			if apiErr == nil {
				if row, _ := coding.ExactMatch(rows, corrected); row != nil {
					res := h.codec.Parse(row)
					data.Suggestion = &coding.Suggestion{Code: res.Code, Name: res.Name}
				} else {
					data.SimilarCode = corrected
				}
			}
		}
		h.tmpl.ExecuteTemplate(w, "result.html", data)
		return
	}

	// Use extras (LOINC-specific: units, datatype, relatednames) if available.
	if ep, ok := h.codec.(extrasProvider); ok {
		res, err := ep.ValidateWithExtras(code)
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
		return
	}

	// Generic path for codecs without extra fields (ICD-10-CM, future codecs).
	rows, err := h.http.Validate(h.codec, code)
	if err != nil {
		h.tmpl.ExecuteTemplate(w, "result.html", resultData{
			Code: code, Error: "Could not reach the API — please try again.", SystemID: h.codec.SystemID(),
		})
		return
	}

	row, _ := coding.ExactMatch(rows, code)
	if row == nil {
		h.tmpl.ExecuteTemplate(w, "result.html", resultData{
			Code: code, SystemID: h.codec.SystemID(),
		})
		return
	}

	res := h.codec.Parse(row)
	h.tmpl.ExecuteTemplate(w, "result.html", resultData{
		Code:      res.Code,
		Name:      res.Name,
		Valid:      res.Valid,
		Deprecated: res.Deprecated,
		CheckedAt: res.CheckedAt,
		SystemID:  h.codec.SystemID(),
	})
}
