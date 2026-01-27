// Package demo_test is for testing
package demo_test

import (
	"fmt"
)

type name struct {
	X int
	Y int
}

func main() {
	fmt.Printf("%d", Sum(1, 1))
}

func Sum(x int, y int) int {
	return x + y
}

func ReturnStruct() name {
	x := name{1, 2}
	return x
}
