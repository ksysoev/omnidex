// Package user provides repository implementations for user-related data access.
package user

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// userDAO defines the interface for user data access operations.
type userDAO interface {
	Ping(ctx context.Context) *redis.StatusCmd
}

// UserRepo provides methods to interact with the user data store.
type UserRepo struct {
	dao userDAO
}

// New creates a new instance of UserRepo using the provided userDAO.
// It returns a pointer to the initialized UserRepo.
func New(dao userDAO) *UserRepo {
	return &UserRepo{
		dao: dao,
	}
}

// CheckHealth verifies the health of the user repository by pinging the underlying DAO.
// It returns an error if the health check fails.
func (u *UserRepo) CheckHealth(ctx context.Context) error {
	res := u.dao.Ping(ctx)

	if err := res.Err(); err != nil {
		return fmt.Errorf("fail to check health for user repo: %w", err)
	}

	return nil
}
