package git

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const DefaultMaster = "master"

var (
	gitCmd string

	ErrorNotMirrored = errors.New("mirror not found")
)

type MirroredRepositories struct {
	mirrorPath string
}

func init() {
	var err error
	if gitCmd, err = exec.LookPath("git"); err != nil {
		log.Fatal("git is not found in PATH")
	}
}

func NewMirroredRepositories(path string) *MirroredRepositories {
	return &MirroredRepositories{
		mirrorPath: path,
	}
}

func (service *MirroredRepositories) All() ([]*Repository, error) {
	return service.findGitRepos("")
}

func (service *MirroredRepositories) Get(fullName string) (*Repository, error) {
	if !service.checkDirIsRepository(fullName) {
		return nil, ErrorNotMirrored
	}

	repo := service.repositoryFromDir(fullName)
	repo.LatestMasterCommit = service.commitFromDir(fullName)

	return repo, nil
}

func (service *MirroredRepositories) Create(fullName, gitURL string) error {
	return service.cloneMirror(gitURL, fullName)
}

func (service *MirroredRepositories) Update(fullName string) error {
	return service.updateRemote(fullName)
}

func (service *MirroredRepositories) checkDirIsRepository(path string) bool {
	fullPath := filepath.Join(service.mirrorPath, path)
	if fileInfo, err := os.Stat(fullPath); err != nil {
		if os.IsNotExist(err) {
			return false
		}

		log.Printf("[WARN] failed to stat %s (%s)", fullPath, err)
		return false
	} else if !fileInfo.IsDir() {
		return false
	}

	output, err := service.execGitCommand(path, "rev-parse", "--is-inside-git-dir")
	if err != nil {
		log.Printf("[WARN] git rev-parse --is-inside-git-dir returned error %s for %s (%s)", err, path, string(output))
		return false
	}

	switch string(output) {
	case "true":
		return true
	case "false":
		return false
	default:
		log.Printf("[WARN] git rev-parse --is-inside-git-dir returned unexpected output for %s: %q", path, string(output))
		return false
	}
}

func (service *MirroredRepositories) repositoryFromDir(path string) *Repository {
	return &Repository{
		FullName: path,
		Master:   service.currentBranch(path),
	}
}

func (service *MirroredRepositories) commitFromDir(path string) *Commit {
	output, err := service.execGitCommand(path, "log", "-n", "1", "--pretty=%H\n%cn\n%cd\n%s", "--date=format:%FT%T%z")
	if err != nil {
		log.Printf("[WARN] git log returned error %s for %s (%s)", err, path, string(output))
		return nil
	}

	lines := strings.SplitN(string(output), "\n", 4)
	if len(lines) < 4 {
		log.Printf("[WARN] unexpected output from git log for %s (%s)", path, string(output))
		return nil
	}

	commit := &Commit{
		SHA:     lines[0],
		Author:  lines[1],
		Message: lines[3],
	}

	commitDate, err := time.Parse("2006-01-02T15:04:05Z0700", lines[2])
	if err != nil {
		log.Printf("[WARN] unexpected date format from git log for %s (%s)", path, lines[2])
	} else {
		commit.Date = commitDate
	}

	return commit
}

func (service *MirroredRepositories) findGitRepos(path string) ([]*Repository, error) {
	if service.checkDirIsRepository(path) {
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

func (service *MirroredRepositories) currentBranch(path string) string {
	refName, err := service.execGitCommand(path, "symbolic-ref", "HEAD")
	if err != nil {
		log.Printf("[WARN] git symbolic-ref HEAD returned error %s for %s (%s)", err, path, string(refName))
		return DefaultMaster
	}

	if !bytes.HasPrefix(refName, []byte("refs/heads/")) {
		log.Printf("[WARN] unexpected reference name for %s (%q)", path, refName)
		return DefaultMaster
	}

	return string(bytes.TrimPrefix(refName, []byte("refs/heads/")))
}

func (service *MirroredRepositories) cloneMirror(gitURL, path string) error {
	path, projectName := filepath.Dir(path), filepath.Base(path)
	fullPath := filepath.Join(service.mirrorPath, path)

	if err := os.MkdirAll(fullPath, 0755); err != nil {
		log.Printf("failed to create %s (%s)", fullPath, err)
		return fmt.Errorf("failed to clone %s to %s", gitURL, path)
	}

	output, err := service.execGitCommand(path, "clone", "--mirror", gitURL, projectName)
	if err != nil {
		log.Printf("git clone --mirror %s to %s returned %s (%s)", gitURL, path, err, string(output))
		return fmt.Errorf("failed to clone %s to %s", gitURL, path)
	}

	return nil
}

func (service *MirroredRepositories) updateRemote(path string) error {
	output, err := service.execGitCommand(path, "remote", "update")
	if err != nil {
		log.Printf("[WARN] git remote update returned error %s for %s (%s)", err, path, string(output))
		return errors.New("update failed")
	}

	return nil
}

func (service *MirroredRepositories) execGitCommand(path string, args ...string) ([]byte, error) {
	cmd := exec.Command(gitCmd, args...)
	cmd.Dir = filepath.Join(service.mirrorPath, path)

	output, err := cmd.CombinedOutput()
	output = bytes.TrimSpace(output)

	return output, err
}
