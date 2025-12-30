package timeconflict_test

import (
	"testing"
	"time"
)

//go:generate impgen timeconflict.Scheduler --dependency

func TestStdlibTimeTypes(t *testing.T) {
	t.Parallel()

	mock := MockScheduler(t)

	// Test time.Time as parameter
	scheduledTime := time.Date(2025, 12, 27, 10, 30, 0, 0, time.UTC)

	go func() {
		_ = mock.Interface().ScheduleAt("task1", scheduledTime)
	}()

	mock.ScheduleAt.ExpectCalledWithExactly("task1", scheduledTime).InjectReturnValues(nil)

	// Test time.Duration as parameter
	delay := 5 * time.Minute

	go func() {
		_ = mock.Interface().Delay("task2", delay)
	}()

	mock.Delay.ExpectCalledWithExactly("task2", delay).InjectReturnValues(nil)

	// Test time.Time as return value
	expectedTime := time.Date(2025, 12, 28, 10, 30, 0, 0, time.UTC)

	go func() {
		_, _ = mock.Interface().NextRun()
	}()

	mock.NextRun.ExpectCalledWithExactly().InjectReturnValues(expectedTime, nil)

	// Test time.Duration as return value
	go func() {
		_ = mock.Interface().GetInterval("task3")
	}()

	mock.GetInterval.ExpectCalledWithExactly("task3").InjectReturnValues(10 * time.Second)
}
