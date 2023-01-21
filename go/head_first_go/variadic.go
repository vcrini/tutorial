package main

import "fmt"

func main() {
 myFunc(1) 
 myFunc(1,2) 
 elements:=[]int{1,2,3}
 myFunc(elements...)

}

func myFunc(numbers ...int) {
  fmt.Println(numbers)
}
