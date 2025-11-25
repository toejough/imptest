package run

// exampleInt is an interface for demonstration.
type ExampleInt interface {
	Print(string)
	Add(int, int) int
	Transform(float64) (error, []byte)
}

// RunExample calls the methods of ExampleInt.
func RunExample(e ExampleInt) {
	e.Print("test")
	println("add result:", e.Add(2, 3))
	err, data := e.Transform(1.23)
	println("transform result:", err, data)
}
