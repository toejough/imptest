package core_test

//go:generate ../../bin/impgen TestReporter --dependency

// TestGenericCallDone tests GenericCall.Done.

// TestGenericCallName tests GenericCall.Name.

// TestImpFatalf tests that Imp.Fatalf delegates to the underlying test reporter.

// Handle the Fatalf call

// TestImpGetCallEventually_QueuesOtherMethods tests that Imp.GetCallEventually
// queues calls with different method names while waiting for the matching method.

// Handle Helper calls (one per GetCallEventually call)

// Validator that accepts any arguments

// Accept any args

// Start waiter for "Add" method

// Give waiter time to register

// Send "Multiply" call first (should be queued)

// Send "Add" call second (should match the waiter)

// Verify we received the "Add" call

// Verify "Multiply" call is still queued
// We use GetCall (which checks queue first) with a validator for "Multiply"
