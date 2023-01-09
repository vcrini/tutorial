package main

import (
	"fmt"
)

func main() {
	var width, height, area float64
	width = 4.2
	height = 3.0
	area = width * height
	fmt.Printf("%.2f\n", area/10.0)
	width = 5.2
	height = 3.5
	area = width * height
	fmt.Printf("%.2f\n", area/10.0)
}
