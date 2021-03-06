package main

import (
	"html/template"
	"net/http"
)

var (
	internalErrorTemplate = template.Must(template.ParseFiles("templates/layout.html.template", "templates/errors/internal_error.html.template"))
	notFoundErrorTemplate = template.Must(template.ParseFiles("templates/layout.html.template", "templates/errors/not_found.html.template"))
)

// UserError wraps an internal server error and replaces its message with a
// user-friendly version, while keeping the original error for inspection.
type UserError struct {
	Message       string
	BackURL       string
	OriginalError error
}

// Error returns the message that is being sent back to user.
func (e UserError) Error() string {
	return e.Message
}

// WriteErrorPage renders an internal server error in a user-friendly way.
func WriteErrorPage(w http.ResponseWriter, err error, status int) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(status)
	internalErrorTemplate.Execute(w, err)
}

// WriteNotFoundPage renders an HTTP 404 Not Found page with an optional message and a link back.
func WriteNotFoundPage(w http.ResponseWriter, message, backURL string) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusNotFound)
	notFoundErrorTemplate.Execute(w, struct{ Message, BackURL string }{message, backURL})
}
