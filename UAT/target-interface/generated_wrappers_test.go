package targetinterface_test

import (
	"github.com/toejough/imptest/imptest"
)

// WrapBasicCalculator wraps a BasicCalculator struct for testing.
func WrapBasicCalculator(t imptest.TestReporter, instance *BasicCalculator) *WrapBasicCalculatorWrapper {
	imp := imptest.NewImp(t)
	wrapper := &WrapBasicCalculatorWrapper{
		imp:      imp,
		instance: instance,
	}
	wrapper.Add = &WrapBasicCalculatorAddMethod{wrapper: wrapper}

	return wrapper
}

// WrapBasicCalculatorWrapper provides a fluent API for calling and verifying BasicCalculator methods.
type WrapBasicCalculatorWrapper struct {
	imp      *imptest.Imp
	instance *BasicCalculator
	Add      *WrapBasicCalculatorAddMethod
}

// WrapBasicCalculatorAddMethod wraps the Add method.
type WrapBasicCalculatorAddMethod struct {
	wrapper    *WrapBasicCalculatorWrapper
	returnChan chan WrapBasicCalculatorAddReturns
	panicChan  chan any
	returned   *WrapBasicCalculatorAddReturns
	panicked   any
}

// Start begins execution of the Add method in a goroutine.
func (m *WrapBasicCalculatorAddMethod) Start(arg1, arg2 int) *WrapBasicCalculatorAddMethod {
	m.returnChan = make(chan WrapBasicCalculatorAddReturns, 1)
	m.panicChan = make(chan any, 1)
	m.returned = nil
	m.panicked = nil

	go func() {
		defer func() {
			if r := recover(); r != nil {
				m.panicChan <- r
			}
		}()

		result := m.wrapper.instance.Add(arg1, arg2)
		m.returnChan <- WrapBasicCalculatorAddReturns{R1: result}
	}()

	return m
}

// WaitForResponse blocks until the method completes (return or panic).
func (m *WrapBasicCalculatorAddMethod) WaitForResponse() {
	if m.returned != nil || m.panicked != nil {
		return
	}

	select {
	case ret := <-m.returnChan:
		m.returned = &ret
	case p := <-m.panicChan:
		m.panicked = p
	}
}

// ExpectReturnsEqual verifies the method returned exact values.
func (m *WrapBasicCalculatorAddMethod) ExpectReturnsEqual(expected int) {
	m.wrapper.imp.Helper()
	m.WaitForResponse()

	if m.panicked != nil {
		m.wrapper.imp.Fatalf("expected method to return, but it panicked with: %v", m.panicked)
		return
	}

	if m.returned.R1 != expected {
		m.wrapper.imp.Fatalf("expected %v, got %v", expected, m.returned.R1)
	}
}

// WrapBasicCalculatorAddReturns provides type-safe access to Add return values.
type WrapBasicCalculatorAddReturns struct {
	R1 int
}

// WrapCalculator wraps a Calculator implementation for testing.
func WrapCalculator(t imptest.TestReporter, instance Calculator) *WrapCalculatorWrapper {
	imp := imptest.NewImp(t)
	wrapper := &WrapCalculatorWrapper{
		imp:      imp,
		instance: instance,
	}
	wrapper.Add = &WrapCalculatorAddMethod{wrapper: wrapper}
	wrapper.Subtract = &WrapCalculatorSubtractMethod{wrapper: wrapper}
	wrapper.Divide = &WrapCalculatorDivideMethod{wrapper: wrapper}

	return wrapper
}

// WrapCalculatorWrapper provides a fluent API for calling and verifying Calculator methods.
type WrapCalculatorWrapper struct {
	imp      *imptest.Imp
	instance Calculator
	Add      *WrapCalculatorAddMethod
	Subtract *WrapCalculatorSubtractMethod
	Divide   *WrapCalculatorDivideMethod
}

// WrapCalculatorAddMethod wraps the Add method.
type WrapCalculatorAddMethod struct {
	wrapper    *WrapCalculatorWrapper
	returnChan chan WrapCalculatorAddReturns
	panicChan  chan any
	returned   *WrapCalculatorAddReturns
	panicked   any
}

// Start begins execution of the Add method in a goroutine.
func (m *WrapCalculatorAddMethod) Start(arg1, arg2 int) *WrapCalculatorAddMethod {
	m.returnChan = make(chan WrapCalculatorAddReturns, 1)
	m.panicChan = make(chan any, 1)
	m.returned = nil
	m.panicked = nil

	go func() {
		defer func() {
			if r := recover(); r != nil {
				m.panicChan <- r
			}
		}()

		result := m.wrapper.instance.Add(arg1, arg2)
		m.returnChan <- WrapCalculatorAddReturns{R1: result}
	}()

	return m
}

// WaitForResponse blocks until the method completes (return or panic).
func (m *WrapCalculatorAddMethod) WaitForResponse() {
	if m.returned != nil || m.panicked != nil {
		return
	}

	select {
	case ret := <-m.returnChan:
		m.returned = &ret
	case p := <-m.panicChan:
		m.panicked = p
	}
}

// ExpectReturnsEqual verifies the method returned exact values.
func (m *WrapCalculatorAddMethod) ExpectReturnsEqual(expected int) {
	m.wrapper.imp.Helper()
	m.WaitForResponse()

	if m.panicked != nil {
		m.wrapper.imp.Fatalf("expected method to return, but it panicked with: %v", m.panicked)
		return
	}

	if m.returned.R1 != expected {
		m.wrapper.imp.Fatalf("expected %v, got %v", expected, m.returned.R1)
	}
}

