package main

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/andrewslotin/doppelganger/git"
)

var (
	repoTemplate      = template.Must(template.ParseFiles("templates/layout.html.template", "templates/repo/show.html.template"))
	newMirrorTemplate = template.Must(template.ParseFiles("templates/layout.html.template", "templates/repo/mirror.html.template"))
)

// RepoHandler is a type that implements http.Handler interface and is used by ReposHandler to handle single repository
// requests containing "name" parameter. The value of this parameter is used to lookup the repository and render it using Show method.
// If repository is not found a new mirror page is rendered instead using NewMirror.
type RepoHandler struct {
	repositories git.RepositoryService
}

// NewRepoHandler creates and initializes a new handler.
func NewRepoHandler(repositoryService git.RepositoryService) *RepoHandler {
	return &RepoHandler{
		repositories: repositoryService,
	}
}

func (handler *RepoHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	startTime := time.Now()

	repoName, ok := handler.fetchRepoFromRequest(req)
	if !ok {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	switch req.Method {
	case "GET":
		switch repo, err := handler.repositories.Get(repoName); err {
		case git.ErrorNotFound: // GitHub repository not found
			http.Error(w, fmt.Sprintf("No such repository %q", repoName), http.StatusNotFound)
		case git.ErrorNotMirrored: // Mirror repository not found, offer to create a new one
			repo = &git.Repository{
				FullName: repoName,
			}

			if err := handler.NewMirror(w, repo); err != nil {
				log.Printf("failed to render repo/mirror %s (%s)", repo.FullName, err)
				WriteErrorPage(w, errors.New("Internal server error"), http.StatusInternalServerError)
			} else {
				log.Printf("rendered repo/mirror %s [%s]", repo.FullName, time.Since(startTime))
			}
		case nil: // Repository found
			if err := handler.Show(w, repo); err != nil {
				log.Printf("failed to render repo/show %s with latest commit from %q (%s)", repo.FullName, repo.Master, err)
				WriteErrorPage(w, errors.New("Internal server error"), http.StatusInternalServerError)
			} else {
				log.Printf("rendered repo/show %s with latest commit from %q [%s]", repo.FullName, repo.Master, time.Since(startTime))
			}
		default: // Failed to fetch repository
			log.Printf("failed to fetch %s (%s)", repoName, err)
			WriteErrorPage(w, errors.New("Internal server error"), http.StatusInternalServerError)
		}
	case "POST":
		WriteErrorPage(w, errors.New("Not implemented"), http.StatusNotImplemented)
	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

// Show renders a repository page using templates/repo/show.html.template
func (handler *RepoHandler) Show(w http.ResponseWriter, repo *git.Repository) error {
	return repoTemplate.Execute(w, repo)
}

// NewMirror renders a new repository mirror page using templates/repo/mirror.html.template
func (handler *RepoHandler) NewMirror(w http.ResponseWriter, repo *git.Repository) error {
	return newMirrorTemplate.Execute(w, repo)
}

func (handler *RepoHandler) fetchRepoFromRequest(req *http.Request) (string, bool) {
	owner, repo := req.URL.Query().Get(":owner"), req.URL.Query().Get(":repo")
	if owner == "" || repo == "" {
		return "", false
	}

	return owner + "/" + repo, true
}
