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

func main() {
	// This is a placeholder for the main function.
	fmt.Println("Hello, World!")
}
