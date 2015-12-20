package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
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

	switch action := strings.ToLower(req.FormValue("action")); action {
	case "create":
		if err := handler.CreateMirror(w, repoName); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		log.Printf("mirrored %s [%s]", repoName, time.Since(startTime))
	case "update":
		if err := handler.UpdateMirror(w, repoName); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		log.Printf("updated mirror %s [%s]", repoName, time.Since(startTime))
	default:
		http.Error(w, fmt.Sprintf("Unsupported action %q", action), http.StatusBadRequest)
		return
	}

	http.Redirect(w, req, fmt.Sprintf("/mirrored?repo=%s", url.QueryEscape(repoName)), http.StatusSeeOther)
}

func (handler *MirrorHandler) CreateMirror(w http.ResponseWriter, repoName string) error {
	switch repo, err := handler.githubRepos.Get(repoName); err {
	case nil:
		return handler.mirroredRepos.Create(repo.FullName, repo.GitURL)
	case git.ErrorNotFound:
		http.Error(w, "Source repository not found", http.StatusNotFound)
		return nil
	default:
		return err
	}
}

func (handler *MirrorHandler) UpdateMirror(w http.ResponseWriter, repoName string) error {
	switch repo, err := handler.githubRepos.Get(repoName); err {
	case nil:
		return handler.mirroredRepos.Update(repo.FullName)
	case git.ErrorNotMirrored:
		http.Error(w, "Source repository not mirrored", http.StatusNotFound)
		return nil
	default:
		return err
	}
}
