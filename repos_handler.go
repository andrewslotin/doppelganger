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
}

// NewReposHandler creates and initializes a new handler.
func NewReposHandler(repositoryService git.RepositoryService) *ReposHandler {
	return &ReposHandler{
		repositories: repositoryService,
	}
}

func (handler *ReposHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	startTime := time.Now()

	repos, err := handler.repositories.All()
	if err != nil {
		log.Printf("failed to get repos (%s) %v", err, req)
		WriteErrorPage(w, UserError{Message: "Internal server error", OriginalError: err}, http.StatusInternalServerError)
		return
	}

	if err := reposTemplate.Execute(w, repos); err != nil {
		log.Printf("failed to render repos/index with %d entries (%s)", len(repos), err)
		WriteErrorPage(w, UserError{Message: "Internal server error", OriginalError: err}, http.StatusInternalServerError)
	} else {
		log.Printf("rendered repos/index with %d entries [%s]", len(repos), time.Since(startTime))
	}
}
