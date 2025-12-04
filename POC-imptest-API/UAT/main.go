package main

import (
	"fmt"

	"github.com/toejough/imptest/POC-imptest-API/UAT/run"
)

type exampleStruct struct{}

func (exampleStruct) Print(s string) {
	fmt.Println("exampleStruct.print called with:", s)
}

func (exampleStruct) Add(a, b int) int {
	fmt.Println("exampleStruct.add called with:", a, b)
	return a + b
}

func (exampleStruct) Format(i int) string {
	fmt.Println("exampleStruct.transform called with:", i)
	return fmt.Sprintf("Number: %d", i)
}

func main() {
	fmt.Println("Hello, World!")
	run.PrintSum(1, 2, exampleStruct{})
}
