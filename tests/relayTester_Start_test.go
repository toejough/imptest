package imptest_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/toejough/protest/imptest"
	"pgregory.net/rapid"
)

func TestStartRunsFUTInGoroutine(t *testing.T) {
	t.Parallel()

	// Given test needs
	tester := imptest.NewRelayTester(t)
	// Given inputs
	lockchan := make(chan struct{})
	waitchan := make(chan struct{})
	wait := func() {
		<-lockchan
	}

	// release the lock at the end of the test
	defer close(lockchan)

	// When the func is run
	go func() {
		tester.Start(wait)
		close(waitchan)
	}()

	// Then the return from waitchan should be immediate
	select {
	case <-waitchan:
	case <-time.After(time.Second):
		t.Error("waitchan never closed, indicating function was run synchronously instead of in a goroutine.")
	}
}

func testStartPanicsWithNonFunction(rapidT *rapid.T) {
	// Given testing needs
	mockedt := newMockedTestingT()
	tester := imptest.NewRelayTester(mockedt)

	// Given FUT

	gen := generateRandomTypeZeroValueNonFunc()
	argFunc := gen.Draw(rapidT, "argFunc")
	// TODO: use rapid.Custom to generate arbitrary non-function types from reflect.
	// https://pkg.go.dev/pgregory.net/rapid#Custom
	// https://pkg.go.dev/reflect#Type
	// https://pkg.go.dev/reflect#New

	mockedt.Wrap(func() {
		// When the func is run with something that isn't a function
		defer expectPanicWith(mockedt, "must pass a function")
		// TODO: capture goroutine panics and re-escalate at wait for shutdown?
		//   feels like a lot of work to capture the backtrace & everything? is it? might be gratifying
		//   to get that right, but also feels low priority.
		// TODO: test for nil functions? If we do that then we need a generator that does more
		//  than just return zero functions... seems low priority.
		tester.Start(argFunc)
	})

	// Then the test is marked as passed
	if mockedt.Failed() {
		rapidT.Fatalf(
			"The test should've passed. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}

func generateRandomTypeNonFunc(rapidT *rapid.T) reflect.Type {
	generators := listTypeGeneratorsNonFunc(rapidT)
	index := rapid.IntRange(0, len(generators)-1).Draw(rapidT, "index")

	return generators[index]()
}

func listTypeGeneratorsNonFunc(rapidT *rapid.T) []func() reflect.Type {
	return []func() reflect.Type{
		func() reflect.Type { var v bool; return reflect.TypeOf(v) },
		func() reflect.Type { var v byte; return reflect.TypeOf(v) },
		func() reflect.Type { var v float32; return reflect.TypeOf(v) },
		func() reflect.Type { var v float64; return reflect.TypeOf(v) },
		func() reflect.Type { var v int; return reflect.TypeOf(v) },
		func() reflect.Type { var v int8; return reflect.TypeOf(v) },
		func() reflect.Type { var v int16; return reflect.TypeOf(v) },
		func() reflect.Type { var v int32; return reflect.TypeOf(v) },
		func() reflect.Type { var v int64; return reflect.TypeOf(v) },
		func() reflect.Type { var v rune; return reflect.TypeOf(v) },
		func() reflect.Type { var v string; return reflect.TypeOf(v) },
		func() reflect.Type { var v uint; return reflect.TypeOf(v) },
		func() reflect.Type { var v uint8; return reflect.TypeOf(v) },
		func() reflect.Type { var v uint16; return reflect.TypeOf(v) },
		func() reflect.Type { var v uint32; return reflect.TypeOf(v) },
		func() reflect.Type { var v uint64; return reflect.TypeOf(v) },
		func() reflect.Type { var v uintptr; return reflect.TypeOf(v) },
		func() reflect.Type { return reflect.TypeOf(complex(float32(0), float32(0))) },
		func() reflect.Type { return reflect.TypeOf(complex(float64(0), float64(0))) },
		func() reflect.Type {
			return reflect.ArrayOf(rapid.IntRange(0, 1000).Draw(rapidT, "arrayLen"), generateRandomType(rapidT))
		},
		func() reflect.Type {
			return reflect.ChanOf(
				rapid.SampledFrom([]reflect.ChanDir{reflect.RecvDir, reflect.SendDir, reflect.BothDir}).Draw(rapidT, "chanDir"),
				generateRandomType(rapidT),
			)
		},
		func() reflect.Type { return reflect.MapOf(generateRandomType(rapidT), generateRandomType(rapidT)) },
		func() reflect.Type { return reflect.PointerTo(generateRandomType(rapidT)) },
		func() reflect.Type { return reflect.SliceOf(generateRandomType(rapidT)) },
	}
}

func listTypeGenerators(rapidT *rapid.T) []func() reflect.Type {
	generators := listTypeGeneratorsNonFunc(rapidT)
	generators = append(generators,
		func() reflect.Type {
			ins := generateSliceOfRandomTypes(rapidT)
			outs := generateSliceOfRandomTypes(rapidT)

			return reflect.FuncOf(ins, outs, decideIfVariadic(rapidT, ins))
		},
	)

	return generators
}

func generateRandomType(rapidT *rapid.T) reflect.Type {
	generators := listTypeGenerators(rapidT)

	index := rapid.IntRange(0, len(generators)-1).Draw(rapidT, "index")

	return generators[index]()
}

func decideIfVariadic(rapidT *rapid.T, ins []reflect.Type) bool {
	if len(ins) == 0 {
		return false
	}

	if ins[len(ins)-1].Kind() != reflect.Slice {
		return false
	}

	return rapid.Bool().Draw(rapidT, "isVariadic")
}

func generateSliceOfRandomTypes(rapidT *rapid.T) []reflect.Type {
	s := []reflect.Type{}
	for i := 0; i < rapid.IntRange(0, 10).Draw(rapidT, "sliceLen"); i++ {
		s = append(s, generateRandomType(rapidT))
	}

	return s
}

func generateRandomTypeZeroValueNonFunc() *rapid.Generator[any] {
	return rapid.Custom(func(rapidT *rapid.T) any {
		return reflect.New(generateRandomTypeNonFunc(rapidT)).Elem().Interface()
	})
}

func TestStartPanicsWithNonFunction(t *testing.T) {
	t.Parallel()
	rapid.Check(t, testStartPanicsWithNonFunction)
}

func TestStartPanicsWithTooFewArgs(t *testing.T) {
	t.Parallel()

	// Given testing needs
	mockedt := newMockedTestingT()
	tester := imptest.NewRelayTester(mockedt)

	// Given FUT
	argFunc := func(_, _, _ int) {}

	mockedt.Wrap(func() {
		// When the func is run with the wrong number of args
		defer expectPanicWith(mockedt, "Too few args")
		tester.Start(argFunc)
	})

	// Then the test is marked as passed
	if mockedt.Failed() {
		t.Fatalf(
			"The test should've passed. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}

func TestStartPanicsWithTooManyArgs(t *testing.T) {
	t.Parallel()

	// Given testing needs
	mockedt := newMockedTestingT()
	tester := imptest.NewRelayTester(mockedt)

	// Given FUT
	argFunc := func(_ int) {}

	mockedt.Wrap(func() {
		// When the func is run with the wrong number of args
		defer expectPanicWith(mockedt, "Too many args")
		tester.Start(argFunc, 1, 2, 3)
	})

	// Then the test is marked as passed
	if mockedt.Failed() {
		t.Fatalf(
			"The test should've passed. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}

func TestStartPanicsWithWrongArgTypes(t *testing.T) {
	t.Parallel()

	// Given testing needs
	mockedt := newMockedTestingT()
	tester := imptest.NewRelayTester(mockedt)

	// Given FUT
	argFunc := func(_ int) {}

	mockedt.Wrap(func() {
		// When the func is run with the wrong number of args
		defer expectPanicWith(mockedt, "Wrong arg type")
		tester.Start(argFunc, "1")
	})

	// Then the test is marked as passed
	if mockedt.Failed() {
		t.Fatalf(
			"The test should've passed. Instead the failure was: %s",
			mockedt.Failure(),
		)
	}
}
