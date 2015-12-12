package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/andrewslotin/doppelganger/git"
)

var (
	repoTemplate = template.Must(template.ParseFiles("templates/repo/show.html.template"))
)

type RepoHandler struct {
	repositories git.RepositoryService
}

func NewRepoClient(repositoryService git.RepositoryService) *RepoHandler {
	return &RepoHandler{
		repositories: repositoryService,
	}
}

func (handler *RepoHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	startTime := time.Now()
	repoName := handler.fetchRepoFromRequest(req)

	switch repo, err := handler.repositories.Get(repoName); err {
	case nil:
		repoTemplate.Execute(w, repo)
		log.Printf("rendered repo/show %s with latest commit from %q [%s]", repo.FullName, repo.Master, time.Since(startTime))
	case git.ErrorNotFound:
		http.Error(w, fmt.Sprintf("No such repository %q", repoName), http.StatusNotFound)
	default:
		log.Printf("failed to fetch %s (%s)", repoName, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (handler *RepoHandler) fetchRepoFromRequest(req *http.Request) string {
	return req.FormValue("repo")
}
