package handlers

import (
	"bufio"
	"encoding/json"
	"html/template"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/roncofaber/loinc-validator/internal/loinc"
)

const maxWorkers = 10
const maxFileSize = 5 << 20 // 5 MB

type BatchHandler struct {
	tmpl   *template.Template
	client *loinc.Client
}

func NewBatchHandler(tmpl *template.Template, client *loinc.Client) *BatchHandler {
	return &BatchHandler{tmpl: tmpl, client: client}
}

type batchSummary struct {
	Total   int
	Valid   int
	Invalid int
	Errors  int
}

type batchTemplateData struct {
	Results     []loinc.LOINCResult
	ResultsJSON string
	Summary     batchSummary
	Error       string
}

func (h *BatchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxFileSize)
	file, _, err := r.FormFile("file")
	if err != nil {
		h.tmpl.ExecuteTemplate(w, "batch_result.html", batchTemplateData{
			Error: "Please upload a file (no file received).",
		})
		return
	}
	defer file.Close()

	var codes []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
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
	})
}

func (h *BatchHandler) validateConcurrent(codes []string) []loinc.LOINCResult {
	results := make([]loinc.LOINCResult, len(codes))
	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup

	for i, code := range codes {
		wg.Add(1)
		go func(idx int, c string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := loinc.ValidateFormat(c); err != nil {
				results[idx] = loinc.LOINCResult{Code: c, Error: err.Error(), CheckedAt: time.Now()}
				return
			}
			result, err := h.client.Validate(c)
			if err != nil {
				result.Code = c
				result.Error = "API error: " + err.Error()
			}
			results[idx] = result
		}(i, code)
	}

	wg.Wait()
	return results
}
