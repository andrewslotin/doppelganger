package git

type Repository struct {
	FullName    string
	Description string
	Master      string
	HTMLURL     string
	GitURL      string

	LatestMasterCommit *Commit
}

func (repo *Repository) Mirrored() bool {
	return repo.HTMLURL == ""
}
