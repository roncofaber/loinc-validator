package handlers

import (
	"html/template"
	"net/http"
	"strings"
	"sync"

	"github.com/roncofaber/loinc-validator/internal/coding"
)

type SimilarHandler struct {
	tmpl  *template.Template
	codec coding.Codec
}

func NewSimilarHandler(tmpl *template.Template, codec coding.Codec) *SimilarHandler {
	return &SimilarHandler{tmpl: tmpl, codec: codec}
}

func (h *SimilarHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		return
	}

	candidates := h.codec.SimilarCandidates(code)
	if len(candidates) == 0 {
		return
	}

	results := make([]coding.Result, len(candidates))
	var wg sync.WaitGroup
	for i, candidate := range candidates {
		wg.Add(1)
		go func(idx int, c string) {
			defer wg.Done()
			res, err := h.codec.Validate(c)
			if err == nil {
				results[idx] = res
			}
		}(i, candidate)
	}
	wg.Wait()

	var suggestions []coding.Suggestion
	seen := make(map[string]bool)
	for _, res := range results {
		if res.Valid && !seen[res.Code] {
			seen[res.Code] = true
			suggestions = append(suggestions, coding.Suggestion{Code: res.Code, Name: res.Name})
		}
	}

	if len(suggestions) == 0 {
		return
	}

	h.tmpl.ExecuteTemplate(w, "similar.html", suggestions)
}
