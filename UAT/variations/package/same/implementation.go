package whitebox

// ProcessWithOps uses both exported and unexported methods.
// This demonstrates that the generated mock can handle unexported interface methods
// when using whitebox testing.
func ProcessWithOps(o Ops, val int) int {
	internal := o.internalMethod(val)

	return o.PublicMethod(internal)
}
