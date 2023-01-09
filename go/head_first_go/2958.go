package main

import "fmt"

func main() {
	number := 4
	double(&number)
	fmt.Printf("out from function number is %d\n", number)

}
func double(number *int){
	*number *= 2
	fmt.Printf("in function number is %d\n", *number)
}
