package main

import (
	"fmt"
)

func main() {

	// just to show how to use arrays
	var element [3]int
	element[0] = 2
	for i := 0; i < 3; i++ {
		fmt.Printf("il %d numero è  %d\n", i, element[i])
	}
	element2 := [3]int{1, 2, 3}
	//calculating media
	sum := 0
	const nOfElements int = 3
	for _, v := range element2 {
		sum += v
	}
	media := float64(sum / nOfElements)
	fmt.Printf(" la media è %2.2f\n", media)

}
