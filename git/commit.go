package git

import "time"

// Commit represents a single commit in Git repository.
type Commit struct {
	SHA       string
	Message   string
	Author    string
	Committer string
	Date      time.Time
}
