package git

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/net/context"
)

// DefaultMaster is a default name for master branch.
const DefaultMaster = "master"

// ErrorNotMirrored is an error returned by Get if given repository does not exist.
var ErrorNotMirrored = errors.New("mirror not found")

// MirroredRepositories is a type that is intended for maintaining local Git repository mirrors
// located under the mirrorPath directory.
type MirroredRepositories struct {
	cmd        Command
	mirrorPath string
}

// NewMirroredRepositories creates and initializes an instance of MirroredRepositories reading and creating
// repositories under path directory.
func NewMirroredRepositories(path string, gitCommand Command) *MirroredRepositories {
	return &MirroredRepositories{
		cmd:        gitCommand,
		mirrorPath: path,
	}
}

// All recursively searches and returns a list of repositories under mirrorPath. Unlike Get, All returns
// only basic information about Git repository, such as its name and the name of master branch.
func (service *MirroredRepositories) All(ctx context.Context) ([]*Repository, error) {
	return service.findGitRepos(ctx, "")
}

// Get searches for a git repository in <mirrorPath>/<fullName> and returns the name of its name, master branch
// and lastest commit. If specified directory does not exist or not a git repository ErrorNotMirrored is returned.
func (service *MirroredRepositories) Get(ctx context.Context, fullName string) (*Repository, error) {
	if !service.cmd.IsRepository(ctx, service.resolveMirrorPath(fullName)) {
		return nil, ErrorNotMirrored
	}

	repo := service.repositoryFromDir(ctx, fullName)
	repo.LatestMasterCommit = service.commitFromDir(ctx, fullName)

	return repo, nil
}

// Create creates a local mirror of remote repository from gitURL by calling "git --mirror <gitURL> <fullName>".
func (service *MirroredRepositories) Create(ctx context.Context, fullName, gitURL string) error {
	fullPath := service.resolveMirrorPath(fullName)

	if _, err := os.Stat(fullPath); err == nil {
		log.Printf("[WARN] %s already exists, removing", fullPath)
		if err := os.RemoveAll(fullPath); err != nil {
			return fmt.Errorf("failed to remove an existing file/directory %s: %s", fullPath, err)
		}
	}

	return service.cmd.CloneMirror(ctx, gitURL, fullPath)
}

// Update downloads latest changes from remote repository into a local mirror discarding any changes that were pushed
// to mirror only. Update calls "git remote update" in <mirrorPath>/<fullName>.
func (service *MirroredRepositories) Update(ctx context.Context, fullName string) error {
	return service.cmd.UpdateRemote(ctx, service.resolveMirrorPath(fullName))
}

func (service *MirroredRepositories) findGitRepos(ctx context.Context, path string) ([]*Repository, error) {
	if service.cmd.IsRepository(ctx, service.resolveMirrorPath(path)) {
		return []*Repository{service.repositoryFromDir(ctx, path)}, nil
	}

	entries, err := ioutil.ReadDir(filepath.Join(service.mirrorPath, path))
	if err != nil {
		return nil, err
	}

	var repos []*Repository
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		r, err := service.findGitRepos(ctx, filepath.Join(path, entry.Name()))
		if err != nil {
			return nil, err
		}
		repos = append(repos, r...)
	}

	return repos, nil
}

func (service *MirroredRepositories) repositoryFromDir(ctx context.Context, path string) *Repository {
	return &Repository{
		FullName:           path,
		Master:             service.cmd.CurrentBranch(ctx, service.resolveMirrorPath(path)),
		LatestMasterCommit: service.commitFromDir(ctx, path),
	}
}

func (service *MirroredRepositories) commitFromDir(ctx context.Context, path string) *Commit {
	commit, err := service.cmd.LastCommit(ctx, service.resolveMirrorPath(path))
	if err != nil {
		return nil
	}

	return &commit
}

func (service *MirroredRepositories) resolveMirrorPath(path string) string {
	return filepath.Join(service.mirrorPath, path)
}
