package main

import (
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/google/go-github/github"
)

var (
	reposTemplate = template.Must(template.ParseFiles("templates/repos/index.html.template"))
)

type ReposHandler struct {
	client *github.Client
}

func NewReposHandler(client *github.Client) *ReposHandler {
	return &ReposHandler{
		client: client,
	}
}

func (handler *ReposHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	startTime := time.Now()

	repos, err := handler.listRepos()
	if err != nil {
		log.Printf("failed to get repos (%s) %s", err, req)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	reposTemplate.Execute(w, repos)
	log.Printf("rendered repos/index with %d entries [%s]", len(repos), time.Since(startTime))
}

func (handler *ReposHandler) listRepos() ([]github.Repository, error) {
	opts := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{
			PerPage: 50,
		},
	}

	allRepos := make([]github.Repository, 0, 50)
	for {
		repos, response, err := handler.client.Repositories.List("", opts)
		if err != nil {
			return nil, err
		}

		allRepos = append(allRepos, repos...)
		if response.NextPage == 0 {
			break
		}

		opts.ListOptions.Page = response.NextPage
	}

	return allRepos, nil
}
