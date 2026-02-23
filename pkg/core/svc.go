// Package core provides core service logic and interfaces.
package core

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// userRepo defines the interface for user repository operations.
type userRepo interface {
	CheckHealth(ctx context.Context) error
}

// someAPIProv defines the interface for a provider that can check health status.
type someAPIProv interface {
	CheckHealth(ctx context.Context) error
}

// Service encapsulates core business logic and dependencies.
type Service struct {
	users   userRepo
	someAPI someAPIProv
}

// New creates a new Service instance with the provided userRepo and someAPI.
func New(users userRepo, someAPI someAPIProv) *Service {
	return &Service{
		users:   users,
		someAPI: someAPI,
	}
}

// CheckHealth checks the health of the core service and its dependencies.
func (s *Service) CheckHealth(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error { return s.someAPI.CheckHealth(ctx) })
	eg.Go(func() error { return s.users.CheckHealth(ctx) })

	return eg.Wait()
}
