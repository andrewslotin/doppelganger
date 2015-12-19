package main

import (
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/andrewslotin/doppelganger/git"
)

var (
	reposTemplate = template.Must(template.ParseFiles("templates/repos/index.html.template"))
)

type ReposHandler struct {
	repositories git.RepositoryService
}

func NewReposHandler(repositoryService git.RepositoryService) *ReposHandler {
	return &ReposHandler{
		repositories: repositoryService,
	}
}

func (handler *ReposHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	startTime := time.Now()

	if repoName := req.FormValue("repo"); repoName != "" {
		NewRepoClient(handler.repositories).ServeHTTP(w, req)
		return
	}

	repos, err := handler.repositories.All()
	if err != nil {
		log.Printf("failed to get repos (%s) %s", err, req)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := reposTemplate.Execute(w, repos); err != nil {
		log.Printf("failed to render repos/index with %d entries (%s)", len(repos), err)
	} else {
		log.Printf("rendered repos/index with %d entries [%s]", len(repos), time.Since(startTime))
	}
}
