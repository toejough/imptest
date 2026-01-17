package service_test

import (
	"errors"
	"testing"

	"github.com/toejough/imptest/UAT/variations/package/dot-imports/business/service"
	"github.com/toejough/imptest/UAT/variations/package/dot-imports/business/storage"
)

// TestUserServiceDeleteUser verifies DeleteUser with mocked repository.
func TestUserServiceDeleteUser(t *testing.T) {
	t.Parallel()

	mock, imp := MockRepository(t)
	svc := service.NewUserService(mock)

	// Launch goroutine to call DeleteUser
	go func() {
		_ = svc.DeleteUser("user999")
	}()

	// Verify and inject return
	call := imp.Delete.ExpectCalledWithExactly("user999")
	call.InjectReturnValues(nil)

	// Verify args
	args := call.GetArgs()
	if args.Key != "user999" {
		t.Fatalf("expected key = 'user999', got %q", args.Key)
	}
}

// TestUserServiceGetUser verifies GetUser with mocked repository.
func TestUserServiceGetUser(t *testing.T) {
	t.Parallel()

	mock, imp := MockRepository(t)
	svc := service.NewUserService(mock)

	expectedData := []byte("loaded user data")

	// Launch goroutine to call GetUser
	go func() {
		_, _ = svc.GetUser("user789")
	}()

	// Verify and inject return values
	call := imp.Load.ExpectCalledWithExactly("user789")
	call.InjectReturnValues(expectedData, nil)

	// Verify args
	args := call.GetArgs()
	if args.Key != "user789" {
		t.Fatalf("expected key = 'user789', got %q", args.Key)
	}
}

//go:generate impgen storage.Repository --dependency

// TestUserServiceSaveUser verifies SaveUser with mocked repository.
func TestUserServiceSaveUser(t *testing.T) {
	t.Parallel()

	mock, imp := MockRepository(t)
	svc := service.NewUserService(mock)

	userData := []byte("user data")

	// Launch goroutine to call SaveUser
	go func() {
		_ = svc.SaveUser("user123", userData)
	}()

	// Verify and inject return
	call := imp.Save.ExpectCalledWithExactly("user123", userData)
	call.InjectReturnValues(nil)

	// Verify args
	args := call.GetArgs()
	if args.Key != "user123" {
		t.Fatalf("expected key = 'user123', got %q", args.Key)
	}
}

// TestUserServiceSaveUserError verifies SaveUser handles repository errors.
func TestUserServiceSaveUserError(t *testing.T) {
	t.Parallel()

	mock, imp := MockRepository(t)
	svc := service.NewUserService(mock)

	userData := []byte("user data")
	expectedErr := errors.New("storage failure")

	// Launch goroutine to call SaveUser
	go func() {
		_ = svc.SaveUser("user456", userData)
	}()

	// Inject error return
	call := imp.Save.ExpectCalledWithExactly("user456", userData)
	call.InjectReturnValues(expectedErr)
}

// unexported variables.
var (
	_ storage.Repository = (*mockRepositoryImpl)(nil)
)
