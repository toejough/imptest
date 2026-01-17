package timeconflict_test

import (
	"testing"
	"time"
)

//go:generate impgen Scheduler --dependency

func TestStdlibTimeTypes(t *testing.T) {
	t.Parallel()

	mock, imp := MockScheduler(t)

	// Test time.Time as parameter
	scheduledTime := time.Date(2025, 12, 27, 10, 30, 0, 0, time.UTC)

	go func() {
		_ = mock.ScheduleAt("task1", scheduledTime)
	}()

	imp.ScheduleAt.Expect("task1", scheduledTime).Return(nil)

	// Test time.Duration as parameter
	delay := 5 * time.Minute

	go func() {
		_ = mock.Delay("task2", delay)
	}()

	imp.Delay.Expect("task2", delay).Return(nil)

	// Test time.Time as return value
	expectedTime := time.Date(2025, 12, 28, 10, 30, 0, 0, time.UTC)

	go func() {
		_, _ = mock.NextRun()
	}()

	imp.NextRun.Expect().Return(expectedTime, nil)

	// Test time.Duration as return value
	go func() {
		_ = mock.GetInterval("task3")
	}()

	imp.GetInterval.Expect("task3").Return(10 * time.Second)
}
