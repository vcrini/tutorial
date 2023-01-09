package main

import (
	"fmt"
	"reflect"
)

func main() {
	var myInt int
  var myIntPointer *int
	fmt.Println(reflect.TypeOf(myInt))
	fmt.Println(reflect.TypeOf(myIntPointer))
	fmt.Println(&myInt)
	fmt.Println(&myIntPointer)
  myIntPointer=&myInt
	fmt.Println(&myInt)
	fmt.Println(&myIntPointer)
	var myFloat float64
	fmt.Println(reflect.TypeOf(myFloat))
	var myBool bool
	fmt.Println(reflect.TypeOf(myBool))

}
