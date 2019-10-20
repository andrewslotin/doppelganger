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

	"golang.org/x/net/context"
)

const (
	// GitCommandDateLayout corresponds to `git log --date=format:%FT%T%z` date format.
	GitCommandDateLayout = "2006-01-02T15:04:05-0700"

	gitPrettyFormat = "%H\n%an\n%cn\n%cd\n%s"
	gitDateFormat   = "format:%FT%T%z"
)

var (
	gitPrettyFormatFieldsNum = strings.Count(gitPrettyFormat, "\n") + 1
	errUnexpectedExit        = errors.New("unexpected exit")
)

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
func (gitCmd systemGit) Exec(ctx context.Context, path, command string, args ...string) (output []byte, err error) {
	cmd := exec.CommandContext(ctx, string(gitCmd), append([]string{command}, args...)...)
	cmd.Dir = path

	output, err = cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			err = errors.New(string(exitErr.Stderr))
		} else {
			err = errUnexpectedExit
		}

		return nil, err
	}

	return bytes.TrimSpace(output), nil
}

// IsRepository checks if there if `path` is a git repository.
func (gitCmd systemGit) IsRepository(ctx context.Context, path string) bool {
	if fileInfo, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false
		}

		log.Printf("[WARN] failed to stat %s (%s)", path, err)
		return false
	} else if !fileInfo.IsDir() {
		return false
	}

	output, err := gitCmd.Exec(ctx, path, "rev-parse", "--is-inside-git-dir")
	if err == errUnexpectedExit {
		log.Printf("[WARN] git rev-parse --is-inside-git-dir returned %s for %s (%s)", err, path, string(output))
	} else if err != nil {
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

// CurrentBranch returns the name of current branch in `path`.
func (gitCmd systemGit) CurrentBranch(ctx context.Context, path string) string {
	refName, err := gitCmd.Exec(ctx, path, "symbolic-ref", "HEAD")
	if err != nil {
		log.Printf("[WARN] git symbolic-ref HEAD returned %s for %s (%s)", err, path, string(refName))
		return DefaultMaster
	}

	if !bytes.HasPrefix(refName, []byte("refs/heads/")) {
		log.Printf("[WARN] unexpected reference name for %s (%q)", path, refName)
		return DefaultMaster
	}

	return string(bytes.TrimPrefix(refName, []byte("refs/heads/")))
}

// LastCommit returns the latest commit from `path`.
func (gitCmd systemGit) LastCommit(ctx context.Context, path string) (commit Commit, err error) {
	output, err := gitCmd.Exec(ctx, path, "log", "-n", "1", "--pretty="+gitPrettyFormat, "--date="+gitDateFormat)
	if err != nil {
		log.Printf("[WARN] git log returned error %s for %s (%s)", err, path, string(output))
		return commit, nil
	}

	lines := strings.SplitN(string(output), "\n", gitPrettyFormatFieldsNum)
	if len(lines) < gitPrettyFormatFieldsNum {
		log.Printf("[WARN] unexpected output from git log for %s (%s)", path, string(output))
		return commit, nil
	}

	commit.SHA, commit.Author, commit.Committer, commit.Message = lines[0], lines[1], lines[2], lines[4]
	commit.Date, err = time.Parse(GitCommandDateLayout, lines[3])
	if err != nil {
		log.Printf("[WARN] unexpected date format from git log for %s (%s)", path, lines[3])
		commit.Date = time.Time{}
	}

	return commit, nil
}

// CloneMirror performs mirror clone of specified git URL to `path`.
func (gitCmd systemGit) CloneMirror(ctx context.Context, gitURL, path string) error {
	dir, projectName := filepath.Dir(path), filepath.Base(path)

	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("failed to create %s (%s)", dir, err)
		return fmt.Errorf("failed to clone %s to %s", gitURL, path)
	}

	output, err := gitCmd.Exec(ctx, dir, "clone", "--mirror", gitURL, projectName)
	if err != nil {
		log.Printf("git clone --mirror %s to %s returned %s (%s)", gitURL, path, err, string(output))
		return fmt.Errorf("failed to clone %s to %s", gitURL, path)
	}

	return nil
}

// UpdateRemote does `git remote update` in specified `path`.
func (gitCmd systemGit) UpdateRemote(ctx context.Context, path string) error {
	output, err := gitCmd.Exec(ctx, path, "remote", "update")
	if err != nil {
		log.Printf("[WARN] git remote update returned %s for %s (%s)", err, path, string(output))
		return errors.New("update failed")
	}

	return nil
}
