package main

import "fmt"

type Liters float64
type Gallons float64

func main() {
	var l Liters = 10.0
	l.sayHi()

}

func (m Liters) sayHi() {
	fmt.Println("Ciao da ", m)
}
