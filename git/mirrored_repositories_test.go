package git

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type commandMock struct {
	mock.Mock
}

func (cmd *commandMock) IsRepository(fullPath string) bool {
	args := cmd.Mock.Called(fullPath)
	return args.Bool(0)
}

func (cmd *commandMock) CurrentBranch(fullPath string) string {
	args := cmd.Mock.Called(fullPath)
	return args.String(0)
}

func (cmd *commandMock) LastCommit(fullPath string) (string, string, string, time.Time, error) {
	args := cmd.Mock.Called(fullPath)

	var createdAt time.Time
	if t, err := time.Parse("2006-01-02 15:04:05", args.String(3)); err == nil {
		createdAt = t
	}

	return args.String(0), args.String(1), args.String(2), createdAt, args.Error(4)
}

func (cmd *commandMock) CloneMirror(gitURL, fullPath string) error {
	args := cmd.Mock.Called(gitURL, fullPath)
	return args.Error(0)
}

func (cmd *commandMock) UpdateRemote(fullPath string) error {
	args := cmd.Mock.Called(fullPath)
	return args.Error(0)
}

func TestMirroredRepositoriesAll(t *testing.T) {
	mirrorPath, err := ioutil.TempDir("", "mirroredReposXXX")
	require.NoError(t, err)
	defer os.RemoveAll(mirrorPath)

	reposWithBranches := map[string]string{
		"a":      "staging",
		"b/b1":   "master",
		"b/b2/z": "master",
		"c":      " production",
	}

	cmd := &commandMock{}

	cmd.On("IsRepository", mirrorPath).Return(false)
	cmd.On("IsRepository", filepath.Join(mirrorPath, "b")).Return(false)
	cmd.On("IsRepository", filepath.Join(mirrorPath, "b/b2")).Return(false)

	for repoName, masterBranch := range reposWithBranches {
		path := filepath.Join(mirrorPath, repoName)
		os.MkdirAll(path, 0755)

		cmd.On("IsRepository", path).Return(true)
		cmd.On("CurrentBranch", path).Return(masterBranch)
	}

	mirroredRepos := NewMirroredRepositories(mirrorPath, cmd)
	mirrors, err := mirroredRepos.All()
	require.NoError(t, err)
	cmd.AssertExpectations(t)

	if assert.Len(t, mirrors, 4) {
		for _, repo := range mirrors {
			assert.Equal(t, reposWithBranches[repo.FullName], repo.Master)
		}
	}
}
