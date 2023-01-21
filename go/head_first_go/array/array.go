package main

import (
	"example.com/vcrini/x/go/array/file"
	"example.com/vcrini/x/go/array/media"
	"fmt"
)

func main() {

	// just to show how to use arrays
	var element [3]int
	element[0] = 2
	for i := 0; i < 3; i++ {
		fmt.Printf("il %d numero è  %d\n", i, element[i])
	}
  //again elements but dinamically declared, array is still static
	element2 := [3]float64{1.0, 2.0, 3.0}
	//calculating media
	sum := 0.0
	for _, v := range element2 {
		sum += v
	}
	n := float64(len(element2))

	fmt.Printf("la media di %v è %0.2f\n", element2, sum/n)
  // reading elements from file, array is dynamic
	fmt.Println("il contenuto del file è")
	elements := file.ReadFile()
	fmt.Printf("la media dei valori di %v è %0.2f\n", elements, media.Media(elements))
}
