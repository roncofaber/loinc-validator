package handlers

import (
	"bufio"
	"encoding/json"
	"html/template"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/roncofaber/loinc-validator/internal/coding"
)

const maxWorkers = 10
const maxFileSize = 5 << 20

type BatchHandler struct {
	tmpl  *template.Template
	codec coding.Codec
}

func NewBatchHandler(tmpl *template.Template, codec coding.Codec) *BatchHandler {
	return &BatchHandler{tmpl: tmpl, codec: codec}
}

type batchSummary struct {
	Total   int
	Valid   int
	Invalid int
	Errors  int
}

type batchTemplateData struct {
	Results     []coding.Result
	ResultsJSON string
	Summary     batchSummary
	Error       string
	SystemID    string
}

func (h *BatchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxFileSize)
	file, _, err := r.FormFile("file")
	if err != nil {
		msg := "Please upload a file (no file received)."
		if err.Error() == "http: request body too large" {
			msg = "File too large — maximum size is 5 MB."
		}
		h.tmpl.ExecuteTemplate(w, "batch_result.html", batchTemplateData{Error: msg})
		return
	}
	defer file.Close()

	var codes []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if i := strings.Index(line, "#"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if line != "" {
			codes = append(codes, line)
		}
	}

	if len(codes) == 0 {
		h.tmpl.ExecuteTemplate(w, "batch_result.html", batchTemplateData{
			Error: "The uploaded file is empty or contains no valid lines.",
		})
		return
	}

	results := h.validateConcurrent(codes)
	summary := batchSummary{Total: len(results)}
	for _, res := range results {
		switch {
		case res.Error != "":
			summary.Errors++
		case res.Valid:
			summary.Valid++
		default:
			summary.Invalid++
		}
	}

	jsonBytes, _ := json.Marshal(results)
	h.tmpl.ExecuteTemplate(w, "batch_result.html", batchTemplateData{
		Results:     results,
		ResultsJSON: string(jsonBytes),
		Summary:     summary,
		SystemID:    h.codec.SystemID(),
	})
}

func (h *BatchHandler) validateConcurrent(codes []string) []coding.Result {
	results := make([]coding.Result, len(codes))
	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup

	for i, code := range codes {
		wg.Add(1)
		go func(idx int, c string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := h.codec.ValidateFormat(c); err != nil {
				results[idx] = coding.Result{Code: c, Error: err.Error(), CheckedAt: time.Now()}
				return
			}
			res, err := h.codec.Validate(c)
			if err != nil {
				results[idx] = coding.Result{Code: c, Error: "API error: " + err.Error(), CheckedAt: time.Now()}
				return
			}
			results[idx] = res
		}(i, code)
	}

	wg.Wait()
	return results
}
