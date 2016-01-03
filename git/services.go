package git

type RepositoryService interface {
	All() ([]*Repository, error)
	Get(name string) (*Repository, error)
}

type MirrorService interface {
	RepositoryService

	Create(name, url string) error
	Update(name string) error
}

type TrackingService interface {
	Track(name, callbackURL string) error
}
