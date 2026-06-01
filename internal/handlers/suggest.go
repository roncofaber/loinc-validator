package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/roncofaber/loinc-validator/internal/coding"
)

type SuggestHandler struct {
	tmpl  *template.Template
	codec coding.Codec
}

func NewSuggestHandler(tmpl *template.Template, codec coding.Codec) *SuggestHandler {
	return &SuggestHandler{tmpl: tmpl, codec: codec}
}

type suggestion struct {
	Code string
	Name string
}

func (h *SuggestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("code"))
	if len(q) < 2 {
		w.WriteHeader(http.StatusOK)
		return
	}

	rows, err := h.codec.Suggest(q, 6)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	var suggestions []suggestion
	for _, row := range rows {
		if len(row) >= 2 {
			suggestions = append(suggestions, suggestion{Code: row[0], Name: row[1]})
		}
	}

	h.tmpl.ExecuteTemplate(w, "suggest.html", suggestions)
	fmt.Fprint(w, "")
}
