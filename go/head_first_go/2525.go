package main

import (
	"fmt"
)
func painNeeded(width float64, height float64) {
  area := width*height
	fmt.Printf("%.2f\n", area/10.0)
}
func main() {
	var width, height float64
	width = 4.2
	height = 3.0
  painNeeded(width,height)
	width = 5.2
	height = 3.5
  painNeeded(width,height)
}
