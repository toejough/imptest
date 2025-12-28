the example for passing exact data in is incorrect - we pass data in through callable wapper "Start" functions and
callback "invoke" functions right now. "ExpectArgsAre" is another way we receive exactly.

I also want to implement matchers for panics, to unify that with the return value matchers.

I don't think blocking/timeouts need to be expressly discussed in the outcome dimensions. 

Handling concurrency should be called out somewhere - what are the conditions under which we use "Within"?

Actually, I don't even think we can do channels - there's no way to mock those and intercept the calls without replacing
the channel itself and risking all kinds of edge cases.

What if we completely revamped the API to focus on the taxonomy of interfaces vs funcs, target business logic vs
dependencies, strict ordering vs unordered, and exact values vs matchers?

```go
// package-alias scenarios
// single-word: "time" -> time 
// final-segment: "github.com/toejough/imptest" -> imptest
// obscured: "github.com/toejough/imptest" -> mud (package is _at_ the path, but when resolved, its name is "mud")
// aliased: nick "github.com/toejough/imptest" -> nick (package is _at_ the path, but when resolved, its name is "mud", but we alias it to "nick")

// type naming scenarios
// unqualified: name: the named thing must be defined in this package, though not necessarily in the same file. 
// qualified: package-alias.name: the named thing must be defined in the indicated package.

// generator commands always need to use the package-alias as it is used in the code in the same file as the command.
// generated code goes into the same package, and should use the same type names and package aliases as the source code.
// any packages the generated code needs to import, independently from the source code, should be aliased with as many
// leading _ as necessary to avoid conflict with the source code's imports.

//go:generate impgen --[target|depencency] (package-alias.)[struct|interface|function|function-type]

// Imp: coordinates calls and expectations
imp := NewImp(t)
// Target: wraps a callable (function/method)
runTarget := NewRunTarget(imp, callable)
// Struct Target: wraps a struct/interface
runStructTarget:= NewRunStructTarget(imp, structInstance) // or interfaceInstance
// Dependency: 
serviceDependency:= NewServiceDependency(imp)
// Struct Dependency: 
serviceStructDependency:= NewServiceStructDependency(imp)

// That's it. You can't wrap channels or shared state.

// Target API
instance := RunTarget.CallWith(args) // 0+ args, compile-time typesafe
// Target Struct API
instance = RunStructTarget.Func1.CallWith(args) // 0+ args, compile-time typesafe
// Target Instance API
// by default instances are strictly ordered. Each call will expect the very next interaction is for it to validate, and
// pass or fail instantly.
instance = instance.Within(timeout) // an unordered instance is returned from a Within(), and behaves
// differently: it continuously pulls interactions from imp's channel as they happen, enqueuing any misses for other calls,
// and returning only when either the timeout is reached (failing the test) or the expected interaction has been matched. This allows for testing asynchronous behavior.
// The following is the API for both.
instance.ExpectReturnsEqual(values) // 0+ values, compile-time typesafe
instance.ExpectReturnsMatch(matchers) // 0+ matchers, # values are compile-time checked, types are runtime checked
instance.ExpectPanicEquals(value) // 1 value, any type
instance.ExpectPanicMatches(matcher) // 1 matcher
// you can always get the exact values, in a typesafe way:
instance.GetReturns().Name // fails the test if there's no return, with the same ordered/unordered behavior as above
instance.GetPanic() // fails the test if there's no panic, with the same ordered/unordered behavior as above

// Dependency API 
instance := ServiceDependency.ExpectCalledWithExactly(args) //0+ args, compile-time typesafe
instance = ServiceDependency.ExpectCalledWithMatches(matchers) //0+ matchers, # values are compile-time checked, types are runtime checked
instance = ServiceDependency.Within(timeout).ExpectCalledWithExactly(args) //0+ args, compile-time typesafe
instance = ServiceDependency.Within(timeout).ExpectCalledWithMatches(matchers) //0+ matchers, # values are compile-time checked, types are runtime checked
// Dependency Struct API
instance := ServiceStructDependency.Func1.ExpectCalledWithExactly(args) //0+ args, compile-time typesafe
instance := ServiceStructDependency.Func1.ExpectCalledWithMatches(matchers) //0+ matchers, # values are compile-time checked, types are runtime checked
instance := ServiceStructDependency.Func1.Within(timeout).ExpectCalledWithExactly(args) //0+ args, compile-time typesafe
instance := ServiceStructDependency.Func1.Within(timeout).ExpectCalledWithMatches(matchers) //0+ matchers, # values are compile-time checked, types are runtime checked
// you can always get the exact args, in a typesafe way:
instance.GetArgs().Name
// Dependency Instance API
instance.InjectReturnValues(values) // 0+ values, compile-time typesafe
instance.InjectPanicValue(value) // 1 value, any type

// Imp API
// for maximum flexibility, you can always get the next interaction yourself:
interaction := imp.NextInteraction()
```
