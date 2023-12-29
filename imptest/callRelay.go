// Package imptest provides impure function testing functionality.
package imptest

// This file provides CallRelay

import (
	"errors"
	"fmt"
	"reflect"
	"time"
)

// CallRelay is intended to be used to relay calls from inside of dependencies of a
// function under test to the test.
type (
	CallRelay struct {
		callChan chan *Call
	}
)

// NewCallRelay creates and returns a pointer to a new CallRelay, with the underlying
// channel set up properly.
func NewCallRelay() *CallRelay {
	return &CallRelay{callChan: make(chan *Call)}
}

// Get gets a call from the relay.
func (cr *CallRelay) Get() (*Call, error) {
	select {
	case c, ok := <-cr.callChan:
		if !ok {
			return nil, errCallRelayAlreadyShutDown
		}

		return c, nil
	case <-time.After(time.Second):
		panic("testing timeout waiting for a call")
	}
}

// Shutdown shuts the relay down by closing the internal call channel.
func (cr *CallRelay) Shutdown() {
	close(cr.callChan)
}

// WaitForShutdown waits the given time for the relay to be shut down, or returns an error
// if the given time was exceeded.
func (cr *CallRelay) WaitForShutdown(waitTime time.Duration) error {
	select {
	case thisCall, ok := <-cr.callChan:
		if !ok {
			// channel is closed
			return nil
		}

		return fmt.Errorf("had a call queued: %v: %w", thisCall, errCallRelayNotShutDown)
	case <-time.After(waitTime):
		return errCallRelayShutdownTimeout
	}
}

// putCall puts a function & args onto the relay as a call.
func (cr *CallRelay) putCall(rf reflect.Value, args ...any) *Call {
	c := newCall(rf, args...)
	cr.callChan <- c

	return c
}

// Errors.
var (
	errCallRelayNotShutDown     = errors.New("call relay was not shut down")
	errCallRelayShutdownTimeout = errors.New("call relay timed out waiting for shutdown")
	errCallRelayAlreadyShutDown = errors.New("expected a call, but the relay was already shut down")
)
