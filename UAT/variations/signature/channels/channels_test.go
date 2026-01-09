package channels_test

import (
	"testing"
)

func TestChannelHandler_Bidirectional(t *testing.T) {
	t.Parallel()

	mock := MockChannelHandler(t)

	testChannel := make(chan bool, 1)

	// Expect call with bidirectional channel
	go func() {
		call := mock.Method.Bidirectional.ExpectCalledWithExactly(testChannel)
		call.InjectReturnValues(true)
	}()

	// Call the mock
	result := mock.Mock.Bidirectional(testChannel)

	if !result {
		t.Fatal("expected true, got false")
	}
}

func TestChannelHandler_ReceiveOnly(t *testing.T) {
	t.Parallel()

	mock := MockChannelHandler(t)

	testChannel := make(chan string, 1)
	testChannel <- "test message"

	close(testChannel)

	// Expect call with receive-only channel
	go func() {
		call := mock.Method.ReceiveOnly.ExpectCalledWithExactly(testChannel)
		call.InjectReturnValues("received", nil)
	}()

	// Call the mock
	result, err := mock.Mock.ReceiveOnly(testChannel)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if result != "received" {
		t.Fatalf("expected 'received', got %q", result)
	}
}

func TestChannelHandler_ReturnChannel(t *testing.T) {
	t.Parallel()

	mock := MockChannelHandler(t)

	resultCh := make(chan int, 1)
	resultCh <- 42

	close(resultCh)

	// Convert to receive-only channel
	var recvOnlyCh <-chan int = resultCh

	// Expect call and return a channel
	go func() {
		call := mock.Method.ReturnChannel.ExpectCalledWithExactly()
		call.InjectReturnValues(recvOnlyCh)
	}()

	// Call the mock
	channel := mock.Mock.ReturnChannel()

	value := <-channel
	if value != 42 {
		t.Fatalf("expected 42, got %d", value)
	}
}

//go:generate impgen channels.ChannelHandler --dependency

func TestChannelHandler_SendOnly(t *testing.T) {
	t.Parallel()

	mock := MockChannelHandler(t)

	testChannel := make(chan int, 1)

	// Expect call with send-only channel
	go func() {
		call := mock.Method.SendOnly.ExpectCalledWithExactly(testChannel)
		call.InjectReturnValues(nil)
	}()

	// Call the mock
	err := mock.Mock.SendOnly(testChannel)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}
