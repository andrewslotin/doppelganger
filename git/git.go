package git

import "time"

// Command is the interface that wraps calls to Git.
type Command interface {
	IsRepository(fullPath string) bool
	CurrentBranch(fullPath string) string
	LastCommit(fullPath string) (rev, author, message string, createdAt time.Time, err error)
	CloneMirror(gitURL, fullPath string) error
	UpdateRemote(fullPath string) error
}
