package git

import "golang.org/x/net/context"

// RepositoryService is a type that wraps All and Get methods.
//
// Repository service is used to list and lookup repositories. Two implementations
type RepositoryService interface {
	All(ctx context.Context) ([]*Repository, error)
	Get(ctx context.Context, name string) (*Repository, error)
}

// MirrorService is a type that extends RepositoryService adding two more methods: Create and Update.
//
// Mirror service is used to create new repository mirrors and update existing ones.
type MirrorService interface {
	RepositoryService

	Create(ctx context.Context, name, url string) error
	Update(ctx context.Context, name string) error
}

// TrackingService is a type that wraps Track method.
//
// Tracking service is used to set up tracking changes in a repository.
type TrackingService interface {
	Track(ctx context.Context, name, callbackURL string) error
}
