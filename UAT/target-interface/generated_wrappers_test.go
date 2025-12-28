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
	wrapper.Subtract = &WrapBasicCalculatorSubtractMethod{wrapper: wrapper}
	wrapper.Divide = &WrapBasicCalculatorDivideMethod{wrapper: wrapper}
	return wrapper
}

// WrapBasicCalculatorWrapper provides a fluent API for calling and verifying BasicCalculator methods.
type WrapBasicCalculatorWrapper struct {
	imp      *imptest.Imp
	instance *BasicCalculator
	Add      *WrapBasicCalculatorAddMethod
	Subtract *WrapBasicCalculatorSubtractMethod
	Divide   *WrapBasicCalculatorDivideMethod
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
func (m *WrapBasicCalculatorAddMethod) Start(a, b int) *WrapBasicCalculatorAddMethod {
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

		result := m.wrapper.instance.Add(a, b)
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

// ExpectReturnsMatch verifies return values match the given matchers.
func (m *WrapBasicCalculatorAddMethod) ExpectReturnsMatch(matchers ...any) {
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

// WrapBasicCalculatorAddReturns provides type-safe access to Add return values.
type WrapBasicCalculatorAddReturns struct {
	R1 int
}

// GetReturns returns type-safe return values for Add.
func (m *WrapBasicCalculatorAddMethod) GetReturns() *WrapBasicCalculatorAddReturns {
	m.wrapper.imp.Helper()
	m.WaitForResponse()

	if m.panicked != nil {
		m.wrapper.imp.Fatalf("cannot get returns: method panicked with %v", m.panicked)
		return nil
	}

	return m.returned
}

// WrapBasicCalculatorSubtractMethod wraps the Subtract method.
type WrapBasicCalculatorSubtractMethod struct {
	wrapper    *WrapBasicCalculatorWrapper
	returnChan chan WrapBasicCalculatorSubtractReturns
	panicChan  chan any
	returned   *WrapBasicCalculatorSubtractReturns
	panicked   any
}

// Start begins execution of the Subtract method in a goroutine.
func (m *WrapBasicCalculatorSubtractMethod) Start(a, b int) *WrapBasicCalculatorSubtractMethod {
	m.returnChan = make(chan WrapBasicCalculatorSubtractReturns, 1)
	m.panicChan = make(chan any, 1)
	m.returned = nil
	m.panicked = nil

	go func() {
		defer func() {
			if r := recover(); r != nil {
				m.panicChan <- r
			}
		}()

		result := m.wrapper.instance.Subtract(a, b)
		m.returnChan <- WrapBasicCalculatorSubtractReturns{R1: result}
	}()

	return m
}

// WaitForResponse blocks until the method completes (return or panic).
func (m *WrapBasicCalculatorSubtractMethod) WaitForResponse() {
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
func (m *WrapBasicCalculatorSubtractMethod) ExpectReturnsEqual(expected int) {
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

// ExpectReturnsMatch verifies return values match the given matchers.
func (m *WrapBasicCalculatorSubtractMethod) ExpectReturnsMatch(matchers ...any) {
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

// WrapBasicCalculatorSubtractReturns provides type-safe access to Subtract return values.
type WrapBasicCalculatorSubtractReturns struct {
	R1 int
}

// GetReturns returns type-safe return values for Subtract.
func (m *WrapBasicCalculatorSubtractMethod) GetReturns() *WrapBasicCalculatorSubtractReturns {
	m.wrapper.imp.Helper()
	m.WaitForResponse()

	if m.panicked != nil {
		m.wrapper.imp.Fatalf("cannot get returns: method panicked with %v", m.panicked)
		return nil
	}

	return m.returned
}

// WrapBasicCalculatorDivideMethod wraps the Divide method.
type WrapBasicCalculatorDivideMethod struct {
	wrapper    *WrapBasicCalculatorWrapper
	returnChan chan WrapBasicCalculatorDivideReturns
	panicChan  chan any
	returned   *WrapBasicCalculatorDivideReturns
	panicked   any
}

// Start begins execution of the Divide method in a goroutine.
func (m *WrapBasicCalculatorDivideMethod) Start(a, b int) *WrapBasicCalculatorDivideMethod {
	m.returnChan = make(chan WrapBasicCalculatorDivideReturns, 1)
	m.panicChan = make(chan any, 1)
	m.returned = nil
	m.panicked = nil

	go func() {
		defer func() {
			if r := recover(); r != nil {
				m.panicChan <- r
			}
		}()

		r1, r2 := m.wrapper.instance.Divide(a, b)
		m.returnChan <- WrapBasicCalculatorDivideReturns{R1: r1, R2: r2}
	}()

	return m
}

// WaitForResponse blocks until the method completes (return or panic).
func (m *WrapBasicCalculatorDivideMethod) WaitForResponse() {
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
func (m *WrapBasicCalculatorDivideMethod) ExpectReturnsEqual(expectedR1 int, expectedR2 error) {
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

// ExpectReturnsMatch verifies return values match the given matchers.
func (m *WrapBasicCalculatorDivideMethod) ExpectReturnsMatch(matchers ...any) {
	m.wrapper.imp.Helper()
	m.WaitForResponse()

	if m.panicked != nil {
		m.wrapper.imp.Fatalf("expected method to return, but it panicked with: %v", m.panicked)
		return
	}

	if len(matchers) != 2 {
		m.wrapper.imp.Fatalf("expected 2 matchers, got %d", len(matchers))
		return
	}

	matcher0, ok0 := matchers[0].(imptest.Matcher)
	matcher1, ok1 := matchers[1].(imptest.Matcher)
	if !ok0 {
		m.wrapper.imp.Fatalf("argument 0 is not a Matcher")
		return
	}
	if !ok1 {
		m.wrapper.imp.Fatalf("argument 1 is not a Matcher")
		return
	}

	success, err := matcher0.Match(m.returned.R1)
	if err != nil {
		m.wrapper.imp.Fatalf("matcher 0 error: %v", err)
		return
	}
	if !success {
		m.wrapper.imp.Fatalf("return value 0: %s", matcher0.FailureMessage(m.returned.R1))
		return
	}

	success, err = matcher1.Match(m.returned.R2)
	if err != nil {
		m.wrapper.imp.Fatalf("matcher 1 error: %v", err)
		return
	}
	if !success {
		m.wrapper.imp.Fatalf("return value 1: %s", matcher1.FailureMessage(m.returned.R2))
	}
}

// ExpectPanicEquals verifies the method panicked with an exact value.
func (m *WrapBasicCalculatorDivideMethod) ExpectPanicEquals(expected any) {
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

// ExpectPanicMatches verifies the panic value matches the given matcher.
func (m *WrapBasicCalculatorDivideMethod) ExpectPanicMatches(matcher imptest.Matcher) {
	m.wrapper.imp.Helper()
	m.WaitForResponse()

	if m.returned != nil {
		m.wrapper.imp.Fatalf("expected method to panic, but it returned normally")
		return
	}

	success, err := matcher.Match(m.panicked)
	if err != nil {
		m.wrapper.imp.Fatalf("panic matcher error: %v", err)
		return
	}
	if !success {
		m.wrapper.imp.Fatalf("panic value: %s", matcher.FailureMessage(m.panicked))
	}
}

// WrapBasicCalculatorDivideReturns provides type-safe access to Divide return values.
type WrapBasicCalculatorDivideReturns struct {
	R1 int
	R2 error
}

// GetReturns returns type-safe return values for Divide.
func (m *WrapBasicCalculatorDivideMethod) GetReturns() *WrapBasicCalculatorDivideReturns {
	m.wrapper.imp.Helper()
	m.WaitForResponse()

	if m.panicked != nil {
		m.wrapper.imp.Fatalf("cannot get returns: method panicked with %v", m.panicked)
		return nil
	}

	return m.returned
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
func (m *WrapCalculatorAddMethod) Start(a, b int) *WrapCalculatorAddMethod {
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

		result := m.wrapper.instance.Add(a, b)
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

// ExpectReturnsMatch verifies return values match the given matchers.
func (m *WrapCalculatorAddMethod) ExpectReturnsMatch(matchers ...any) {
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

// WrapCalculatorAddReturns provides type-safe access to Add return values.
type WrapCalculatorAddReturns struct {
	R1 int
}

// GetReturns returns type-safe return values for Add.
func (m *WrapCalculatorAddMethod) GetReturns() *WrapCalculatorAddReturns {
	m.wrapper.imp.Helper()
	m.WaitForResponse()

	if m.panicked != nil {
		m.wrapper.imp.Fatalf("cannot get returns: method panicked with %v", m.panicked)
		return nil
	}

	return m.returned
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
func (m *WrapCalculatorSubtractMethod) Start(a, b int) *WrapCalculatorSubtractMethod {
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

		result := m.wrapper.instance.Subtract(a, b)
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

// ExpectReturnsEqual verifies the method returned exact values.
func (m *WrapCalculatorSubtractMethod) ExpectReturnsEqual(expected int) {
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

// GetReturns returns type-safe return values for Subtract.
func (m *WrapCalculatorSubtractMethod) GetReturns() *WrapCalculatorSubtractReturns {
	m.wrapper.imp.Helper()
	m.WaitForResponse()

	if m.panicked != nil {
		m.wrapper.imp.Fatalf("cannot get returns: method panicked with %v", m.panicked)
		return nil
	}

	return m.returned
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
func (m *WrapCalculatorDivideMethod) Start(a, b int) *WrapCalculatorDivideMethod {
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

		r1, r2 := m.wrapper.instance.Divide(a, b)
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

// ExpectReturnsMatch verifies return values match the given matchers.
func (m *WrapCalculatorDivideMethod) ExpectReturnsMatch(matchers ...any) {
	m.wrapper.imp.Helper()
	m.WaitForResponse()

	if m.panicked != nil {
		m.wrapper.imp.Fatalf("expected method to return, but it panicked with: %v", m.panicked)
		return
	}

	if len(matchers) != 2 {
		m.wrapper.imp.Fatalf("expected 2 matchers, got %d", len(matchers))
		return
	}

	matcher0, ok0 := matchers[0].(imptest.Matcher)
	matcher1, ok1 := matchers[1].(imptest.Matcher)
	if !ok0 {
		m.wrapper.imp.Fatalf("argument 0 is not a Matcher")
		return
	}
	if !ok1 {
		m.wrapper.imp.Fatalf("argument 1 is not a Matcher")
		return
	}

	success, err := matcher0.Match(m.returned.R1)
	if err != nil {
		m.wrapper.imp.Fatalf("matcher 0 error: %v", err)
		return
	}
	if !success {
		m.wrapper.imp.Fatalf("return value 0: %s", matcher0.FailureMessage(m.returned.R1))
		return
	}

	success, err = matcher1.Match(m.returned.R2)
	if err != nil {
		m.wrapper.imp.Fatalf("matcher 1 error: %v", err)
		return
	}
	if !success {
		m.wrapper.imp.Fatalf("return value 1: %s", matcher1.FailureMessage(m.returned.R2))
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

// ExpectPanicMatches verifies the panic value matches the given matcher.
func (m *WrapCalculatorDivideMethod) ExpectPanicMatches(matcher imptest.Matcher) {
	m.wrapper.imp.Helper()
	m.WaitForResponse()

	if m.returned != nil {
		m.wrapper.imp.Fatalf("expected method to panic, but it returned normally")
		return
	}

	success, err := matcher.Match(m.panicked)
	if err != nil {
		m.wrapper.imp.Fatalf("panic matcher error: %v", err)
		return
	}
	if !success {
		m.wrapper.imp.Fatalf("panic value: %s", matcher.FailureMessage(m.panicked))
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
