package channels_test

import (
	"testing"
)

func TestChannelHandler_Bidirectional(t *testing.T) {
	mock := MockChannelHandler(t)

	ch := make(chan bool, 1)

	// Expect call with bidirectional channel
	go func() {
		call := mock.Bidirectional.ExpectCalledWithExactly(ch)
		call.InjectReturnValues(true)
	}()

	// Call the mock
	result := mock.Interface().Bidirectional(ch)

	if !result {
		t.Fatal("expected true, got false")
	}
}

func TestChannelHandler_ReceiveOnly(t *testing.T) {
	mock := MockChannelHandler(t)

	ch := make(chan string, 1)
	ch <- "test message"
	close(ch)

	// Expect call with receive-only channel
	go func() {
		call := mock.ReceiveOnly.ExpectCalledWithExactly(ch)
		call.InjectReturnValues("received", nil)
	}()

	// Call the mock
	result, err := mock.Interface().ReceiveOnly(ch)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if result != "received" {
		t.Fatalf("expected 'received', got %q", result)
	}
}

func TestChannelHandler_ReturnChannel(t *testing.T) {
	mock := MockChannelHandler(t)

	resultCh := make(chan int, 1)
	resultCh <- 42
	close(resultCh)

	// Convert to receive-only channel
	var recvOnlyCh <-chan int = resultCh

	// Expect call and return a channel
	go func() {
		call := mock.ReturnChannel.ExpectCalledWithExactly()
		call.InjectReturnValues(recvOnlyCh)
	}()

	// Call the mock
	ch := mock.Interface().ReturnChannel()

	value := <-ch
	if value != 42 {
		t.Fatalf("expected 42, got %d", value)
	}
}

//go:generate go run ../../impgen --dependency channels.ChannelHandler

func TestChannelHandler_SendOnly(t *testing.T) {
	mock := MockChannelHandler(t)

	ch := make(chan int, 1)

	// Expect call with send-only channel
	go func() {
		call := mock.SendOnly.ExpectCalledWithExactly(ch)
		call.InjectReturnValues(nil)
	}()

	// Call the mock
	err := mock.Interface().SendOnly(ch)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}
