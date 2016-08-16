package git

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// GitCommandDateLayout corresponds to `git log --date=format:%FT%T%z` date format.
const GitCommandDateLayout = "2006-01-02T15:04:05-0700"

type systemGit string

// SystemGit returns an object that wraps system git command and implements git.Command interface.
// If git is not found in PATH an error will be returned.
func SystemGit() (systemGit, error) {
	cmd, err := exec.LookPath("git")
	if err != nil {
		return systemGit(""), errors.New("git is not found in PATH")
	}

	return systemGit(cmd), nil
}

// Exec runs specified Git command in path and returns its output. If Git returns a non-zero
// status, an error is returned and output contains error details.
func (gitCmd systemGit) Exec(path, command string, args ...string) (output []byte, err error) {
	cmd := exec.Command(string(gitCmd), append([]string{command}, args...)...)
	cmd.Dir = path

	output, err = cmd.CombinedOutput()
	output = bytes.TrimSpace(output)

	return output, err
}

func (gitCmd systemGit) IsRepository(fullPath string) bool {
	if fileInfo, err := os.Stat(fullPath); err != nil {
		if os.IsNotExist(err) {
			return false
		}

		log.Printf("[WARN] failed to stat %s (%s)", fullPath, err)
		return false
	} else if !fileInfo.IsDir() {
		return false
	}

	output, err := gitCmd.Exec(fullPath, "rev-parse", "--is-inside-git-dir")
	if err != nil {
		log.Printf("[WARN] git rev-parse --is-inside-git-dir returned error %s for %s (%s)", err, fullPath, string(output))
		return false
	}

	switch string(output) {
	case "true":
		return true
	case "false":
		return false
	default:
		log.Printf("[WARN] git rev-parse --is-inside-git-dir returned unexpected output for %s: %q", fullPath, string(output))
		return false
	}
}

func (gitCmd systemGit) CurrentBranch(fullPath string) string {
	refName, err := gitCmd.Exec(fullPath, "symbolic-ref", "HEAD")
	if err != nil {
		log.Printf("[WARN] git symbolic-ref HEAD returned error %s for %s (%s)", err, fullPath, string(refName))
		return DefaultMaster
	}

	if !bytes.HasPrefix(refName, []byte("refs/heads/")) {
		log.Printf("[WARN] unexpected reference name for %s (%q)", fullPath, refName)
		return DefaultMaster
	}

	return string(bytes.TrimPrefix(refName, []byte("refs/heads/")))
}

func (gitCmd systemGit) LastCommit(fullPath string) (commit Commit, err error) {
	output, err := gitCmd.Exec(fullPath, "log", "-n", "1", "--pretty=%H\n%cn\n%cd\n%s", "--date=format:%FT%T%z")
	if err != nil {
		log.Printf("[WARN] git log returned error %s for %s (%s)", err, fullPath, string(output))
		return commit, nil
	}

	lines := strings.SplitN(string(output), "\n", 4)
	if len(lines) < 4 {
		log.Printf("[WARN] unexpected output from git log for %s (%s)", fullPath, string(output))
		return commit, nil
	}

	commit.SHA, commit.Author, commit.Message = lines[0], lines[1], lines[3]
	commit.Date, err = time.Parse(GitCommandDateLayout, lines[2])
	if err != nil {
		log.Printf("[WARN] unexpected date format from git log for %s (%s)", fullPath, lines[2])
		commit.Date = time.Time{}
	}

	return commit, nil
}

func (gitCmd systemGit) CloneMirror(gitURL, fullPath string) error {
	path, projectName := filepath.Dir(fullPath), filepath.Base(fullPath)

	if err := os.MkdirAll(path, 0755); err != nil {
		log.Printf("failed to create %s (%s)", path, err)
		return fmt.Errorf("failed to clone %s to %s", gitURL, fullPath)
	}

	output, err := gitCmd.Exec(path, "clone", "--mirror", gitURL, projectName)
	if err != nil {
		log.Printf("git clone --mirror %s to %s returned %s (%s)", gitURL, fullPath, err, string(output))
		return fmt.Errorf("failed to clone %s to %s", gitURL, fullPath)
	}

	return nil
}

func (gitCmd systemGit) UpdateRemote(fullPath string) error {
	output, err := gitCmd.Exec(fullPath, "remote", "update")
	if err != nil {
		log.Printf("[WARN] git remote update returned error %s for %s (%s)", err, fullPath, string(output))
		return errors.New("update failed")
	}

	return nil
}
