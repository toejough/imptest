For Direction Determines Role

* I liked the "what we do" column. Add that back with "We wrap it" to Target row, and "We mock it" to the Dependency row

Generator Command

* //go:generate impgen --[target|dependency] (package-alias.)[interface|struct|function-type|function]

API

* For the unordered case, I think it will be simpler if we shift from "within(timeout)" to just "Eventually"
* GetArgs, GetPanic, GetReturns should all follow the same semantics as the "Expect" methods for both ordered and
  unordered cases - in the ordered case, they will expect the very next interaction to be the one they match to (a call,
  panic, or return, respectively, to or from their instance) and fail the test if they don't match.  In the unordered case, they will wait for the matchinginteraction, queueing up any interactions that come in before it. In both cases, they'll wait for
  the next interaction as long as necessary (no timeout).

Design Decisions

* go doesn't allow slices or maps with properties - you can't have both `instance.GetArgs().Name` and `instance.GetArgs()[0]`. Instead let's just use indexes as names, like `instance.GetArgs().A1` or `GetReturns().R1`
