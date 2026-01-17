// Package service demonstrates business logic using dot-imported interfaces.
package service

import (
	"fmt"

	. "github.com/toejough/imptest/UAT/variations/package/dot-imports/business/storage" //nolint:staticcheck,revive // Dot import intentional for UAT
)

type UserService struct {
	repo Repository
}

// NewUserService creates a new user service.
func NewUserService(repo Repository) *UserService {
	return &UserService{repo: repo}
}

// DeleteUser removes user data from the repository.
func (s *UserService) DeleteUser(userID string) error {
	err := s.repo.Delete(userID)
	if err != nil {
		return fmt.Errorf("failed to delete user %s: %w", userID, err)
	}

	return nil
}

// GetUser retrieves user data from the repository.
func (s *UserService) GetUser(userID string) ([]byte, error) {
	data, err := s.repo.Load(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to load user %s: %w", userID, err)
	}

	return data, nil
}

// SaveUser saves user data to the repository.
func (s *UserService) SaveUser(userID string, userData []byte) error {
	err := s.repo.Save(userID, userData)
	if err != nil {
		return fmt.Errorf("failed to save user %s: %w", userID, err)
	}

	return nil
}
