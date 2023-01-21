package main

import "fmt"

func main() {
 myFunc(1) 
 myFunc(1,2) 

}

func myFunc(numbers ...int) {
  fmt.Println(numbers)
}
