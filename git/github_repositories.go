package git

import (
	"errors"
	"log"
	"net/http"
	"strings"

	api "github.com/google/go-github/github"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"

	"github.com/andrewslotin/doppelganger/git/internal"
)

var (
	// ErrorNotFound is returned by Get method if specified repository cannot be found.
	ErrorNotFound = errors.New("not found")
	// GithubToken is a context.Context key for Github auth token.
	GithubToken internal.TokenContextKey
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

	if c, ok := ctx.Value(internal.HttpClient).(*api.Client); ok {
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
func (service *GithubRepositories) All(ctx context.Context) ([]*Repository, error) {
	opts := &api.RepositoryListOptions{
		ListOptions: api.ListOptions{
			PerPage: 50,
		},
	}

	var allRepos []*Repository

	paginatedRepos := make([]*Repository, 0, opts.ListOptions.PerPage)
	for {
		githubRepos, response, err := service.client.Repositories.List(ctx, "", opts)
		if err != nil {
			return nil, err
		}

		for _, githubRepo := range githubRepos {
			if githubRepo.FullName == nil {
				log.Printf("[WARN] excluding GitHub repository without full_name %v", githubRepo)
				continue
			}

			if githubRepo.SSHURL == nil {
				log.Printf("[WARN] excluding GitHub repository without ssh_url %v", githubRepo)
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

		allRepos = append(allRepos, paginatedRepos[0:]...)
		paginatedRepos = paginatedRepos[:0]

		if response.NextPage == 0 {
			break
		}
		opts.ListOptions.Page = response.NextPage
	}

	return allRepos, nil
}

// Get retireves GitHub repositories details and returns an instance of Repository containing last commit information.
func (service *GithubRepositories) Get(ctx context.Context, fullName string) (*Repository, error) {
	repoOwner, repoName := ParseRepositoryName(fullName)

	githubRepo, response, err := service.client.Repositories.Get(ctx, repoOwner, repoName)
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			return nil, ErrorNotFound
		}

		return nil, err
	}

	masterBranch, _, err := service.client.Repositories.GetBranch(ctx, repoOwner, repoName, *githubRepo.DefaultBranch)
	if err != nil {
		return nil, err
	}

	lastCommit, _, err := service.client.Git.GetCommit(ctx, repoOwner, repoName, *masterBranch.Commit.SHA)
	if err != nil {
		return nil, err
	}

	repo := repositoryFromGithub(githubRepo)
	repo.LatestMasterCommit = commitFromGithub(lastCommit)

	return repo, nil
}

// Track sets up "push" event GitHub webhook to be sent to callbackURL.
func (service *GithubRepositories) Track(ctx context.Context, fullName, callbackURL string) error {
	owner, name := ParseRepositoryName(fullName)
	return service.registerPushWebhook(ctx, owner, name, callbackURL)
}

func (service *GithubRepositories) registerPushWebhook(ctx context.Context, owner, repo, cbURL string) error {
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

	_, _, err := service.client.Repositories.CreateHook(ctx, owner, repo, hook)
	if err != nil {
		errorResponse, ok := err.(*api.ErrorResponse)
		if !ok || errorResponse.Message != "Validation Failed" {
			return err
		}

		if len(errorResponse.Errors) != 1 || errorResponse.Errors[0].Code != "custom" {
			return err
		}

		if service.checkPushWebhookExists(ctx, owner, repo, cbURL) {
			log.Printf("push webhook to %s for %s/%s has already been set up", cbURL, owner, repo)
			return nil
		}
	}

	return err
}

func (service *GithubRepositories) checkPushWebhookExists(ctx context.Context, owner, repo, cbURL string) bool {
	opts := &api.ListOptions{
		PerPage: 50,
	}

	for {
		hooks, response, err := service.client.Repositories.ListHooks(ctx, owner, repo, opts)
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
	repo := &Repository{
		FullName:    *githubRepo.FullName,
		Description: *githubRepo.Description,
		Master:      *githubRepo.DefaultBranch,
		HTMLURL:     *githubRepo.HTMLURL,
		GitURL:      *githubRepo.GitURL,
	}

	// Use git+ssh to clone private repos
	if githubRepo.Private != nil && *githubRepo.Private {
		repo.GitURL = *githubRepo.SSHURL
	}

	return repo
}

func commitFromGithub(githubCommit *api.Commit) *Commit {
	return &Commit{
		SHA:       *githubCommit.SHA,
		Message:   *githubCommit.Message,
		Author:    *githubCommit.Author.Name,
		Committer: *githubCommit.Committer.Name,
		Date:      *githubCommit.Committer.Date,
	}
}
