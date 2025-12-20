package callable

// ExternalService is a dependency for our business logic.
type ExternalService interface {
	FetchData(id int) (string, error)
	Process(data string) string
}

// BusinessLogic is the function we want to test.
// It orchestrates calls to an ExternalService.
func BusinessLogic(svc ExternalService, id int) (string, error) {
	data, err := svc.FetchData(id)
	if err != nil {
		return "", err
	}

	result := svc.Process(data)

	return "Result: " + result, nil
}
