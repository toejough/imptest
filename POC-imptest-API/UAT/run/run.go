package run

// exampleInt is an interface for demonstration.
type IntOps interface {
	Add(int, int) int
	Format(int) string
	Print(string)
}

// PrintSum calculates the sum of two integers using the provided ExampleInt dependency,
// formats the result, and prints it using the dependency's methods.
func PrintSum(a, b int, deps IntOps) (int, int, string) {
	sum := deps.Add(a, b)
	formatted := deps.Format(sum)
	deps.Print(formatted)
	return a, b, formatted
}
