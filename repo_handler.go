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
	repoName := handler.fetchRepoFromRequest(req)

	switch req.Method {
	case "GET":
		handler.Show(w, repoName)
	case "POST":
		http.Error(w, "Not implemented", http.StatusNotImplemented)
	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

func (handler *RepoHandler) Show(w http.ResponseWriter, repoName string) {
	startTime := time.Now()

	switch repo, err := handler.repositories.Get(repoName); err {
	case nil:
		if err := repoTemplate.Execute(w, repo); err != nil {
			log.Printf("failed to render repo/show %s with latest commit from %q (%s)", repo.FullName, repo.Master, err)
		} else {
			log.Printf("rendered repo/show %s with latest commit from %q [%s]", repo.FullName, repo.Master, time.Since(startTime))
		}
	case git.ErrorNotFound:
		http.Error(w, fmt.Sprintf("No such repository %q", repoName), http.StatusNotFound)
	case git.ErrorNotMirrored:
		repo := &git.Repository{
			FullName: repoName,
			GitURL:   fmt.Sprintf("git@doppelganger.local:%s.git", repoName),
		}

		if err := newMirrorTemplate.Execute(w, repo); err != nil {
			log.Printf("failed to render repo/mirror %s (%s)", repo.FullName, err)
		} else {
			log.Printf("rendered repo/mirror %s [%s]", repo.FullName, time.Since(startTime))
		}
	default:
		log.Printf("failed to fetch %s (%s)", repoName, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (handler *RepoHandler) fetchRepoFromRequest(req *http.Request) string {
	return req.FormValue("repo")
}
