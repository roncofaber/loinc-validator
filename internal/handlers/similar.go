package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"sync"

	"github.com/roncofaber/loinc-validator/internal/loinc"
)

type SimilarHandler struct {
	tmpl   *template.Template
	client *loinc.Client
}

func NewSimilarHandler(tmpl *template.Template, client *loinc.Client) *SimilarHandler {
	return &SimilarHandler{tmpl: tmpl, client: client}
}

func (h *SimilarHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		return
	}

	candidates := transpositionCandidates(code)
	if len(candidates) == 0 {
		return
	}

	results := make([]loinc.LOINCResult, len(candidates))
	var wg sync.WaitGroup
	for i, candidate := range candidates {
		wg.Add(1)
		go func(idx int, c string) {
			defer wg.Done()
			res, err := h.client.Validate(c)
			if err == nil {
				results[idx] = res
			}
		}(i, candidate)
	}
	wg.Wait()

	var suggestions []loinc.Suggestion
	seen := make(map[string]bool)
	for _, res := range results {
		if res.Valid && !seen[res.Code] {
			seen[res.Code] = true
			suggestions = append(suggestions, loinc.Suggestion{
				Code: res.Code,
				Name: res.Name,
			})
		}
	}

	if len(suggestions) == 0 {
		return
	}

	h.tmpl.ExecuteTemplate(w, "similar.html", suggestions)
}

// transpositionCandidates returns codes formed by swapping each pair of
// adjacent digits in the numeric prefix, with recomputed check digits.
func transpositionCandidates(code string) []string {
	parts := strings.SplitN(code, "-", 2)
	if len(parts) != 2 {
		return nil
	}
	prefix := parts[0]
	seen := make(map[string]bool)
	var candidates []string
	for i := 0; i < len(prefix)-1; i++ {
		b := []byte(prefix)
		b[i], b[i+1] = b[i+1], b[i]
		newPrefix := string(b)
		if newPrefix == prefix {
			continue
		}
		chk := loinc.CheckDigit(newPrefix)
		candidate := fmt.Sprintf("%s-%d", newPrefix, chk)
		if !seen[candidate] {
			seen[candidate] = true
			candidates = append(candidates, candidate)
		}
	}
	return candidates
}
