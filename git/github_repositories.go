package git

import (
	"errors"
	"log"
	"net/http"
	"strings"

	api "github.com/google/go-github/github"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

type tokenContextKey struct{}
type clientContextKey struct{}

var (
	// ErrorNotFound is returned by Get method if specified repository cannot be found.
	ErrorNotFound = errors.New("not found")
	// GithubToken is a context.Context key for Github auth token.
	GithubToken tokenContextKey
	httpClient  clientContextKey
)

// GithubRepositories is a type intended to list and lookup GitHub repositories as well as setting webhooks.
type GithubRepositories struct {
	client *api.Client
}

// NewGithubRepositories creates and initializes a new instance of GithubRepositories.
// Context is expected to have GitHub auth token set with git.GithubToken as a key.
// Provided token is used to authorize requests to GitHub API and must be given "repo"
// or "public_repo" permissions.
// If token is not set or is empty an error will be returned.
func NewGithubRepositories(ctx context.Context) (*GithubRepositories, error) {
	token, ok := ctx.Value(GithubToken).(string)
	if !ok || token == "" {
		return nil, errors.New("missing auth token")
	}

	if c, ok := ctx.Value(httpClient).(*api.Client); ok {
		return &GithubRepositories{
			client: c,
		}, nil
	}

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: token,
	})
	oauthClient := oauth2.NewClient(ctx, tokenSource)

	return &GithubRepositories{
		client: api.NewClient(oauthClient),
	}, nil
}

// All returns a list of GitHub repositories accessible with provided API token.
func (service *GithubRepositories) All() ([]*Repository, error) {
	opts := &api.RepositoryListOptions{
		ListOptions: api.ListOptions{
			PerPage: 50,
		},
	}

	var allRepos []*Repository

	paginatedRepos := make([]*Repository, 0, opts.ListOptions.PerPage)
	for {
		githubRepos, response, err := service.client.Repositories.List("", opts)
		if err != nil {
			return nil, err
		}

		for _, githubRepo := range githubRepos {
			if githubRepo.FullName == nil {
				log.Printf("[WARN] excluding GitHub repository without full_name %v", githubRepo)
				continue
			}

			if githubRepo.GitURL == nil {
				log.Printf("[WARN] excluding GitHub repository without git_url %v", githubRepo)
				continue
			}

			repo := &Repository{
				FullName: *githubRepo.FullName,
				Master:   "master",
			}

			if githubRepo.Description != nil {
				repo.Description = *githubRepo.Description
			}

			if githubRepo.DefaultBranch != nil {
				repo.Master = *githubRepo.DefaultBranch
			}

			if githubRepo.HTMLURL != nil {
				repo.HTMLURL = *githubRepo.HTMLURL
			}

			paginatedRepos = append(paginatedRepos, repo)
		}

		allRepos = append(allRepos, paginatedRepos[0:len(paginatedRepos)]...)
		githubRepos = githubRepos[:0]

		if response.NextPage == 0 {
			break
		}
		opts.ListOptions.Page = response.NextPage
	}

	return allRepos, nil
}

// Get retireves GitHub repositories details and returns an instance of Repository containing last commit information.
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

// Track sets up "push" event GitHub webhook to be sent to callbackURL.
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
	if err != nil {
		errorResponse, ok := err.(*api.ErrorResponse)
		if !ok || errorResponse.Message != "Validation Failed" {
			return err
		}

		if len(errorResponse.Errors) != 1 || errorResponse.Errors[0].Code != "custom" {
			return err
		}

		if service.checkPushWebhookExists(owner, repo, cbURL) {
			log.Printf("push webhook to %s for %s/%s has already been set up", cbURL, owner, repo)
			return nil
		}
	}

	return err
}

func (service *GithubRepositories) checkPushWebhookExists(owner, repo, cbURL string) bool {
	opts := &api.ListOptions{
		PerPage: 50,
	}

	for {
		hooks, response, err := service.client.Repositories.ListHooks(owner, repo, opts)
		if err != nil {
			log.Printf("[WARN] failed to get %s/%s webhooks: %s", owner, repo, err)
			return false
		}

		for _, hook := range hooks {
			if hook.Config["url"] != cbURL || len(hook.Events) == 0 {
				continue
			}

			for _, event := range hook.Events {
				if event == "push" {
					return true
				}
			}
		}

		if response.NextPage == 0 {
			break
		}

		opts.Page = response.NextPage
	}

	return false
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
