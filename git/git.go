package git

// Command is the interface that wraps calls to Git.
type Command interface {
	IsRepository(fullPath string) bool
	CurrentBranch(fullPath string) string
	LastCommit(fullPath string) (Commit, error)
	Clone(gitURL, fullPath string) error
	CloneMirror(gitURL, fullPath string) error
	UpdateRemote(fullPath string) error
}
