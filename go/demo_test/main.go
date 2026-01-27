package demo_test

import (
	"fmt"
)

func main() {
	fmt.Printf("%d", Sum(1, 1))
}

func Sum(x int, y int) int {
	return x + y
}
