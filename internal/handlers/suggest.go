package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const suggestBaseURL = "https://clinicaltables.nlm.nih.gov/api/loinc_items/v3/search"

type SuggestHandler struct {
	tmpl       *template.Template
	httpClient *http.Client
}

func NewSuggestHandler(tmpl *template.Template) *SuggestHandler {
	return &SuggestHandler{
		tmpl:       tmpl,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
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

	params := url.Values{}
	params.Set("terms", q)
	params.Set("df", "LOINC_NUM,LONG_COMMON_NAME")
	params.Set("maxList", "6")

	resp, err := h.httpClient.Get(suggestBaseURL + "?" + params.Encode())
	if err != nil || resp.StatusCode != http.StatusOK {
		w.WriteHeader(http.StatusOK)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	var raw []json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil || len(raw) < 4 {
		w.WriteHeader(http.StatusOK)
		return
	}

	var displayFields [][]string
	if err := json.Unmarshal(raw[3], &displayFields); err != nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	var suggestions []suggestion
	for _, fields := range displayFields {
		if len(fields) >= 2 {
			suggestions = append(suggestions, suggestion{Code: fields[0], Name: fields[1]})
		}
	}

	h.tmpl.ExecuteTemplate(w, "suggest.html", suggestions)
	fmt.Fprint(w, "") // ensure non-nil response
}
