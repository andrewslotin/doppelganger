package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"golang.org/x/net/context"
)

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