// WrapCalculatorAddReturns provides type-safe access to Add return values.
type WrapCalculatorAddReturns struct {
	R1 int
}

// WrapCalculatorSubtractMethod wraps the Subtract method.
type WrapCalculatorSubtractMethod struct {
	wrapper    *WrapCalculatorWrapper
	returnChan chan WrapCalculatorSubtractReturns
	panicChan  chan any
	returned   *WrapCalculatorSubtractReturns
	panicked   any
}

// Start begins execution of the Subtract method in a goroutine.
func (m *WrapCalculatorSubtractMethod) Start(arg1, arg2 int) *WrapCalculatorSubtractMethod {
	m.returnChan = make(chan WrapCalculatorSubtractReturns, 1)
	m.panicChan = make(chan any, 1)
	m.returned = nil
	m.panicked = nil

	go func() {
		defer func() {
			if r := recover(); r != nil {
				m.panicChan <- r
			}
		}()

		result := m.wrapper.instance.Subtract(arg1, arg2)
		m.returnChan <- WrapCalculatorSubtractReturns{R1: result}
	}()

	return m
}

// WaitForResponse blocks until the method completes (return or panic).
func (m *WrapCalculatorSubtractMethod) WaitForResponse() {
	if m.returned != nil || m.panicked != nil {
		return
	}

	select {
	case ret := <-m.returnChan:
		m.returned = &ret
	case p := <-m.panicChan:
		m.panicked = p
	}
}

// ExpectReturnsMatch verifies return values match the given matchers.
func (m *WrapCalculatorSubtractMethod) ExpectReturnsMatch(matchers ...any) {
	m.wrapper.imp.Helper()
	m.WaitForResponse()

	if m.panicked != nil {
		m.wrapper.imp.Fatalf("expected method to return, but it panicked with: %v", m.panicked)
		return
	}

	if len(matchers) != 1 {
		m.wrapper.imp.Fatalf("expected 1 matcher, got %d", len(matchers))
		return
	}

	matcher, ok := matchers[0].(imptest.Matcher)
	if !ok {
		m.wrapper.imp.Fatalf("argument 0 is not a Matcher")
		return
	}

	success, err := matcher.Match(m.returned.R1)
	if err != nil {
		m.wrapper.imp.Fatalf("matcher error: %v", err)
		return
	}

	if !success {
		m.wrapper.imp.Fatalf("return value: %s", matcher.FailureMessage(m.returned.R1))
	}
}

// WrapCalculatorSubtractReturns provides type-safe access to Subtract return values.
type WrapCalculatorSubtractReturns struct {
	R1 int
}

// WrapCalculatorDivideMethod wraps the Divide method.
type WrapCalculatorDivideMethod struct {
	wrapper    *WrapCalculatorWrapper
	returnChan chan WrapCalculatorDivideReturns
	panicChan  chan any
	returned   *WrapCalculatorDivideReturns
	panicked   any
}

// Start begins execution of the Divide method in a goroutine.
func (m *WrapCalculatorDivideMethod) Start(arg1, arg2 int) *WrapCalculatorDivideMethod {
	m.returnChan = make(chan WrapCalculatorDivideReturns, 1)
	m.panicChan = make(chan any, 1)
	m.returned = nil
	m.panicked = nil

	go func() {
		defer func() {
			if r := recover(); r != nil {
				m.panicChan <- r
			}
		}()

		r1, r2 := m.wrapper.instance.Divide(arg1, arg2)
		m.returnChan <- WrapCalculatorDivideReturns{R1: r1, R2: r2}
	}()

	return m
}

// WaitForResponse blocks until the method completes (return or panic).
func (m *WrapCalculatorDivideMethod) WaitForResponse() {
	if m.returned != nil || m.panicked != nil {
		return
	}

	select {
	case ret := <-m.returnChan:
		m.returned = &ret
	case p := <-m.panicChan:
		m.panicked = p
	}
}

// ExpectReturnsEqual verifies the method returned exact values.
func (m *WrapCalculatorDivideMethod) ExpectReturnsEqual(expectedR1 int, expectedR2 error) {
	m.wrapper.imp.Helper()
	m.WaitForResponse()

	if m.panicked != nil {
		m.wrapper.imp.Fatalf("expected method to return, but it panicked with: %v", m.panicked)
		return
	}

	if m.returned.R1 != expectedR1 {
		m.wrapper.imp.Fatalf("expected R1=%v, got R1=%v", expectedR1, m.returned.R1)
	}

	if m.returned.R2 != expectedR2 {
		m.wrapper.imp.Fatalf("expected R2=%v, got R2=%v", expectedR2, m.returned.R2)
	}
}

// ExpectPanicEquals verifies the method panicked with an exact value.
func (m *WrapCalculatorDivideMethod) ExpectPanicEquals(expected any) {
	m.wrapper.imp.Helper()
	m.WaitForResponse()

	if m.returned != nil {
		m.wrapper.imp.Fatalf("expected method to panic, but it returned normally")
		return
	}

	if m.panicked != expected {
		m.wrapper.imp.Fatalf("expected panic with %v, got %v", expected, m.panicked)
	}
}

// WrapCalculatorDivideReturns provides type-safe access to Divide return values.
type WrapCalculatorDivideReturns struct {
	R1 int
	R2 error
}

// GetReturns returns type-safe return values for Divide.
func (m *WrapCalculatorDivideMethod) GetReturns() *WrapCalculatorDivideReturns {
	m.wrapper.imp.Helper()
	m.WaitForResponse()

	if m.panicked != nil {
		m.wrapper.imp.Fatalf("cannot get returns: method panicked with %v", m.panicked)
		return nil
	}

	return m.returned
}
