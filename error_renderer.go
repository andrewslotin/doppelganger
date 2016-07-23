package main

import (
	"html/template"
	"net/http"
)

var (
	internalErrorTemplate = template.Must(template.ParseFiles("templates/layout.html.template", "templates/errors/internal_error.html.template"))
)

// WriteErrorPage renders an internal server error in a user-friendly way.
func WriteErrorPage(w http.ResponseWriter, err error, status int) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(status)
	internalErrorTemplate.Execute(w, err)
}
