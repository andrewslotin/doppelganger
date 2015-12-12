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
	repoName := handler.fetchRepoFromRequest(req)

	repo, response, err := handler.client.Repositories.Get(ParseRepositoryName(repoName))
	if err == nil {
		err = github.CheckResponse(response.Response)
	}

	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			http.Error(w, fmt.Sprintf("No such repository %q", repoName), http.StatusNotFound)
			return
		}

		log.Printf("failed to fetch %s (%s)", repoName, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	repoTemplate.Execute(w, repo)
	log.Printf("rendered repo/show %s [%s]", repoName, time.Since(startTime))
}

func (handler *RepoHandler) fetchRepoFromRequest(req *http.Request) string {
	return req.FormValue("repo")
}
