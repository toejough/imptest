package main

import "fmt"

//go:generate go run generate.go exampleInt
type exampleInt interface {
	print(string)
	add(int, int) int
	transform(float64) (error, []byte)
}

type otherInt interface {
	show(string)
	combine(int, int) int
	change(float64) (error, []byte)
}

type exampleStruct struct{}

func (exampleStruct) print(s string) {
	fmt.Println("exampleStruct.print called with:", s)
}

func (exampleStruct) add(a, b int) int {
	fmt.Println("exampleStruct.add called with:", a, b)
	return a + b
}

func (exampleStruct) transform(f float64) (error, []byte) {
	fmt.Println("exampleStruct.transform called with:", f)
	return nil, []byte{1, 2, 3}
}

func runExample(e exampleInt) {
	e.print("test")
	fmt.Println("add result:", e.add(2, 3))
	err, data := e.transform(1.23)
	fmt.Println("transform result:", err, data)
}

func main() {
	// This is a placeholder for the main function.
	fmt.Println("Hello, World!")
	runExample(exampleStruct{})
}
