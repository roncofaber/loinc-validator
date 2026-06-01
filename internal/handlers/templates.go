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
	// Load partials
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
	// Load system tab templates (loinc/tab.html, icd10/tab.html, etc.)
	systemTabs, err := filepath.Glob(filepath.Join(dir, "*/tab.html"))
	if err != nil {
		panic("failed to glob system tabs: " + err.Error())
	}
	if len(systemTabs) > 0 {
		tmpl, err = tmpl.ParseFiles(systemTabs...)
		if err != nil {
			panic("failed to parse system tab templates: " + err.Error())
		}
	}
	return tmpl
}
