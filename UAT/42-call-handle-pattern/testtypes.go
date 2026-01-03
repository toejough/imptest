package callhandle

import (
	"sync"
	"time"
)

// Calculator interface with multiple methods for comprehensive testing
type Calculator interface {
	Add(a, b int) int
	Divide(numerator, denominator int) (int, bool)
	Multiply(value int) int
	ProcessValue(value int) int
}

// CalculatorImpl is a concrete implementation for testing
type CalculatorImpl struct {
	multiplier int
}

func NewCalculatorImpl(multiplier int) *CalculatorImpl {
	return &CalculatorImpl{multiplier: multiplier}
}

func (c *CalculatorImpl) Add(a, b int) int {
	return c.multiplier + a + b
}

func (c *CalculatorImpl) Divide(numerator, denominator int) (int, bool) {
	if denominator == 0 {
		return 0, false
	}

	return numerator / denominator, true
}

func (c *CalculatorImpl) Multiply(value int) int {
	return value * c.multiplier
}

func (c *CalculatorImpl) ProcessValue(value int) int {
	const offset = 10

	if value < 0 {
		panic("negative values not supported")
	}

	return c.Multiply(value) + offset
}

// Counter is a simple struct for testing struct wrappers
type Counter struct {
	mu    sync.Mutex
	value int
}

func NewCounter(initialValue int) *Counter {
	return &Counter{value: initialValue}
}

func (c *Counter) AddAmount(amount int) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.value += amount

	return c.value
}

func (c *Counter) GetValue() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.value
}

func (c *Counter) Increment() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.value++

	return c.value
}

// SlowService simulates async behavior
type SlowService struct {
	delay time.Duration
}

func NewSlowService(delay time.Duration) *SlowService {
	return &SlowService{delay: delay}
}

func (s *SlowService) Process(x int) int {
	const multiplier = 2

	time.Sleep(s.delay)

	return x * multiplier
}
