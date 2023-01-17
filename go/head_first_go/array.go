package main

// just to show how to use arrays
import "fmt"

func main() {

	var element [3]int
	element[0] = 2
	for i := 0; i < 3; i++ {
		fmt.Printf("il %d numero è  %d\n", i ,element[i])
	}

}
