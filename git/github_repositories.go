package git

import (
	"errors"
	"log"
	"net/http"
	"strings"

	api "github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

var ErrorNotFound = errors.New("not found")

type GithubRepositories struct {
	client *api.Client
}

func NewGithubRepositories(token string) *GithubRepositories {
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: token,
	})
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)

	return &GithubRepositories{
		client: api.NewClient(oauthClient),
	}
}

func (service *GithubRepositories) All() ([]*Repository, error) {
	opts := &api.RepositoryListOptions{
		ListOptions: api.ListOptions{
			PerPage: 50,
		},
	}

	var allRepos []*Repository

	paginatedRepos := make([]*Repository, opts.ListOptions.PerPage)
	for {
		githubRepos, response, err := service.client.Repositories.List("", opts)
		if err != nil {
			return nil, err
		}

		for i, githubRepo := range githubRepos {
			if githubRepo.FullName == nil {
				log.Printf("[WARN] excluding GitHub repository without full_name %v", githubRepo)
				continue
			}

			if githubRepo.GitURL == nil {
				log.Printf("[WARN] excluding GitHub repository without git_url %v", githubRepo)
				continue
			}

			paginatedRepos[i] = &Repository{
				FullName: *githubRepo.FullName,
				Master:   "master",
			}

			if githubRepo.Description != nil {
				paginatedRepos[i].Description = *githubRepo.Description
			}

			if githubRepo.DefaultBranch != nil {
				paginatedRepos[i].Master = *githubRepo.DefaultBranch
			}

			if githubRepo.HTMLURL != nil {
				paginatedRepos[i].HTMLURL = *githubRepo.HTMLURL
			}
		}

		allRepos = append(allRepos, paginatedRepos[0:len(githubRepos)]...)
		if response.NextPage == 0 {
			break
		}

		opts.ListOptions.Page = response.NextPage
	}

	return allRepos, nil
}

func (service *GithubRepositories) Get(fullName string) (*Repository, error) {
	repoOwner, repoName := ParseRepositoryName(fullName)

	githubRepo, response, err := service.client.Repositories.Get(repoOwner, repoName)
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			return nil, ErrorNotFound
		}

		return nil, err
	}

	masterBranch, _, err := service.client.Repositories.GetBranch(repoOwner, repoName, *githubRepo.DefaultBranch)
	if err != nil {
		return nil, err
	}

	lastCommit, _, err := service.client.Git.GetCommit(repoOwner, repoName, *masterBranch.Commit.SHA)
	if err != nil {
		return nil, err
	}

	repo := repositoryFromGithub(githubRepo)
	repo.LatestMasterCommit = commitFromGithub(lastCommit)

	return repo, nil
}

func (service *GithubRepositories) Track(fullName, callbackURL string) error {
	owner, name := ParseRepositoryName(fullName)
	return service.registerPushWebhook(owner, name, callbackURL)
}

func (service *GithubRepositories) registerPushWebhook(owner, repo, cbURL string) error {
	hook := &api.Hook{
		Name:   new(string),
		Active: new(bool),
		Events: []string{"push"},
		Config: map[string]interface{}{
			"url":          cbURL,
			"content_type": "json",
		},
	}
	*hook.Name = "web"
	*hook.Active = true

	_, _, err := service.client.Repositories.CreateHook(owner, repo, hook)

	return err
}

// ParseRepositoryName returns owner and project name for given GitHub repository.
func ParseRepositoryName(fullName string) (string, string) {
	fields := strings.SplitN(fullName, "/", 2)
	return fields[0], fields[1]
}

func repositoryFromGithub(githubRepo *api.Repository) *Repository {
	return &Repository{
		FullName:    *githubRepo.FullName,
		Description: *githubRepo.Description,
		Master:      *githubRepo.DefaultBranch,
		HTMLURL:     *githubRepo.HTMLURL,
		GitURL:      *githubRepo.GitURL,
	}
}

func commitFromGithub(githubCommit *api.Commit) *Commit {
	return &Commit{
		SHA:     *githubCommit.SHA,
		Message: *githubCommit.Message,
		Author:  *githubCommit.Committer.Name,
		Date:    *githubCommit.Committer.Date,
	}
}
