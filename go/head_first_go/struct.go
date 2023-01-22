package main

import "fmt"

func main() {
	type details struct {
		name    string
		surname string
		age     int
	}
	var e details
	e.name = "Valerio"
	e.surname = "Crini"
	e.age = 48
	fmt.Printf("person is %s %s with age %d\n", e.name, e.surname, e.age)
	var a details
	a.name = "Adele"
	a.surname = "Crini"
	a.age = 8
	fmt.Printf("person is %s %s with age %d\n", a.name, a.surname, a.age)
}
