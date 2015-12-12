package git

type RepositoryService interface {
	All() ([]*Repository, error)
	Get(name string) (*Repository, error)
}
