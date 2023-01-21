package main

import (
	"fmt"
	"math"
)

func main() {
	fmt.Printf("max is %f.2\n", maximum(1.0, 34, 5, 1000))
	fmt.Printf("max is %f.2\n", maximum(5.0, 4, 5, 10))
}
func maximum(numbers ...float64) float64 {
	max := math.Inf(-1)
	for _, number := range numbers {
		if number > max {
			max = number
		}
	}
	return max
}
