package imptest_test

//
// import (
// 	"fmt"
// 	"runtime"
// 	"sync"
// )
//
// // MockedTestingT.
// func newMockedTestingT() *mockedTestingT { return &mockedTestingT{failure: ""} }
//
// type mockedTestingT struct{ failure string }
//
// func (mt *mockedTestingT) Fatalf(message string, args ...any) {
// 	mt.failure = fmt.Sprintf(message, args...)
//
// 	// kill off the current goroutine
// 	runtime.Goexit()
// }
// func (mt *mockedTestingT) Helper()                   {}
// func (mt *mockedTestingT) Failed() bool              { return mt.failure != "" }
// func (mt *mockedTestingT) Failure() string           { return mt.failure }
// func (mt *mockedTestingT) Error(...any)              {}
// func (mt *mockedTestingT) Errorf(s string, a ...any) { mt.Fatalf(s, a...) }
// func (mt *mockedTestingT) Fail()                     {}
// func (mt *mockedTestingT) FailNow()                  {}
// func (mt *mockedTestingT) Fatal(...any)              {}
// func (mt *mockedTestingT) Log(...any)                {}
// func (mt *mockedTestingT) Logf(string, ...any)       {}
// func (mt *mockedTestingT) Name() string              { return "" }
// func (mt *mockedTestingT) Skip(...any)               {}
// func (mt *mockedTestingT) SkipNow()                  {}
// func (mt *mockedTestingT) Skipf(string, ...any)      {}
//
// // Wrap wraps the test calls such that the mocked testing library can catch
// // when a fatal error occurs and correctly return callflow to the test.
// // Without this, either a call to Fatalf would have to _not_ exit the goroutine,
// // which breaks the expectations of anyone using the standard testing library,
// // or it would literally exit the test, without having informed the standard test library
// // about it.
// // We could pass the failure on to the standard test library, but that's not the point -
// // the point of this library is to be able to capture whether those calls are made
// // as expected, and then often _that_ means the test _passes_.
// func (mt *mockedTestingT) Wrap(wrapped func()) {
// 	waitgroup := &sync.WaitGroup{}
// 	waitgroup.Add(1)
//
// 	go func() {
// 		defer waitgroup.Done()
// 		wrapped()
// 	}()
// 	waitgroup.Wait()
// }
