package main

import (
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/andrewslotin/doppelganger/git"
)

var (
	reposTemplate = template.Must(template.ParseFiles("templates/layout.html.template", "templates/repos/index.html.template"))
)

// ReposHandler is a type that implements http.Handler interface and is used to render repository lists.
// Doppelganger uses ReposHandler to display both GitHub and local repositories.
type ReposHandler struct {
	repositories git.RepositoryService
	mirrors      bool
}

// NewReposHandler creates and initializes a new handler.
func NewReposHandler(repositoryService git.RepositoryService, mirrors bool) *ReposHandler {
	return &ReposHandler{
		repositories: repositoryService,
		mirrors:      mirrors,
	}
}

func (handler *ReposHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	startTime := time.Now()
	ctx := req.Context()

	repos, err := handler.repositories.All(ctx)
	if err != nil {
		log.Printf("failed to get repos (%s) %v", err, req)
		WriteErrorPage(w, UserError{Message: "Internal server error", BackURL: req.Referer(), OriginalError: err}, http.StatusInternalServerError)
		return
	}

	values := struct {
		Repositories []*git.Repository
		Mirrors      bool
	}{repos, handler.mirrors}
	if err := reposTemplate.Execute(w, values); err != nil {
		log.Printf("failed to render repos/index with %d entries (%s)", len(repos), err)
		WriteErrorPage(w, UserError{Message: "Internal server error", BackURL: req.Referer(), OriginalError: err}, http.StatusInternalServerError)
	} else {
		log.Printf("rendered repos/index with %d entries [%s]", len(repos), time.Since(startTime))
	}
}
