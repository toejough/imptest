package generics

// Repository is a generic interface for storage operations.
type Repository[T any] interface {
	Save(item T) error
	Get(id string) (T, error)
}

// ProcessItem is a generic function that uses a generic repository.
func ProcessItem[T any](repo Repository[T], id string, transformer func(T) T) error {
	item, err := repo.Get(id)
	if err != nil {
		return err
	}

	transformed := transformer(item)

	return repo.Save(transformed)
}
