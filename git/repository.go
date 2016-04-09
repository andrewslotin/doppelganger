package git

// Repository represents single git repository.
type Repository struct {
	// Full repository name. For GitHub repositories it's set to <user>/<repo>,
	// for local mirrors this is a path inside the mirror directory.
	FullName string
	// Description is optional and mostly applies to GitHub repositories.
	Description string
	// The name of master branch.
	Master string
	// A link to repository on GitHub.
	HTMLURL string
	// Remote URL.
	GitURL string

	// The latest commit from master.
	LatestMasterCommit *Commit
}

// Mirrored returns true if this is a local repository mirror.
func (repo *Repository) Mirrored() bool {
	return repo.HTMLURL == ""
}
