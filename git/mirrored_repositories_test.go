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

func (cmd *commandMock) LastCommit(fullPath string) (string, string, string, time.Time, error) {
	args := cmd.Mock.Called(fullPath)

	var createdAt time.Time
	if t, err := time.Parse(git.GitCommandDateLayout, args.String(3)); err == nil {
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

	latestCommit := git.Commit{SHA: "abc123", Author: "Jon Doe", Message: "HI MOM", Date: time.Now().UTC().Truncate(time.Second).Add(-10 * time.Hour)}
	for repoName, masterBranch := range reposWithBranches {
		path := filepath.Join(mirrorsDir, repoName)
		os.MkdirAll(path, 0755)

		cmd.On("IsRepository", path).Return(true)
		cmd.On("CurrentBranch", path).Return(masterBranch)
		cmd.On("LastCommit", path).Return(
			latestCommit.SHA,
			latestCommit.Author,
			latestCommit.Message,
			latestCommit.Date.Format(git.GitCommandDateLayout),
			nil,
		)
	}

	mirroredRepos := git.NewMirroredRepositories(mirrorsDir, cmd)
	mirrors, err := mirroredRepos.All()
	require.NoError(t, err)
	cmd.AssertExpectations(t)

	if assert.Len(t, mirrors, 4) {
		for _, repo := range mirrors {
			assert.Equal(t, reposWithBranches[repo.FullName], repo.Master)
			if assert.NotNil(t, repo.LatestMasterCommit) {
				assert.Equal(t, latestCommit.SHA, repo.LatestMasterCommit.SHA)
				assert.Equal(t, latestCommit.Author, repo.LatestMasterCommit.Author)
				assert.Equal(t, latestCommit.Message, repo.LatestMasterCommit.Message)
				assert.True(t, repo.LatestMasterCommit.Date.Equal(latestCommit.Date))
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
	cmd.On("IsRepository", mirroredRepoPath).Return(true)
	cmd.On("CurrentBranch", mirroredRepoPath).Return("production")
	cmd.On("LastCommit", mirroredRepoPath).Return("abc123", "Jon Doe", "HI MOM", "2016-04-23T16:12:39+0000", nil)

	mirroredRepos := git.NewMirroredRepositories(mirrorsDir, cmd)
	repo, err := mirroredRepos.Get("a/b")
	require.NoError(t, err)

	if cmd.AssertExpectations(t) {
		assert.Equal(t, "a/b", repo.FullName)
		assert.Equal(t, "production", repo.Master)

		if commit := repo.LatestMasterCommit; assert.NotNil(t, commit) {
			assert.Equal(t, "abc123", commit.SHA)
			assert.Equal(t, "Jon Doe", commit.Author)
			assert.Equal(t, "HI MOM", commit.Message)
			assert.True(t, commit.Date.Equal(time.Date(2016, 4, 23, 16, 12, 39, 0, time.UTC)))
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
