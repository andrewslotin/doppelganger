package git

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-github/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"golang.org/x/net/context"
)

func TestParseRepositoryName(t *testing.T) {
	owner, repo := ParseRepositoryName("test/me")
	assert.Equal(t, owner, "test")
	assert.Equal(t, repo, "me")
}

func TestNewGithubRepositories_WithToken(t *testing.T) {
	ctx := context.WithValue(context.Background(), GithubToken, "secret_token")

	r, err := NewGithubRepositories(ctx)
	require.NoError(t, err, "Expected NewGithubRepositories to succeed")
	assert.NotNil(t, r, "Expected NewGithubRepositories to return new instance")
}

func TestNewGithubRepositories_NoToken(t *testing.T) {
	_, err := NewGithubRepositories(context.Background())
	assert.Error(t, err, "Expected NewGithubRepositories to return an error")
}

func TestNewGithubRepositories_EmptyToken(t *testing.T) {
	ctx := context.WithValue(context.Background(), GithubToken, "")

	_, err := NewGithubRepositories(ctx)
	assert.Error(t, err, "Expected NewGithubRepositories to return an error")
}

func TestGithubRepositoriesAll_SingleRepository_DefaultFields(t *testing.T) {
	ctx, mux, teardown := setup()
	defer teardown()

	mux.HandleFunc("/user/repos", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{
                    "full_name": "user1/repo1",
                    "git_url": "git@github.com:user1/repo1.git"
                }]`)
	})

	githubRepos, err := NewGithubRepositories(ctx)
	require.NoError(t, err)

	repos, err := githubRepos.All()
	require.NoError(t, err)

	if assert.Len(t, repos, 1) {
		repo := repos[0]
		assert.Equal(t, repo.FullName, "user1/repo1")
		assert.Equal(t, repo.Master, "master")
		assert.Empty(t, repo.Description)
		assert.Empty(t, repo.GitURL)
		assert.Empty(t, repo.HTMLURL)
	}
}

func TestGithubRepositoriesAll_SingleRepository_AllFields(t *testing.T) {
	ctx, mux, teardown := setup()
	defer teardown()

	mux.HandleFunc("/user/repos", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{
                    "full_name": "user1/repo1",
                    "description": "Repo1",
                    "default_branch": "production",
                    "git_url": "git@github.com:user1/repo1.git",
                    "html_url": "https://github.com/user1/repo1"
                }]`)
	})

	githubRepos, err := NewGithubRepositories(ctx)
	require.NoError(t, err)

	repos, err := githubRepos.All()
	require.NoError(t, err)

	if assert.Len(t, repos, 1) {
		repo := repos[0]
		assert.Equal(t, repo.FullName, "user1/repo1")
		assert.Equal(t, repo.Description, "Repo1")
		assert.Equal(t, repo.Master, "production")
		assert.Empty(t, repo.GitURL)
	}
}

func TestGithubRepositoriesAll_MultipleRepositories(t *testing.T) {
	ctx, mux, teardown := setup()
	defer teardown()

	mux.HandleFunc("/user/repos", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{
                    "full_name": "user1/repo1",
                    "git_url": "https://github.com/user1/repo1"
                },{
                    "full_name": "user2/repo2",
                    "git_url": "https://github.com/user2/repo2"
                }]`)
	})

	githubRepos, err := NewGithubRepositories(ctx)
	require.NoError(t, err)

	repos, err := githubRepos.All()
	require.NoError(t, err)

	assert.Len(t, repos, 2)
}

func TestGithubRepositoriesAll_SkipWithoutGitURL(t *testing.T) {
	ctx, mux, teardown := setup()
	defer teardown()

	mux.HandleFunc("/user/repos", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{
                    "full_name": "user1/repo1"
                },{
                    "full_name": "user2/repo2",
                    "git_url": "git:git@github.com:user2/repo2.git"
                }]`)
	})

	githubRepos, err := NewGithubRepositories(ctx)
	require.NoError(t, err)

	repos, err := githubRepos.All()
	require.NoError(t, err)

	if assert.Len(t, repos, 1, "Should exclude one repository without git_url") {
		assert.Equal(t, repos[0].FullName, "user2/repo2")
	}
}

