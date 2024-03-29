package main

import (
	"fmt"
  "log"
)

func paintNeeded(width float64, height float64) (float64, error) {
	area := width * height
	if width < 0 {
		return 0, fmt.Errorf("width %.2f can't be negative: ", width)
	}
	if height < 0 {
		return 0, fmt.Errorf("height %.2f can't be negative: ", height)
	}
	return area / 10.0, nil
}
func main() {
	var width, height float64
	width = 4.2
	height = 3.0
	amount, err := paintNeeded(width, height)
	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Printf("area is %.2f\n", amount)
	}
	width = -1.0
	height = 3.5
	amount, err = paintNeeded(width, height)
	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Printf("area is %.2f\n", amount)
	}
}
