package main

import (
	"log"
	"net/http"

	"github.com/roncofaber/loinc-validator/internal/handlers"
	"github.com/roncofaber/loinc-validator/internal/loinc"
)

func main() {
	tmpl := handlers.MustLoadTemplates("templates")
	client := loinc.NewDefaultClient()

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		tmpl.ExecuteTemplate(w, "index.html", nil)
	})
	mux.Handle("/validate", handlers.NewValidateHandler(tmpl, client))
	mux.Handle("/batch", handlers.NewBatchHandler(tmpl, client))
	mux.Handle("/export", handlers.NewExportHandler(tmpl))

	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
