package timeconflict_test

import (
	"testing"
	"time"
)

//go:generate impgen Scheduler --dependency

func TestStdlibTimeTypes(t *testing.T) {
	t.Parallel()

	mock := MockScheduler(t)

	// Test time.Time as parameter
	scheduledTime := time.Date(2025, 12, 27, 10, 30, 0, 0, time.UTC)

	go func() {
		_ = mock.Mock.ScheduleAt("task1", scheduledTime)
	}()

	mock.Method.ScheduleAt.ExpectCalledWithExactly("task1", scheduledTime).InjectReturnValues(nil)

	// Test time.Duration as parameter
	delay := 5 * time.Minute

	go func() {
		_ = mock.Mock.Delay("task2", delay)
	}()

	mock.Method.Delay.ExpectCalledWithExactly("task2", delay).InjectReturnValues(nil)

	// Test time.Time as return value
	expectedTime := time.Date(2025, 12, 28, 10, 30, 0, 0, time.UTC)

	go func() {
		_, _ = mock.Mock.NextRun()
	}()

	mock.Method.NextRun.ExpectCalledWithExactly().InjectReturnValues(expectedTime, nil)

	// Test time.Duration as return value
	go func() {
		_ = mock.Mock.GetInterval("task3")
	}()

	mock.Method.GetInterval.ExpectCalledWithExactly("task3").InjectReturnValues(10 * time.Second)
}
