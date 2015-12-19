package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/andrewslotin/doppelganger/git"
)

type MirrorHandler struct {
	githubRepos   *git.GithubRepositories
	mirroredRepos *git.MirroredRepositories
}

func NewMirrorHandler(githubRepos *git.GithubRepositories, mirroredRepos *git.MirroredRepositories) *MirrorHandler {
	return &MirrorHandler{
		githubRepos:   githubRepos,
		mirroredRepos: mirroredRepos,
	}
}

func (handler *MirrorHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	startTime := time.Now()

	repoName := req.FormValue("repo")
	if repoName == "" {
		http.Error(w, "Missing source repository name", http.StatusBadRequest)
		return
	}

	repo, err := handler.githubRepos.Get(repoName)
	switch err {
	case nil:
		if err := handler.mirroredRepos.Create(repo.FullName, repo.GitURL); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		log.Printf("mirrored %s from %s [%s]", repo.FullName, repo.GitURL, time.Since(startTime))
		http.Redirect(w, req, fmt.Sprintf("/mirrored?repo=%s", url.QueryEscape(repo.FullName)), http.StatusSeeOther)
	case git.ErrorNotFound:
		http.Error(w, "Source repository not found", http.StatusNotFound)
	default:
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
