package git_test

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/andrewslotin/doppelganger/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

/* ************ Tests objects ************ */

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

func (cmd *commandMock) LastCommit(fullPath string) (git.Commit, error) {
	args := cmd.Mock.Called(fullPath)
	return args.Get(0).(git.Commit), args.Error(1)
}

func (cmd *commandMock) CloneMirror(gitURL, fullPath string) error {
	args := cmd.Mock.Called(gitURL, fullPath)
	return args.Error(0)
}

func (cmd *commandMock) UpdateRemote(fullPath string) error {
	args := cmd.Mock.Called(fullPath)
	return args.Error(0)
}

/* **************** Tests **************** */

func TestMirroredRepositories_All(t *testing.T) {
	mirrorsDir, teardown, err := setupMirrorsDir()
	require.NoError(t, err)
	defer teardown()

	reposWithBranches := map[string]string{
		"a":      "staging",
		"b/b1":   "master",
		"b/b2/z": "master",
		"c":      " production",
	}

	cmd := &commandMock{}

	cmd.On("IsRepository", mirrorsDir).Return(false)
	cmd.On("IsRepository", filepath.Join(mirrorsDir, "b")).Return(false)
	cmd.On("IsRepository", filepath.Join(mirrorsDir, "b/b2")).Return(false)

	lastCommit := git.Commit{
		SHA:       "abc123",
		Author:    "Jon Doe",
		Committer: "Doppel Ganger",
		Message:   "HI MOM",
		Date:      time.Now().UTC().Truncate(time.Second).Add(-10 * time.Hour),
	}
	for repoName, masterBranch := range reposWithBranches {
		path := filepath.Join(mirrorsDir, repoName)
		os.MkdirAll(path, 0755)

		cmd.On("IsRepository", path).Return(true)
		cmd.On("CurrentBranch", path).Return(masterBranch)
		cmd.On("LastCommit", path).Return(lastCommit, nil)
	}

	mirroredRepos := git.NewMirroredRepositories(mirrorsDir, cmd)
	mirrors, err := mirroredRepos.All()
	require.NoError(t, err)
	cmd.AssertExpectations(t)

	if assert.Len(t, mirrors, 4) {
		for _, repo := range mirrors {
			assert.Equal(t, reposWithBranches[repo.FullName], repo.Master)
			if assert.NotNil(t, repo.LatestMasterCommit) {
				assert.Equal(t, lastCommit.SHA, repo.LatestMasterCommit.SHA)
				assert.Equal(t, lastCommit.Author, repo.LatestMasterCommit.Author)
				assert.Equal(t, lastCommit.Committer, repo.LatestMasterCommit.Committer)
				assert.Equal(t, lastCommit.Message, repo.LatestMasterCommit.Message)
				assert.True(t, repo.LatestMasterCommit.Date.Equal(lastCommit.Date))
			}
		}
	}
}

func TestMirroredRepositories_Get_MirrorExists(t *testing.T) {
	mirrorsDir, teardown, err := setupMirrorsDir()
	require.NoError(t, err)
	defer teardown()

	cmd := &commandMock{}

	mirroredRepoPath := path.Join(mirrorsDir, "a", "b")
	lastCommit := git.Commit{
		SHA:       "abc123",
		Author:    "Jon Doe",
		Committer: "Doppel Ganger",
		Message:   "HI MOM",
		Date:      time.Now().Add(-5 * time.Minute).Truncate(time.Second),
	}

	cmd.On("IsRepository", mirroredRepoPath).Return(true)
	cmd.On("CurrentBranch", mirroredRepoPath).Return("production")
	cmd.On("LastCommit", mirroredRepoPath).Return(lastCommit, nil)

	mirroredRepos := git.NewMirroredRepositories(mirrorsDir, cmd)
	repo, err := mirroredRepos.Get("a/b")
	require.NoError(t, err)

	if cmd.AssertExpectations(t) {
		assert.Equal(t, "a/b", repo.FullName)
		assert.Equal(t, "production", repo.Master)

		if commit := repo.LatestMasterCommit; assert.NotNil(t, commit) {
			assert.Equal(t, lastCommit.SHA, commit.SHA)
			assert.Equal(t, lastCommit.Author, commit.Author)
			assert.Equal(t, lastCommit.Committer, commit.Committer)
			assert.Equal(t, lastCommit.Message, commit.Message)
			assert.True(t, commit.Date.Equal(lastCommit.Date), "Expected %s, got %s", lastCommit.Date, commit.Date)
		}
	}
}

func TestMirroredRepositories_Get_NotMirrored(t *testing.T) {
	mirrorsDir, teardown, err := setupMirrorsDir()
	require.NoError(t, err)
	defer teardown()

	cmd := &commandMock{}
	cmd.On("IsRepository", path.Join(mirrorsDir, "a", "b")).Return(false)

	mirroredRepos := git.NewMirroredRepositories(mirrorsDir, cmd)
	_, err = mirroredRepos.Get("a/b")

	cmd.AssertExpectations(t)
	assert.Equal(t, err, git.ErrorNotMirrored)
}

func TestMirroredRepositories_Create(t *testing.T) {
	mirrorsDir, teardown, err := setupMirrorsDir()
	require.NoError(t, err)
	defer teardown()

	cmd := &commandMock{}
	cmd.On("CloneMirror", "git@doppelganger:a/b", path.Join(mirrorsDir, "a", "b")).Return(nil)

	mirroredRepos := git.NewMirroredRepositories(mirrorsDir, cmd)
	require.NoError(t, mirroredRepos.Create("a/b", "git@doppelganger:a/b"))

	cmd.AssertExpectations(t)
}

func TestMirroredRepositories_Create_DirExists(t *testing.T) {
	mirrorsDir, teardown, err := setupMirrorsDir()
	require.NoError(t, err)
	defer teardown()

	mirroredRepoPath := path.Join(mirrorsDir, "a", "b")
	require.NoError(t, os.MkdirAll(mirroredRepoPath, 0755))

	cmd := &commandMock{}
	cmd.On("CloneMirror", "git@doppelganger:a/b", mirroredRepoPath).Return(nil)

	mirroredRepos := git.NewMirroredRepositories(mirrorsDir, cmd)
	require.NoError(t, mirroredRepos.Create("a/b", "git@doppelganger:a/b"))

	cmd.AssertExpectations(t)
}

func TestMirroredRepositories_Update(t *testing.T) {
	mirrorsDir, teardown, err := setupMirrorsDir()
	require.NoError(t, err)
	defer teardown()

	cmd := &commandMock{}
	cmd.On("UpdateRemote", path.Join(mirrorsDir, "a", "b")).Return(nil)

	mirroredRepos := git.NewMirroredRepositories(mirrorsDir, cmd)
	require.NoError(t, mirroredRepos.Update("a/b"))

	cmd.AssertExpectations(t)
}

func setupMirrorsDir() (mirrorsPath string, teardownFn func(), err error) {
	tmpDir, err := ioutil.TempDir(os.TempDir(), "doppelganger")
	if err != nil {
		return "", nil, err
	}

	return tmpDir, func() { os.RemoveAll(tmpDir) }, nil
}
