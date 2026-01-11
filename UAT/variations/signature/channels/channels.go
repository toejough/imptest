// Package channels demonstrates mocking interfaces with channel types.
package channels

// ChannelHandler demonstrates using different channel types in interface methods.
// This tests that the generator correctly handles chan, chan<-, and <-chan types.
type ChannelHandler interface {
	// SendOnly takes a send-only channel
	SendOnly(ch chan<- int) error

	// ReceiveOnly takes a receive-only channel
	ReceiveOnly(ch <-chan string) (string, error)

	// Bidirectional takes a bidirectional channel
	Bidirectional(ch chan bool) bool

	// ReturnChannel returns a receive-only channel
	ReturnChannel() <-chan int
}
