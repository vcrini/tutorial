package main

import "fmt"

func main() {

  // just to show how to use arrays
	var element [3]int
	element[0] = 2
	for i := 0; i < 3; i++ {
		fmt.Printf("il %d numero è  %d\n", i ,element[i])
	}
  element2:=[3]int{1,2,3}
  //reading from file
  for _,v := range element2{
    fmt.Printf("%d\n",v)
    
  }


}
