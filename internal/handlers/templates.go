package handlers

import (
	"html/template"
	"path/filepath"
)

func MustLoadTemplates(dir string) *template.Template {
	tmpl, err := template.ParseGlob(filepath.Join(dir, "*.html"))
	if err != nil {
		panic("failed to parse base templates: " + err.Error())
	}
	partials, err := filepath.Glob(filepath.Join(dir, "partials", "*.html"))
	if err != nil {
		panic("failed to glob partials: " + err.Error())
	}
	if len(partials) > 0 {
		tmpl, err = tmpl.ParseFiles(partials...)
		if err != nil {
			panic("failed to parse partial templates: " + err.Error())
		}
	}
	return tmpl
}
