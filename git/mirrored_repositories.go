package git

import (
	"errors"
	"io/ioutil"
	"path/filepath"
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
func (service *MirroredRepositories) All() ([]*Repository, error) {
	return service.findGitRepos("")
}

// Get searches for a git repository in <mirrorPath>/<fullName> and returns the name of its name, master branch
// and lastest commit. If specified directory does not exist or not a git repository ErrorNotMirrored is returned.
func (service *MirroredRepositories) Get(fullName string) (*Repository, error) {
	if !service.cmd.IsRepository(service.resolveMirrorPath(fullName)) {
		return nil, ErrorNotMirrored
	}

	repo := service.repositoryFromDir(fullName)
	repo.LatestMasterCommit = service.commitFromDir(fullName)

	return repo, nil
}

// Create creates a local mirror of remote repository from gitURL by calling "git --mirror <gitURL> <fullName>".
func (service *MirroredRepositories) Create(fullName, gitURL string) error {
	return service.cmd.CloneMirror(gitURL, service.resolveMirrorPath(fullName))
}

// Update downloads latest changes from remote repository into a local mirror discarding any changes that were pushed
// to mirror only. Update calls "git remote update" in <mirrorPath>/<fullName>.
func (service *MirroredRepositories) Update(fullName string) error {
	return service.cmd.UpdateRemote(service.resolveMirrorPath(fullName))
}

func (service *MirroredRepositories) findGitRepos(path string) ([]*Repository, error) {
	if service.cmd.IsRepository(service.resolveMirrorPath(path)) {
		return []*Repository{service.repositoryFromDir(path)}, nil
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

		r, err := service.findGitRepos(filepath.Join(path, entry.Name()))
		if err != nil {
			return nil, err
		}
		repos = append(repos, r...)
	}

	return repos, nil
}

func (service *MirroredRepositories) repositoryFromDir(path string) *Repository {
	return &Repository{
		FullName: path,
		Master:   service.cmd.CurrentBranch(service.resolveMirrorPath(path)),
	}
}

func (service *MirroredRepositories) commitFromDir(path string) *Commit {
	rev, author, message, createdAt, err := service.cmd.LastCommit(service.resolveMirrorPath(path))
	if err != nil {
		return nil
	}

	return &Commit{
		SHA:     rev,
		Author:  author,
		Message: message,
		Date:    createdAt,
	}
}

func (service *MirroredRepositories) resolveMirrorPath(path string) string {
	return filepath.Join(service.mirrorPath, path)
}
