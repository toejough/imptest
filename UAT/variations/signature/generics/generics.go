// Package generics demonstrates mocking generic interfaces.
package generics

import "fmt"

// Repository is a generic interface for storage operations.
type Repository[T any] interface {
	Save(item T) error
	Get(id string) (T, error)
}

// ProcessItem is a generic function that uses a generic repository.
func ProcessItem[T any](repo Repository[T], id string, transformer func(T) T) error {
	item, err := repo.Get(id)
	if err != nil {
		return fmt.Errorf("failed to get item: %w", err)
	}

	transformed := transformer(item)

	err = repo.Save(transformed)
	if err != nil {
		return fmt.Errorf("failed to save item: %w", err)
	}

	return nil
}
