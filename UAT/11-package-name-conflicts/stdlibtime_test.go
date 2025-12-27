package timeconflict_test

import (
	"testing"
	"time"
)

//go:generate impgen timeconflict.Scheduler

func TestStdlibTimeTypes(t *testing.T) {
	t.Parallel()

	imp := NewSchedulerImp(t)

	// Test time.Time as parameter
	scheduledTime := time.Date(2025, 12, 27, 10, 30, 0, 0, time.UTC)

	go func() {
		_ = imp.Mock.ScheduleAt("task1", scheduledTime)
	}()

	imp.ExpectCallIs.ScheduleAt().ExpectArgsAre("task1", scheduledTime).InjectResult(nil)

	// Test time.Duration as parameter
	delay := 5 * time.Minute

	go func() {
		_ = imp.Mock.Delay("task2", delay)
	}()

	imp.ExpectCallIs.Delay().ExpectArgsAre("task2", delay).InjectResult(nil)

	// Test time.Time as return value
	expectedTime := time.Date(2025, 12, 28, 10, 30, 0, 0, time.UTC)

	go func() {
		_, _ = imp.Mock.NextRun()
	}()

	imp.ExpectCallIs.NextRun().InjectResults(expectedTime, nil)

	// Test time.Duration as return value
	go func() {
		_ = imp.Mock.GetInterval("task3")
	}()

	imp.ExpectCallIs.GetInterval().ExpectArgsAre("task3").InjectResult(10 * time.Second)
}
