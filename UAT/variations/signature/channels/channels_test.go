package channels_test

import (
	"testing"

	// Import for impgen to resolve the package.
	_ "github.com/toejough/imptest/UAT/variations/signature/channels"
)

func TestChannelHandler_Bidirectional(t *testing.T) {
	t.Parallel()

	mock, imp := MockChannelHandler(t)

	testChannel := make(chan bool, 1)

	// Expect call with bidirectional channel
	go func() {
		call := imp.Bidirectional.ExpectCalledWithExactly(testChannel)
		call.InjectReturnValues(true)
	}()

	// Call the mock
	result := mock.Bidirectional(testChannel)

	if !result {
		t.Fatal("expected true, got false")
	}
}

func TestChannelHandler_ReceiveOnly(t *testing.T) {
	t.Parallel()

	mock, imp := MockChannelHandler(t)

	testChannel := make(chan string, 1)
	testChannel <- "test message"

	close(testChannel)

	// Expect call with receive-only channel
	go func() {
		call := imp.ReceiveOnly.ExpectCalledWithExactly(testChannel)
		call.InjectReturnValues("received", nil)
	}()

	// Call the mock
	result, err := mock.ReceiveOnly(testChannel)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if result != "received" {
		t.Fatalf("expected 'received', got %q", result)
	}
}

func TestChannelHandler_ReturnChannel(t *testing.T) {
	t.Parallel()

	mock, imp := MockChannelHandler(t)

	resultCh := make(chan int, 1)
	resultCh <- 42

	close(resultCh)

	// Convert to receive-only channel
	var recvOnlyCh <-chan int = resultCh

	// Expect call and return a channel
	go func() {
		call := imp.ReturnChannel.ExpectCalledWithExactly()
		call.InjectReturnValues(recvOnlyCh)
	}()

	// Call the mock
	channel := mock.ReturnChannel()

	value := <-channel
	if value != 42 {
		t.Fatalf("expected 42, got %d", value)
	}
}

//go:generate impgen channels.ChannelHandler --dependency

func TestChannelHandler_SendOnly(t *testing.T) {
	t.Parallel()

	mock, imp := MockChannelHandler(t)

	testChannel := make(chan int, 1)

	// Expect call with send-only channel
	go func() {
		call := imp.SendOnly.ExpectCalledWithExactly(testChannel)
		call.InjectReturnValues(nil)
	}()

	// Call the mock
	err := mock.SendOnly(testChannel)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}
