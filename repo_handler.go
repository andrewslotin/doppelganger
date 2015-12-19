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
	repoTemplate      = template.Must(template.ParseFiles("templates/repo/show.html.template"))
	newMirrorTemplate = template.Must(template.ParseFiles("templates/repo/mirror.html.template"))
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

	switch req.Method {
	case "GET":
		switch repo, err := handler.repositories.Get(repoName); err {
		case git.ErrorNotFound: // GitHub repository not found
			http.Error(w, fmt.Sprintf("No such repository %q", repoName), http.StatusNotFound)
		case git.ErrorNotMirrored: // Mirror repository not found, offer to create a new one
			repo = &git.Repository{
				FullName: repoName,
				GitURL:   fmt.Sprintf("git@doppelganger.local:%s.git", repoName),
			}

			if err := handler.NewMirror(w, repo); err != nil {
				log.Printf("failed to render repo/mirror %s (%s)", repo.FullName, err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			} else {
				log.Printf("rendered repo/mirror %s [%s]", repo.FullName, time.Since(startTime))
			}
		case nil: // Repository found
			if err := handler.Show(w, repo); err != nil {
				log.Printf("failed to render repo/show %s with latest commit from %q (%s)", repo.FullName, repo.Master, err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			} else {
				log.Printf("rendered repo/show %s with latest commit from %q [%s]", repo.FullName, repo.Master, time.Since(startTime))
			}
		default: // Failed to fetch repository
			log.Printf("failed to fetch %s (%s)", repoName, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	case "POST":
		http.Error(w, "Not implemented", http.StatusNotImplemented)
	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

func (handler *RepoHandler) Show(w http.ResponseWriter, repo *git.Repository) error {
	return repoTemplate.Execute(w, repo)
}

func (handler *RepoHandler) NewMirror(w http.ResponseWriter, repo *git.Repository) error {
	return newMirrorTemplate.Execute(w, repo)
}

func (handler *RepoHandler) fetchRepoFromRequest(req *http.Request) string {
	return req.FormValue("repo")
}
