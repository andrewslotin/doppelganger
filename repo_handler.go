package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/google/go-github/github"
)

var (
	repoTemplate = template.Must(template.ParseFiles("templates/repo/show.html.template"))
)

type RepoHandler struct {
	client *github.Client
}

func NewRepoClient(client *github.Client) *RepoHandler {
	return &RepoHandler{
		client: client,
	}
}

func (handler *RepoHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	startTime := time.Now()
	repoOwner, repoName := ParseRepositoryName(handler.fetchRepoFromRequest(req))

	repo, response, err := handler.client.Repositories.Get(repoOwner, repoName)
	if err == nil {
		err = github.CheckResponse(response.Response)
	}

	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			http.Error(w, fmt.Sprintf("No such repository %q", repoName), http.StatusNotFound)
			return
		}

		log.Printf("failed to fetch %s/%s (%s)", repoOwner, repoName, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	masterBranch, _, err := handler.client.Repositories.GetBranch(repoOwner, repoName, *repo.MasterBranch)
	if err != nil {
		log.Printf("failed to fetch branch %s of %s (%s)", *repo.MasterBranch, *repo.FullName, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	lastCommit, _, err := handler.client.Repositories.GetCommit(repoOwner, repoName, *masterBranch.Commit.SHA)
	if err != nil {
		log.Printf("failed to fetch latest commit in %s of %s (%s)", *masterBranch.Name, *repo.FullName, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	repoTemplate.Execute(w, struct {
		Repo       *github.Repository
		LastCommit *github.RepositoryCommit
	}{
		Repo:       repo,
		LastCommit: lastCommit,
	})
	log.Printf("rendered repo/show %s with latest commit from %q [%s]", *repo.FullName, *masterBranch.Name, time.Since(startTime))
}

func (handler *RepoHandler) fetchRepoFromRequest(req *http.Request) string {
	return req.FormValue("repo")
}
