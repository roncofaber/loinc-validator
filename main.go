package main

import (
	"log"
	"net/http"

	"github.com/roncofaber/loinc-validator/internal/coding"
	"github.com/roncofaber/loinc-validator/internal/handlers"
	"github.com/roncofaber/loinc-validator/internal/icd10"
	"github.com/roncofaber/loinc-validator/internal/loinc"
)

func main() {
	tmpl := handlers.MustLoadTemplates("templates")

	codecs := []coding.Codec{
		loinc.NewCodec(),
		icd10.NewCodec(),
	}

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.Handle("/export", handlers.NewExportHandler(tmpl))

	for _, codec := range codecs {
		id := codec.SystemID()
		mux.Handle("/"+id+"/validate",        handlers.NewValidateHandler(tmpl, codec))
		mux.Handle("/"+id+"/suggest",         handlers.NewSuggestHandler(tmpl, codec))
		mux.Handle("/"+id+"/suggest-similar", handlers.NewSimilarHandler(tmpl, codec))
		mux.Handle("/"+id+"/batch",           handlers.NewBatchHandler(tmpl, codec))
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		tmpl.ExecuteTemplate(w, "index.html", map[string]any{
			"Codecs": codecs,
		})
	})

	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
