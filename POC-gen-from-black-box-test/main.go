package main

import (
	"fmt"

	"github.com/toejough/imptest/POC-gen-from-black-box-test/run"
)

type exampleStruct struct{}

func (exampleStruct) Print(s string) {
	fmt.Println("exampleStruct.print called with:", s)
}

func (exampleStruct) Add(a, b int) int {
	fmt.Println("exampleStruct.add called with:", a, b)
	return a + b
}

func (exampleStruct) Transform(f float64) (error, []byte) {
	fmt.Println("exampleStruct.transform called with:", f)
	return nil, []byte{1, 2, 3}
}

func main() {
	fmt.Println("Hello, World!")
	run.RunExample(exampleStruct{})
}
