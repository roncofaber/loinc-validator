package handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/roncofaber/loinc-validator/internal/coding"
)

type ExportHandler struct {
	tmpl *template.Template
}

func NewExportHandler(tmpl *template.Template) *ExportHandler {
	return &ExportHandler{tmpl: tmpl}
}

func (h *ExportHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rawJSON := r.FormValue("results")
	var results []coding.Result
	if err := json.Unmarshal([]byte(rawJSON), &results); err != nil {
		http.Error(w, "invalid results data", http.StatusBadRequest)
		return
	}

	filename := fmt.Sprintf("validation_%s.csv", time.Now().UTC().Format("20060102_150405"))
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)

	cw := csv.NewWriter(w)
	cw.Write([]string{"Code", "Valid", "Deprecated", "Name", "CheckedAt", "Error"})
	for _, res := range results {
		valid := "false"
		if res.Valid {
			valid = "true"
		}
		deprecated := "false"
		if res.Deprecated {
			deprecated = "true"
		}
		cw.Write([]string{
			res.Code,
			valid,
			deprecated,
			res.Name,
			res.CheckedAt.UTC().Format(time.RFC3339),
			res.Error,
		})
	}
	cw.Flush()
}
