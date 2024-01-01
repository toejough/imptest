// Package imptest provides impure function testing functionality.
package imptest

// This file provides CallRelay

import (
	"errors"
	"fmt"
	"time"
)

// CallRelay is intended to be used to relay calls from inside of dependencies of a
// function under test to the test.
type (
	CallRelay struct {
		callChan chan *Call
		deps     CallRelayDeps
	}
	CallRelayDeps interface {
		After(duration time.Duration) <-chan time.Time
	}
)

// NewCallRelay creates and returns a pointer to a new CallRelay, with the underlying
// channel set up properly.
func NewCallRelay(deps CallRelayDeps) *CallRelay {
	return &CallRelay{
		callChan: make(chan *Call),
		deps:     deps,
	}
}

// NewDefaultCallRelay creates and returns a pointer to a new CallRelay, with a new
// DefaultCallRelayDeps.
func NewDefaultCallRelay() *CallRelay {
	return &CallRelay{
		callChan: make(chan *Call),
		deps:     NewDefaultCallRelayDeps(),
	}
}

// NewDefaultCallRelayDeps creates and returns a pointer to a new NewDefaultCallRelayDeps.
func NewDefaultCallRelayDeps() *DefaultCallRelayDeps { return &DefaultCallRelayDeps{} }

// DefaultCallRelayDeps is the default implementation of CallRelayDeps, which uses the
// standard lib time.After to supply the After method.
type DefaultCallRelayDeps struct{}

// After takes a duration and returns a channel which returns the time elapsed once the duration
// has been met or exceeded.
func (deps *DefaultCallRelayDeps) After(d time.Duration) <-chan time.Time { return time.After(d) }

// Get gets a call from the relay.
func (cr *CallRelay) Get() (*Call, error) {
	select {
	case c, ok := <-cr.callChan:
		if !ok {
			return nil, errCallRelayAlreadyShutDown
		}

		return c, nil
		// TODO: use the deps after
		// TODO: pass duration in. rename to "GetWithin"
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
	case <-cr.deps.After(waitTime):
		return errCallRelayShutdownTimeout
	}
}

// putCall puts a function & args onto the relay as a call.
func (cr *CallRelay) putCall(f Function, args ...any) *Call {
	c := newCall(f, args...)
	cr.callChan <- c

	return c
}

// Errors.
var (
	errCallRelayNotShutDown     = errors.New("call relay was not shut down")
	errCallRelayShutdownTimeout = errors.New("call relay timed out waiting for shutdown")
	errCallRelayAlreadyShutDown = errors.New("expected a call, but the relay was already shut down")
)