func TestGithubRepositoriesAll_SkipWithoutFullName(t *testing.T) {
	ctx, mux, teardown := setup()
	defer teardown()

	mux.HandleFunc("/user/repos", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{
                    "full_name": "user1/repo1",
                    "git_url": "git:git@github.com:user1/repo1.git"
                },{
                    "git_url": "git:git@github.com:user2/repo2.git"
                }]`)
	})

	githubRepos, err := NewGithubRepositories(ctx)
	require.NoError(t, err)

	repos, err := githubRepos.All()
	require.NoError(t, err)

	if assert.Len(t, repos, 1, "Should exclude one repository without full_name") {
		assert.Equal(t, repos[0].FullName, "user1/repo1")
	}
}

func TestGithubRepositoriesAll_HandlePagination(t *testing.T) {
	ctx, mux, teardown := setup()
	defer teardown()

	mux.HandleFunc("/user/repos", func(w http.ResponseWriter, r *http.Request) {
		if page := r.FormValue("page"); page == "" || page == "1" {
			perPage := r.FormValue("per_page")
			if perPage == "" {
				perPage = "30"
			}

			w.Header().Set("Link", fmt.Sprintf(`<https://api.github.com/user/repos?page=2&per_page=%s>; rel="next"`, perPage))
		}

		fmt.Fprint(w, `[{"full_name": "user1/repo1","git_url": "git:git@github.com:user1/repo1.git"}]`)
	})

	githubRepos, err := NewGithubRepositories(ctx)
	require.NoError(t, err)

	repos, err := githubRepos.All()
	require.NoError(t, err)

	assert.Len(t, repos, 2)
}

func TestGithubRepositoriesGet_RepositoryExists(t *testing.T) {
	ctx, mux, teardown := setup()
	defer teardown()

	// Get repo
	mux.HandleFunc("/repos/user1/repo1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
		    "full_name": "user1/repo1",
		    "description":
		    "Test repo",
		    "default_branch":"production",
		    "git_url":"git:git@github.com:user1/repo1.git",
		    "html_url": "https://github.com/user1/repo1"
		}`)
	})
	// Get branch
	mux.HandleFunc("/repos/user1/repo1/branches/production", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"name":"production","commit":{"sha":"abc123"}}`)
	})
	// Get git commit
	mux.HandleFunc("/repos/user1/repo1/git/commits/abc123", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
		    "author":{"name":"Jon Doe","date":"2011-04-14T16:00:49Z"},
		    "committer":{"name":"Doppel Ganger","date":"2016-04-14T16:00:00Z"},
		    "message":"Commit message",
		    "sha":"abc123"
		}`)
	})

	githubRepos, err := NewGithubRepositories(ctx)
	require.NoError(t, err)

	repo, err := githubRepos.Get("user1/repo1")
	require.NoError(t, err)

	assert.Equal(t, repo.FullName, "user1/repo1")
	assert.Equal(t, repo.GitURL, "git:git@github.com:user1/repo1.git")
}

func TestGithubRepositoriesGet_NotFound(t *testing.T) {
	ctx, _, teardown := setup()
	defer teardown()

	githubRepos, err := NewGithubRepositories(ctx)
	require.NoError(t, err)

	_, err = githubRepos.Get("user1/repo1")
	assert.Equal(t, err, ErrorNotFound)
}

func setup() (ctx context.Context, mux *http.ServeMux, teardownFn func()) {
	mux = http.NewServeMux()
	server := httptest.NewServer(mux)

	client := github.NewClient(nil)
	url, _ := url.Parse(server.URL)
	client.BaseURL = url

	ctx = context.Background()
	ctx = context.WithValue(ctx, GithubToken, "secret_token")
	ctx = context.WithValue(ctx, httpClient, client)

	return ctx, mux, server.Close
}
