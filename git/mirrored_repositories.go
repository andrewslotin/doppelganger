package git

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const DefaultMaster = "master"

var gitCmd string

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
		return nil, ErrorNotFound
	}

	repo := service.repositoryFromDir(fullName)
	repo.LatestMasterCommit = service.commitFromDir(fullName)

	return repo, nil
}

func (service *MirroredRepositories) checkDirIsRepository(path string) bool {
	gitDirName := filepath.Join(service.mirrorPath, path, ".git")

	fileInfo, err := os.Stat(gitDirName)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}

		log.Printf("failed to stat dir %s (%s)", gitDirName, err)
	}

	return fileInfo.IsDir()
}

func (service *MirroredRepositories) repositoryFromDir(path string) *Repository {
	return &Repository{
		FullName: path,
		Master:   service.currentBranch(path),
	}
}

func (service *MirroredRepositories) commitFromDir(path string) *Commit {
	output, err := service.execGitCommand(path, "log", "-n", "1", "--pretty=%H\n%cn\n%cD\n%s")
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

	commitDate, err := time.Parse(time.RFC1123Z, lines[2])
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

func (service *MirroredRepositories) execGitCommand(path string, args ...string) ([]byte, error) {
	cmd := exec.Command(gitCmd, args...)
	cmd.Dir = filepath.Join(service.mirrorPath, path)

	output, err := cmd.CombinedOutput()
	output = bytes.TrimSpace(output)

	return output, err
}
