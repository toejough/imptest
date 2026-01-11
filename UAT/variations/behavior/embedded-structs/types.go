// Package embeddedstructs demonstrates mocking interfaces with embedded structs.
package embeddedstructs

import "fmt"

// Counter is another base struct with counting methods.
type Counter struct {
	count int
}

// Inc increments the counter and returns the new value.
func (c *Counter) Inc() int {
	c.count++
	return c.count
}

// Value returns the current count.
func (c *Counter) Value() int {
	return c.count
}

// Logger is a base struct with logging methods.
type Logger struct {
	prefix string
}

// Log logs a message with the prefix.
func (l *Logger) Log(msg string) string {
	return fmt.Sprintf("[%s] %s", l.prefix, msg)
}

// SetPrefix updates the logger prefix.
func (l *Logger) SetPrefix(prefix string) {
	l.prefix = prefix
}

// TimedLogger embeds Logger and Counter, adding its own method.
// This demonstrates struct embedding with promoted methods.
type TimedLogger struct {
	Logger  // Embedded - Log and SetPrefix are promoted
	Counter // Embedded - Inc and Value are promoted
}

// LogWithCount logs a message and increments the counter.
// This is a method on TimedLogger that uses both embedded structs.
func (t *TimedLogger) LogWithCount(msg string) string {
	count := t.Inc()     // Uses promoted Inc from Counter
	logged := t.Log(msg) // Uses promoted Log from Logger

	return fmt.Sprintf("%s (count: %d)", logged, count)
}

// UseLogger demonstrates using just the Logger methods.
func UseLogger(logger interface {
	Log(msg string) string
	SetPrefix(prefix string)
}, msg string,
) string {
	logger.SetPrefix("INFO")

	return logger.Log(msg)
}

// UseTimedLogger demonstrates using a TimedLogger through an interface.
// This function accepts an interface matching TimedLogger's methods.
func UseTimedLogger(timedLogger interface {
	Log(msg string) string
	SetPrefix(prefix string)
	Inc() int
	Value() int
	LogWithCount(msg string) string
}, msg string,
) string {
	timedLogger.SetPrefix("APP")

	return timedLogger.LogWithCount(msg)
}
