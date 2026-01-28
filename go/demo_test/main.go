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

func ReturnMap() map[string]name {
	return map[string]name{"O": {0, 0}, "A": {1, 2}}
}

func ReturnLiteralSlice() []string {
	return []string{"dungeons", "&", "dragons"}
}

func ReturnArray() [2]string {
	return [2]string{"dangerous", "dragons"}
}

func ReturnStruct() name {
	x := name{1, 2}
	return x
}
