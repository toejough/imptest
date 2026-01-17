package generics_test

import (
	"testing"

	"pgregory.net/rapid"

	generics "github.com/toejough/imptest/UAT/variations/signature/generics"
)

// TestProcessItem_Int_Property proves ProcessItem works correctly for any
// int value and any transformer function.
func TestProcessItem_Int_Property(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		// Generate random test data
		key := rapid.String().Draw(rt, "key")
		value := rapid.Int().Draw(rt, "value")
		multiplier := rapid.IntRange(1, 100).Draw(rt, "multiplier")

		// Create mock and wrapper
		repo, repoImp := MockRepository[int](t)
		wrapper := WrapProcessItem[int](t, generics.ProcessItem[int])

		// Define transformer
		transformer := func(v int) int { return v * multiplier }

		// Start the function
		call := wrapper.Method.Start(repo, key, transformer)

		// Property: Get is called with the key, Save is called with transformed value
		repoImp.Get.ExpectCalledWithExactly(key).InjectReturnValues(value, nil)
		repoImp.Save.ExpectCalledWithExactly(value * multiplier).InjectReturnValues(nil)

		// Property: Function returns nil on success
		call.ExpectReturnsEqual(nil)
	})
}

// TestProcessItem_String_Property proves ProcessItem works correctly for any
// string value and any transformer function.
func TestProcessItem_String_Property(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		// Generate random test data
		key := rapid.String().Draw(rt, "key")
		value := rapid.String().Draw(rt, "value")
		suffix := rapid.String().Draw(rt, "suffix")

		// Create mock and wrapper
		repo, repoImp := MockRepository[string](t)
		wrapper := WrapProcessItem[string](t, generics.ProcessItem[string])

		// Define transformer
		transformer := func(s string) string { return s + suffix }

		// Start the function
		call := wrapper.Method.Start(repo, key, transformer)

		// Property: Get is called with the key, Save is called with transformed value
		repoImp.Get.ExpectCalledWithExactly(key).InjectReturnValues(value, nil)
		repoImp.Save.ExpectCalledWithExactly(value + suffix).InjectReturnValues(nil)

		// Property: Function returns nil on success
		call.ExpectReturnsEqual(nil)
	})
}
