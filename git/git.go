package git

import "golang.org/x/net/context"

// Command is the interface that wraps calls to Git.
type Command interface {
	IsRepository(ctx context.Context, fullPath string) bool
	CurrentBranch(ctx context.Context, fullPath string) string
	LastCommit(ctx context.Context, fullPath string) (Commit, error)
	CloneMirror(ctx context.Context, gitURL, fullPath string) error
	UpdateRemote(ctx context.Context, fullPath string) error
}
